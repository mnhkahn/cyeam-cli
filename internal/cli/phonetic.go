package cli

import (
	"encoding/json"
	"fmt"

	"github.com/mnhkahn/cyeam-cli/internal/output"
	"github.com/mnhkahn/cyeam-cli/internal/phonetic"
	"github.com/spf13/cobra"
)

func newPhoneticCommand(fetcher phonetic.Fetcher) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "phonetic <word>",
		Short: "Get phonetic transcription for an English word",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			word := args[0]

			result, err := fetcher.Fetch(cmd.Context(), word)
			if err != nil {
				return err
			}

			pretty, _ := cmd.Flags().GetBool("pretty")
			if pretty {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), result.Format())
				return err
			}

			body, err := json.Marshal(result)
			if err != nil {
				return err
			}
			return output.WriteJSON(cmd.OutOrStdout(), body)
		},
	}
	return cmd
}
