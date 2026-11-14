package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newASRReviewCommand(service HomeworkService) *cobra.Command {
	var referenceFile, transcriptFile, audioFile, feedbackOut string
	cmd := &cobra.Command{
		Use: "review", Short: "Generate an English reading review with cyeam.com",
		Long: `Compare an English passage with an existing transcript or transcribe a
recording through cyeam.com. Recognition, normalization, alignment, focus words,
grades, and feedback are all calculated by the server. The JSON response retains
the complete server input/report. --pretty prints the server-provided feedback.
Use trello card review --reading-report <report.json> to regrade on the server
and publish the result.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if service == nil {
				return fmt.Errorf("homework service is not configured")
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
			reference, err := readCheckFile(referenceFile, 1<<20)
			if err != nil {
				return err
			}
			if strings.TrimSpace(string(reference)) == "" || len(reference) > 30000 {
				return fmt.Errorf("reference must be nonempty and at most 30000 bytes")
			}
			var result json.RawMessage
			if transcriptFile != "" {
				transcript, err := readCheckFile(transcriptFile, 1<<20)
				if err != nil {
					return err
				}
				if len(transcript) > 60000 {
					return fmt.Errorf("transcript exceeds 60000 bytes")
				}
				input, _ := json.Marshal(map[string]string{"kind": "reading", "reference": string(reference), "transcript": string(transcript)})
				result, err = service.ReadingRegrade(cmd.Context(), input)
				if err != nil {
					return err
				}
			} else {
				audio, err := readCheckFile(audioFile, maxASRAudioBytes)
				if err != nil {
					return err
				}
				result, err = service.Reading(cmd.Context(), string(reference), audioFile, audio)
				if err != nil {
					return err
				}
			}
			if feedbackOut != "" {
				view, err := homeworkResultViewFromJSON(result)
				if err != nil {
					return err
				}
				if err := os.WriteFile(feedbackOut, []byte(view.Report.Feedback), 0600); err != nil {
					return fmt.Errorf("save reading feedback: %w", err)
				}
			}
			return writeHomeworkResult(cmd, result)
		},
	}
	cmd.Flags().StringVar(&referenceFile, "reference", "", "UTF-8 file containing the original English passage")
	cmd.Flags().StringVar(&transcriptFile, "transcript", "", "UTF-8 file containing an existing ASR transcript")
	cmd.Flags().StringVar(&audioFile, "audio", "", "recording to transcribe and review via cyeam.com")
	cmd.Flags().StringVar(&feedbackOut, "feedback-out", "", "also save the server-provided feedback to a UTF-8 file")
	_ = cmd.MarkFlagRequired("reference")
	cmd.MarkFlagsMutuallyExclusive("audio", "transcript")
	cmd.MarkFlagsOneRequired("audio", "transcript")
	return cmd
}
