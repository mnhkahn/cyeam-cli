package cli

import (
	"encoding/json"

	"github.com/mnhkahn/cyeam-cli/internal/mail"
	"github.com/mnhkahn/cyeam-cli/internal/output"
	"github.com/spf13/cobra"
)

func newMailLocalCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "local",
		Short: "Read mail already downloaded by Apple Mail",
	}
	var limit int
	list := &cobra.Command{
		Use:   "list",
		Short: "List locally cached Apple Mail inbox messages",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cache, err := mail.OpenAppleMail()
			if err != nil {
				return err
			}
			defer cache.Close()
			messages, err := mail.ListAppleMail(cache, limit)
			if err != nil {
				return err
			}
			body, err := json.Marshal(map[string]any{"count": len(messages), "messages": messages})
			if err != nil {
				return err
			}
			return output.WriteJSON(cmd.OutOrStdout(), body)
		},
	}
	list.Flags().IntVar(&limit, "limit", 20, "max number of messages to list")
	read := &cobra.Command{
		Use:   "read <id>",
		Short: "Read a locally cached Apple Mail message by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cache, err := mail.OpenAppleMail()
			if err != nil {
				return err
			}
			defer cache.Close()
			message, err := mail.ReadAppleMail(cache, args[0])
			if err != nil {
				return err
			}
			body, err := json.Marshal(message)
			if err != nil {
				return err
			}
			return output.WriteJSON(cmd.OutOrStdout(), body)
		},
	}
	cmd.AddCommand(list, read)
	return cmd
}
