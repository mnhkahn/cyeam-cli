package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mnhkahn/cyeam-cli/internal/asr"
	"github.com/mnhkahn/cyeam-cli/internal/output"
	"github.com/mnhkahn/cyeam-cli/internal/reading"
	"github.com/spf13/cobra"
)

func newASRReviewCommand(transcriber asr.Transcriber) *cobra.Command {
	var referenceFile, transcriptFile, audioFile, feedbackOut, confirmedGrade string
	cmd := &cobra.Command{
		Use: "review", Short: "Generate a deterministic English reading grade and word-practice feedback",
		Long: `Compare an English passage with an existing transcript or a recording transcribed by cyeam web.
Normalize common contractions and spoken equivalents; filter function words.
Only substitutions and matches affect the reference score. Missing/extra words remain evidence.
Grades: 优 >=80%, 良 >60% and <80%, 中 40–60% inclusive, 差 <40%.
Automatic grades are provisional. --confirmed-grade records a human listening decision.
Outputs a replayable JSON report; --pretty prints fixed feedback. Does not publish to Trello.
Use trello card review --reading-report <report.json> to publish the generated result unchanged.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if confirmedGrade != "" && confirmedGrade != "100" && confirmedGrade != "优" && confirmedGrade != "良" && confirmedGrade != "中" && confirmedGrade != "差" {
				return fmt.Errorf("--confirmed-grade must be 100, 优, 良, 中 or 差")
			}
			if feedbackOut != "" {
				for _, path := range []string{referenceFile, transcriptFile, audioFile} {
					if path == "" {
						continue
					}
					a, _ := filepath.Abs(feedbackOut)
					b, _ := filepath.Abs(path)
					x, e1 := os.Stat(feedbackOut)
					y, e2 := os.Stat(path)
					if a == b || (e1 == nil && e2 == nil && os.SameFile(x, y)) {
						return fmt.Errorf("--feedback-out must not overwrite an input")
					}
				}
			}
			ref, err := readCheckFile(referenceFile, 1<<20)
			if err != nil {
				return err
			}
			if err := reading.ValidateReference(string(ref)); err != nil {
				return err
			}
			var transcript string
			if transcriptFile != "" {
				data, err := readCheckFile(transcriptFile, 1<<20)
				if err != nil {
					return err
				}
				transcript = string(data)
			} else {
				audio, err := readCheckFile(audioFile, maxASRAudioBytes)
				if err != nil {
					return err
				}
				result, err := transcriber.Transcribe(cmd.Context(), audioFile, audio)
				if err != nil {
					return err
				}
				if result == nil {
					return fmt.Errorf("ASR returned no result")
				}
				transcript = result.Text
			}
			report, err := reading.Review(string(ref), transcript, reading.Options{ConfirmedGrade: confirmedGrade})
			if err != nil {
				return err
			}
			body, err := json.Marshal(report)
			if err != nil {
				return err
			}
			if feedbackOut != "" {
				if err := os.WriteFile(feedbackOut, []byte(report.Feedback), 0600); err != nil {
					return fmt.Errorf("save reading feedback: %w", err)
				}
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			if pretty {
				_, err = fmt.Fprint(cmd.OutOrStdout(), report.Feedback)
				return err
			}
			return output.WriteJSON(cmd.OutOrStdout(), body)
		},
	}
	cmd.Flags().StringVar(&referenceFile, "reference", "", "UTF-8 file containing the original English passage")
	cmd.Flags().StringVar(&transcriptFile, "transcript", "", "UTF-8 file containing an existing ASR transcript (offline comparison)")
	cmd.Flags().StringVar(&audioFile, "audio", "", "recording to transcribe via cyeam web before comparison")
	cmd.Flags().StringVar(&confirmedGrade, "confirmed-grade", "", "explicit human-confirmed grade: 100|优|良|中|差 (otherwise provisional)")
	cmd.Flags().StringVar(&feedbackOut, "feedback-out", "", "also save the generated fixed feedback to a UTF-8 file")
	cmd.MarkFlagRequired("reference")
	cmd.MarkFlagsMutuallyExclusive("audio", "transcript")
	cmd.MarkFlagsOneRequired("audio", "transcript")
	return cmd
}
