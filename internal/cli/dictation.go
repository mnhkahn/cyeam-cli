package cli

import (
	"fmt"

	"github.com/mnhkahn/cyeam-cli/internal/homework"
	"github.com/spf13/cobra"
)

func newDictationCommand(service HomeworkService) *cobra.Command {
	var wordsFile string
	check := &cobra.Command{
		Use:   "check <image>",
		Short: "Check an English dictation image with the cyeam.com homework service",
		Long: `Upload one JPG, PNG, or WebP image and a UTF-8 word list. The server
recognizes the handwriting, aligns it with the reference words, and returns the
grade. Put one word or phrase per line; commas and semicolons also separate
items. The image must be at most 10 MiB and 40 megapixels.`,
		Example: `  cyeam dictation check dictation.jpg --words words.txt
  cyeam dictation check dictation.jpg --words words.txt --pretty`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if service == nil {
				return fmt.Errorf("homework service is not configured")
			}
			words, err := readHomeworkWords(wordsFile)
			if err != nil {
				return err
			}
			image, err := readHomeworkImage(args[0])
			if err != nil {
				return err
			}
			result, err := service.Review(cmd.Context(), homework.ReviewRequest{Kind: "dictation", Count: len(words), Words: words, Filename: args[0], Image: image})
			if err != nil {
				return err
			}
			return writeHomeworkResult(cmd, result)
		},
	}
	check.Flags().StringVar(&wordsFile, "words", "", "UTF-8 file containing reference words in dictation order")
	_ = check.MarkFlagRequired("words")
	cmd := &cobra.Command{Use: "dictation", Short: "Check English dictation with cyeam.com"}
	cmd.AddCommand(check, newHomeworkRegradeCommand("dictation", service))
	return cmd
}
