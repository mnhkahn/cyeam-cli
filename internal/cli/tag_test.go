package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNextReleaseTag(t *testing.T) {
	for _, tt := range []struct{ tags, mode, previous, next string }{
		{"", "minor", "v0.0.0", "v0.1.0"},
		{"", "major", "v0.0.0", "v1.0.0"},
		{"v0.2.15", "minor", "v0.2.15", "v0.3.0"},
		{"v0.2.15", "major", "v0.2.15", "v1.0.0"},
		{"v1.9.9\nv1.10.2\nv1.10.11\nv1.10.3", "minor", "v1.10.11", "v1.11.0"},
		{"v2.0.0\nv1.99.99", "major", "v2.0.0", "v3.0.0"},
		{"v1.2.3\nv99.0.0-rc.1\nv99.0.0+build\nrelease\nv01.2.3\n2.0.0", "minor", "v1.2.3", "v1.3.0"},
		{"v999999999999999999999.2.3", "major", "v999999999999999999999.2.3", "v1000000000000000000000.0.0"},
	} {
		t.Run(tt.tags+tt.mode, func(t *testing.T) {
			previous, next := nextReleaseTag(tt.tags, tt.mode)
			if previous != tt.previous || next != tt.next {
				t.Fatalf("got %s -> %s; want %s -> %s", previous, next, tt.previous, tt.next)
			}
		})
	}
}

func tagTestGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func tagTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	tagTestGit(t, dir, "init")
	tagTestGit(t, dir, "config", "user.name", "Tag Test")
	tagTestGit(t, dir, "config", "user.email", "tag@example.com")
	tagTestGit(t, dir, "config", "commit.gpgSign", "false")
	tagTestGit(t, dir, "config", "tag.gpgSign", "false")
	tagTestGit(t, dir, "commit", "--allow-empty", "-m", "initial")
	return dir
}

func TestCreateReleaseTag(t *testing.T) {
	dir := tagTestRepo(t)
	tagTestGit(t, dir, "tag", "-a", "v1.2.3", "-m", "release")
	for _, dryRun := range []bool{true, false} {
		result, err := createReleaseTag(context.Background(), dir, "minor", dryRun)
		if err != nil {
			t.Fatal(err)
		}
		if result.Tag != "v1.3.0" || result.Previous != "v1.2.3" || result.DryRun != dryRun {
			t.Fatalf("unexpected result: %+v", result)
		}
		tags := tagTestGit(t, dir, "tag", "--list", "v1.3.0")
		if dryRun && tags != "" || !dryRun && tags != "v1.3.0" {
			t.Fatalf("unexpected tags for dry-run=%v: %s", dryRun, tags)
		}
		if !dryRun && tagTestGit(t, dir, "rev-parse", "v1.3.0^{commit}") != result.Commit {
			t.Fatal("tag does not point at resolved HEAD")
		}
	}
}

func TestReleaseTagRejectsDirtyTree(t *testing.T) {
	for _, kind := range []string{"untracked", "staged", "modified"} {
		t.Run(kind, func(t *testing.T) {
			dir := tagTestRepo(t)
			path := filepath.Join(dir, "file.txt")
			if err := os.WriteFile(path, []byte("initial"), 0600); err != nil {
				t.Fatal(err)
			}
			if kind != "untracked" {
				tagTestGit(t, dir, "add", ".")
			}
			if kind == "modified" {
				tagTestGit(t, dir, "commit", "-m", "file")
				if err := os.WriteFile(path, []byte("changed"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			_, err := createReleaseTag(context.Background(), dir, "major", false)
			if err == nil || !strings.Contains(err.Error(), "not clean") {
				t.Fatalf("expected dirty tree error, got %v", err)
			}
			if tags := tagTestGit(t, dir, "tag", "--list"); tags != "" {
				t.Fatalf("unexpected tags: %s", tags)
			}
		})
	}
}

func TestReleaseTagErrors(t *testing.T) {
	for _, mode := range []string{"minor", "patch"} {
		if _, err := createReleaseTag(context.Background(), t.TempDir(), mode, false); err == nil {
			t.Fatalf("expected error for %s outside a repository", mode)
		}
	}
	dir := t.TempDir()
	tagTestGit(t, dir, "init")
	if _, err := createReleaseTag(context.Background(), dir, "major", false); err == nil {
		t.Fatal("expected error for repository without commits")
	}
}

func TestTagCommand(t *testing.T) {
	t.Setenv("CYEAM_CLI_NO_UPDATE_NOTIFIER", "1")
	t.Chdir(tagTestRepo(t))
	var out bytes.Buffer
	root := NewRootCommand(Dependencies{Stdout: &out})
	root.SetArgs([]string{"tag", "major", "--dry-run"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		OK   bool             `json:"ok"`
		Data releaseTagResult `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if !envelope.OK || envelope.Data.Tag != "v1.0.0" || !envelope.Data.DryRun {
		t.Fatalf("unexpected output: %s", out.String())
	}
	for _, args := range [][]string{{"tag"}, {"tag", "minor", "major"}, {"tag", "patch"}} {
		root := NewRootCommand(Dependencies{})
		root.SetArgs(args)
		if err := root.Execute(); err == nil {
			t.Fatalf("expected error for %v", args)
		}
	}
}
