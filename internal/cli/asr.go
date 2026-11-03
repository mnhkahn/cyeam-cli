package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/mnhkahn/cyeam-cli/internal/asr"
	"github.com/mnhkahn/cyeam-cli/internal/output"
	"github.com/spf13/cobra"
)

const maxASRAudioBytes = 15 << 20

func newASRCommand(transcriber asr.Transcriber) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "asr <audio-file>",
		Short: "Transcribe an audio recording to text via cyeam.com",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filename := args[0]
			f, err := os.Open(filename)
			if err != nil {
				return fmt.Errorf("open audio file: %w", err)
			}
			defer f.Close()
			audio, err := io.ReadAll(io.LimitReader(f, maxASRAudioBytes+1))
			if err != nil {
				return fmt.Errorf("read audio file: %w", err)
			}
			if len(audio) > maxASRAudioBytes {
				return fmt.Errorf("audio file exceeds %d bytes", maxASRAudioBytes)
			}

			result, err := transcriber.Transcribe(cmd.Context(), filename, audio)
			if err != nil {
				return err
			}

			pretty, _ := cmd.Flags().GetBool("pretty")
			if pretty {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), result.Text)
				return err
			}

			body, err := json.Marshal(map[string]string{
				"file": filename,
				"text": result.Text,
			})
			if err != nil {
				return err
			}
			return output.WriteJSON(cmd.OutOrStdout(), body)
		},
	}
	cmd.AddCommand(newASRReviewCommand(transcriber))
	return cmd
}
