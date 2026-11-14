package cli

import (
	"fmt"

	"github.com/mnhkahn/cyeam-cli/internal/homework"
	"github.com/spf13/cobra"
)

func newArithmeticCheckCommand(service HomeworkService) *cobra.Command {
	var expectedCount int
	cmd := &cobra.Command{
		Use:   "check <image>",
		Short: "Check photographed arithmetic with the cyeam.com homework service",
		Long: `Upload one JPG, PNG, or WebP image to cyeam.com. The server recognizes the
questions, calculates the results, and returns the grade. The image must be at
most 10 MiB and 40 megapixels. The expected question count is required and must
be between 1 and 100.`,
		Example: `  cyeam arithmetic check homework.jpg --expected-count 18
  cyeam arithmetic check homework.jpg --expected-count 18 --pretty`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if expectedCount < 1 || expectedCount > 100 {
				return fmt.Errorf("--expected-count must be between 1 and 100")
			}
			if service == nil {
				return fmt.Errorf("homework service is not configured")
			}
			image, err := readHomeworkImage(args[0])
			if err != nil {
				return err
			}
			result, err := service.Review(cmd.Context(), homework.ReviewRequest{Kind: "arithmetic", Count: expectedCount, Filename: args[0], Image: image})
			if err != nil {
				return err
			}
			return writeHomeworkResult(cmd, result)
		},
	}
	cmd.Flags().IntVar(&expectedCount, "expected-count", 0, "expected number of questions (1–100, required)")
	return cmd
}

func newHomeworkRegradeCommand(kind string, service HomeworkService) *cobra.Command {
	return &cobra.Command{
		Use:   "regrade <input.json>",
		Short: "Regrade corrected recognition text with the cyeam.com homework service",
		Long: `Read the input object from a previous check response, or a complete response
containing input/report, and send it to the server for grading. No grade is
calculated locally. Edit the recognized items in the JSON file before running.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if service == nil {
				return fmt.Errorf("homework service is not configured")
			}
			input, err := readHomeworkInput(args[0], kind)
			if err != nil {
				return err
			}
			result, err := service.Regrade(cmd.Context(), input)
			if err != nil {
				return err
			}
			return writeHomeworkResult(cmd, result)
		},
	}
}
