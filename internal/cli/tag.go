package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os/exec"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

var releaseTagPattern = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

func newTagCommand() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "tag <minor|major>",
		Short: "Create a Git release tag with a minor or major version bump",
		Long: `Create a local vMAJOR.MINOR.PATCH tag on HEAD in the current Git repository.
Uses the highest local stable release tag, or v0.0.0 if none exists.
minor: v1.2.3 -> v1.3.0; major: v1.2.3 -> v2.0.0.
Requires a clean working tree. Does not fetch or push tags.`,
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"minor", "major"},
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := createReleaseTag(cmd.Context(), "", args[0], dryRun)
			if err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			if pretty {
				action := "Created"
				if dryRun {
					action = "Would create"
				}
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s %s -> %s on %s\n", action, result.Previous, result.Tag, result.Commit)
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview the next version without creating a tag")
	return cmd
}

type releaseTagResult struct {
	Previous string `json:"previous"`
	Tag      string `json:"tag"`
	Commit   string `json:"commit"`
	DryRun   bool   `json:"dry_run"`
}

func createReleaseTag(ctx context.Context, dir, mode string, dryRun bool) (releaseTagResult, error) {
	var result releaseTagResult
	if mode != "minor" && mode != "major" {
		return result, fmt.Errorf("invalid upgrade mode %q: use minor or major", mode)
	}
	git := func(args ...string) (string, error) {
		cmd := exec.CommandContext(ctx, "git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
		}
		return strings.TrimSpace(string(out)), nil
	}
	commit, err := git("rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return result, err
	}
	status, err := git("status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return result, err
	}
	if status != "" {
		return result, fmt.Errorf("working tree is not clean; commit or stash changes before tagging")
	}
	tags, err := git("tag", "--list")
	if err != nil {
		return result, err
	}
	previous, next := nextReleaseTag(tags, mode)
	result = releaseTagResult{Previous: previous, Tag: next, Commit: commit, DryRun: dryRun}
	if !dryRun {
		// Pin the resolved commit and never overwrite an existing tag.
		if _, err := git("-c", "tag.gpgSign=false", "tag", "--", next, commit); err != nil {
			return result, err
		}
	}
	return result, nil
}

func nextReleaseTag(tags, mode string) (string, string) {
	latest := [3]*big.Int{new(big.Int), new(big.Int), new(big.Int)}
	previous := "v0.0.0"
	for _, tag := range strings.Fields(tags) {
		match := releaseTagPattern.FindStringSubmatch(tag)
		if match == nil {
			continue
		}
		var candidate [3]*big.Int
		for i := range candidate {
			candidate[i], _ = new(big.Int).SetString(match[i+1], 10)
		}
		for i := range candidate {
			cmp := candidate[i].Cmp(latest[i])
			if cmp > 0 {
				latest, previous = candidate, tag
			}
			if cmp != 0 {
				break
			}
		}
	}
	if mode == "major" {
		latest[0].Add(latest[0], big.NewInt(1))
		latest[1].SetInt64(0)
	} else {
		latest[1].Add(latest[1], big.NewInt(1))
	}
	return previous, fmt.Sprintf("v%s.%s.0", latest[0], latest[1])
}
