package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/mnhkahn/cyeam-cli/internal/mail"
	"github.com/mnhkahn/cyeam-cli/internal/output"
	"github.com/spf13/cobra"
)

func newMailCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mail",
		Short: "Read and send email over IMAP/SMTP",
		Long: `Read and send email for accounts configured in ~/.cyeam/mail.json.

Each account's app-specific password is read from the environment variable
named by its "password_env" field.`,
	}
	cmd.AddCommand(newMailListCommand())
	cmd.AddCommand(newMailReadCommand())
	cmd.AddCommand(newMailSendCommand())
	return cmd
}

func loadAccount(name string) (mail.Account, error) {
	cfg, err := mail.LoadConfig()
	if err != nil {
		return mail.Account{}, err
	}
	return cfg.FindAccount(name)
}

func newMailListCommand() *cobra.Command {
	var limit int
	var all bool
	cmd := &cobra.Command{
		Use:   "list <account>",
		Short: "List recent messages in an account's INBOX",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := mail.LoadConfig()
			if err != nil {
				return err
			}

			var accounts []mail.Account
			if all {
				accounts = cfg.Accounts
			} else {
				if len(args) == 0 {
					return fmt.Errorf("account is required unless --all is used")
				}
				acc, err := cfg.FindAccount(args[0])
				if err != nil {
					return err
				}
				accounts = []mail.Account{acc}
			}

			result := make([]map[string]any, 0, len(accounts))
			for _, acc := range accounts {
				cl, err := mail.Dial(acc)
				if err != nil {
					return err
				}
				msgs, err := cl.ListRecent(limit)
				cl.Close()
				if err != nil {
					return err
				}
				result = append(result, map[string]any{
					"account":  acc.Name,
					"count":    len(msgs),
					"messages": msgs,
				})
			}

			var body []byte
			if all {
				body, _ = json.Marshal(map[string]any{"results": result})
			} else {
				body, _ = json.Marshal(result[0])
			}
			return output.WriteJSON(cmd.OutOrStdout(), body)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "max number of messages to list")
	cmd.Flags().BoolVar(&all, "all", false, "list messages from all configured accounts")
	return cmd
}

func newMailReadCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "read <account> <uid>",
		Short: "Read a single message by UID",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			acc, err := loadAccount(args[0])
			if err != nil {
				return err
			}
			uid, err := strconv.ParseUint(args[1], 10, 32)
			if err != nil {
				return fmt.Errorf("invalid uid %q: %w", args[1], err)
			}
			cl, err := mail.Dial(acc)
			if err != nil {
				return err
			}
			defer cl.Close()

			raw, err := cl.FetchRaw(uint32(uid))
			if err != nil {
				return err
			}
			msg, err := mail.ParseMessage(strings.NewReader(string(raw)))
			if err != nil {
				return err
			}
			body, _ := json.Marshal(msg)
			return output.WriteJSON(cmd.OutOrStdout(), body)
		},
	}
}

func newMailSendCommand() *cobra.Command {
	var to, cc []string
	var subject, body, bodyFile string
	cmd := &cobra.Command{
		Use:   "send <account>",
		Short: "Send a message via SMTP",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			acc, err := loadAccount(args[0])
			if err != nil {
				return err
			}
			if len(to) == 0 {
				return fmt.Errorf("--to is required")
			}
			text := body
			if bodyFile != "" {
				data, err := os.ReadFile(bodyFile)
				if err != nil {
					return fmt.Errorf("read --body-file: %w", err)
				}
				text = string(data)
			}
			if err := mail.Send(acc, to, cc, subject, text); err != nil {
				return err
			}
			out, _ := json.Marshal(map[string]any{
				"account": acc.Name,
				"to":      to,
				"sent":    true,
			})
			return output.WriteJSON(cmd.OutOrStdout(), out)
		},
	}
	cmd.Flags().StringSliceVar(&to, "to", nil, "recipient address (repeatable)")
	cmd.Flags().StringSliceVar(&cc, "cc", nil, "cc address (repeatable)")
	cmd.Flags().StringVar(&subject, "subject", "", "message subject")
	cmd.Flags().StringVar(&body, "body", "", "message body text")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", "read body from a file instead of --body")
	return cmd
}
