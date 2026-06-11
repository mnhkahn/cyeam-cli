package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mnhkahn/cyeam-cli/internal/auth"
	"github.com/mnhkahn/cyeam-cli/internal/onedrive"
	"github.com/mnhkahn/cyeam-cli/internal/output"
	"github.com/mnhkahn/cyeam-cli/internal/update"
	"github.com/mnhkahn/cyeam-cli/internal/version"
	"github.com/spf13/cobra"
)

type Service interface {
	AskArchitecture(ctx context.Context, query string, mode string, out io.Writer) error
	Search(ctx context.Context, query string) ([]byte, error)
	DateSlogan(ctx context.Context, date string) ([]byte, error)
	DateHoliday(ctx context.Context, date string) ([]byte, error)
	RoadbookShare(ctx context.Context, body []byte) ([]byte, error)
	RoadbookGet(ctx context.Context, id string) ([]byte, error)
	MoGuwen(ctx context.Context, text string, aiCompose bool) ([]byte, error)
	MoCharDetail(ctx context.Context, char string) ([]byte, error)
	MoCharComposition(ctx context.Context, char string) ([]byte, error)
	MoCharCompose(ctx context.Context, char string) ([]byte, error)
	MoOCR(ctx context.Context, filename string, body []byte) ([]byte, error)
}

type Dependencies struct {
	Service     Service
	Stdout      io.Writer
	Now         func() time.Time
	VersionInfo func() version.Info
	Updater     Updater
}

type Updater interface {
	Update(ctx context.Context, current version.Info) (update.Result, error)
}

func NewRootCommand(deps Dependencies) *cobra.Command {
	if deps.Stdout == nil {
		deps.Stdout = io.Discard
	}
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if deps.VersionInfo == nil {
		deps.VersionInfo = version.Current
	}

	root := &cobra.Command{
		Use:           "cyeam",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(newVersionCommand(deps))
	root.AddCommand(newUpdateCommand(deps))
	root.AddCommand(newAskCommand(deps))
	root.AddCommand(newDateCommand(deps))
	root.AddCommand(newMoCommand(deps))
	root.AddCommand(newRoadbookCommand(deps))
	root.AddCommand(newLoginCommand(deps))
	root.AddCommand(newLogoutCommand(deps))
	root.AddCommand(newWhoamiCommand(deps))
	root.AddCommand(newCnoteCommand(deps))
	return root
}

func newVersionCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := deps.Stdout.Write([]byte(deps.VersionInfo().String()))
			return err
		},
	}
}

func newUpdateCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update cyeam from GitHub Releases",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if deps.Updater == nil {
				return fmt.Errorf("updater is required")
			}
			result, err := deps.Updater.Update(cmd.Context(), deps.VersionInfo())
			if err != nil {
				return err
			}
			_, err = deps.Stdout.Write([]byte(result.String()))
			return err
		},
	}
}

func newAskCommand(deps Dependencies) *cobra.Command {
	var mode string
	cmd := &cobra.Command{
		Use:   "ask <question>",
		Short: "Ask the cyeam architecture assistant",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if deps.Service == nil {
				return fmt.Errorf("service is required")
			}
			switch mode {
			case "fast", "think", "expert":
			default:
				return fmt.Errorf("unsupported mode %q", mode)
			}
			return deps.Service.AskArchitecture(cmd.Context(), args[0], mode, deps.Stdout)
		},
	}
	cmd.Flags().StringVar(&mode, "mode", "fast", "architecture mode: fast, think, expert")
	cmd.AddCommand(newAskSearchCommand(deps))
	return cmd
}

func newAskSearchCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Search cyeam.com",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if deps.Service == nil {
				return fmt.Errorf("service is required")
			}
			body, err := deps.Service.Search(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return output.WriteJSON(deps.Stdout, body)
		},
	}
}

func newDateCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "date",
		Short: "Date utilities",
	}
	cmd.AddCommand(newDateLeafCommand(deps, "slogan"))
	cmd.AddCommand(newDateLeafCommand(deps, "holiday"))
	return cmd
}

func newDateLeafCommand(deps Dependencies, kind string) *cobra.Command {
	return &cobra.Command{
		Use:   kind + " [YYYY-MM-DD]",
		Short: "Get date " + kind,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if deps.Service == nil {
				return fmt.Errorf("service is required")
			}
			date := deps.Now().Format(time.DateOnly)
			if len(args) == 1 {
				if _, err := time.Parse(time.DateOnly, args[0]); err != nil {
					return fmt.Errorf("invalid date %q, want YYYY-MM-DD", args[0])
				}
				date = args[0]
			}
			var (
				body []byte
				err  error
			)
			if kind == "slogan" {
				body, err = deps.Service.DateSlogan(cmd.Context(), date)
			} else {
				body, err = deps.Service.DateHoliday(cmd.Context(), date)
			}
			if err != nil {
				return err
			}
			return output.WriteJSON(deps.Stdout, body)
		},
	}
}

func newRoadbookCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "roadbook",
		Short: "Roadbook sharing",
	}
	cmd.AddCommand(newRoadbookListCommand(deps))
	cmd.AddCommand(&cobra.Command{
		Use:   "share <roadbook.json>",
		Short: "Share a roadbook JSON file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if deps.Service == nil {
				return fmt.Errorf("service is required")
			}
			body, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}
			if len(body) == 0 {
				return fmt.Errorf("roadbook file is empty")
			}
			resp, err := deps.Service.RoadbookShare(cmd.Context(), body)
			if err != nil {
				return err
			}
			return output.WriteJSON(deps.Stdout, resp)
		},
	})
	cmd.AddCommand(newRoadbookGetCommand(deps))
	return cmd
}

func newRoadbookListCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List roadbooks from OneDrive",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			oc := onedrive.NewClient(auth.GetAccessToken)
			items, err := oc.ListFolder(cmd.Context(), "路书")
			if err != nil {
				return err
			}
			if len(items) == 0 {
				_, err := deps.Stdout.Write([]byte("No roadbooks found.\n"))
				return err
			}
			type roadbookMeta struct {
				Name    string
				Title   string
				Modified string
			}
			var results []roadbookMeta
			for _, item := range items {
				title := ""
				content, err := oc.ReadFile(cmd.Context(), "路书", item.Name)
				if err == nil {
					var payload struct {
						Title string `json:"title"`
					}
					if json.Unmarshal(content, &payload) == nil && payload.Title != "" {
						title = payload.Title
					}
				}
				name := strings.TrimSuffix(item.Name, ".json")
				results = append(results, roadbookMeta{
					Name:     name,
					Title:    title,
					Modified: item.LastModifiedDateTime[:10],
				})
			}
			fmt.Fprintf(deps.Stdout, "%-40s %-30s %s\n", "文件名", "标题", "修改时间")
			for _, r := range results {
				title := r.Title
				if title == "" {
					title = "-"
				}
				if len([]rune(title)) > 28 {
					title = string([]rune(title)[:25]) + "..."
				}
				fmt.Fprintf(deps.Stdout, "%-40s %-30s %s\n", r.Name, title, r.Modified)
			}
			return nil
		},
	}
}

func newRoadbookGetCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a shared roadbook",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := deps.Service.RoadbookGet(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return output.WriteJSON(deps.Stdout, resp)
		},
	}
}

func newMoCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mo",
		Short: "Mo calligraphy and OCR",
	}
	cmd.AddCommand(newMoGuwenCommand(deps))
	cmd.AddCommand(newMoCharCommand(deps))
	cmd.AddCommand(newMoOCRCommand(deps))
	return cmd
}

func newMoGuwenCommand(deps Dependencies) *cobra.Command {
	var aiCompose bool
	cmd := &cobra.Command{
		Use:   "guwen <text>",
		Short: "Generate xingshu guwen glyph data",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if deps.Service == nil {
				return fmt.Errorf("service is required")
			}
			body, err := deps.Service.MoGuwen(cmd.Context(), args[0], aiCompose)
			if err != nil {
				return err
			}
			return output.WriteJSON(deps.Stdout, body)
		},
	}
	cmd.Flags().BoolVar(&aiCompose, "ai-compose", false, "enable AI composition for missing glyphs")
	return cmd
}

func newMoCharCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "char",
		Short: "Mo character utilities",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "detail <char>",
		Short: "Get xingshu glyph candidates",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if deps.Service == nil {
				return fmt.Errorf("service is required")
			}
			body, err := deps.Service.MoCharDetail(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return output.WriteJSON(deps.Stdout, body)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "composition <char>",
		Short: "Get character composition",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if deps.Service == nil {
				return fmt.Errorf("service is required")
			}
			body, err := deps.Service.MoCharComposition(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return output.WriteJSON(deps.Stdout, body)
		},
	})
	cmd.AddCommand(newMoCharComposeCommand(deps))
	return cmd
}

func newMoCharComposeCommand(deps Dependencies) *cobra.Command {
	var out string
	cmd := &cobra.Command{
		Use:   "compose <char>",
		Short: "Compose a xingshu character image",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if deps.Service == nil {
				return fmt.Errorf("service is required")
			}
			if out == "" {
				return fmt.Errorf("--out is required")
			}
			body, err := deps.Service.MoCharCompose(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return output.WriteFile(out, body)
		},
	}
	cmd.Flags().StringVar(&out, "out", "", "output PNG path")
	return cmd
}

func newMoOCRCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "ocr <image>",
		Short: "OCR a Mo image",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if deps.Service == nil {
				return fmt.Errorf("service is required")
			}
			body, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}
			resp, err := deps.Service.MoOCR(cmd.Context(), args[0], body)
			if err != nil {
				return err
			}
			return output.WriteJSON(deps.Stdout, resp)
		},
	}
}

func newLoginCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Sign in with Microsoft account",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return auth.Login(cmd.Context(), deps.Stdout)
		},
	}
}

func newLogoutCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Sign out and clear stored credentials",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := auth.Logout(); err != nil {
				return err
			}
			_, err := deps.Stdout.Write([]byte("Logged out.\n"))
			return err
		},
	}
}

func newWhoamiCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show current login status and user info",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := auth.LoadToken()
			if err != nil {
				_, err := deps.Stdout.Write([]byte("Not logged in. Run `cyeam login` first.\n"))
				return err
			}
			expiry := time.Unix(token.Expiry, 0)
			if time.Now().After(expiry) {
				fmt.Fprintf(deps.Stdout, "Status: logged in (token expired %s)\n", expiry.Format("2006-01-02 15:04:05"))
			} else {
				fmt.Fprintf(deps.Stdout, "Status: logged in (token valid until %s)\n", expiry.Format("2006-01-02 15:04:05"))
			}
			oc := onedrive.NewClient(auth.GetAccessToken)
			user, err := oc.GetUserInfo(cmd.Context())
			if err != nil {
				fmt.Fprintf(deps.Stdout, "User info: unavailable (%v)\n", err)
				return nil
			}
			if user.DisplayName != "" {
				fmt.Fprintf(deps.Stdout, "Name: %s\n", user.DisplayName)
			}
			if user.Mail != "" {
				fmt.Fprintf(deps.Stdout, "Email: %s\n", user.Mail)
			}
			if user.UserPrincipalName != "" {
				fmt.Fprintf(deps.Stdout, "Account: %s\n", user.UserPrincipalName)
			}
			return nil
		},
	}
}

func newCnoteCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cnote",
		Short: "CNote - cloud notes on OneDrive",
	}
	cmd.AddCommand(newCnoteListCommand(deps))
	cmd.AddCommand(newCnoteNewCommand(deps))
	cmd.AddCommand(newCnoteAppendCommand(deps))
	return cmd
}

func newCnoteListCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List notes from OneDrive",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			oc := onedrive.NewClient(auth.GetAccessToken)
			items, err := oc.ListFolder(cmd.Context(), "Notes")
			if err != nil {
				return err
			}
			if len(items) == 0 {
				_, err := deps.Stdout.Write([]byte("No notes found.\n"))
				return err
			}
			fmt.Fprintf(deps.Stdout, "%-30s %s\n", "文件名", "修改时间")
			for _, item := range items {
				name := strings.TrimSuffix(item.Name, ".html")
				fmt.Fprintf(deps.Stdout, "%-30s %s\n", name, item.LastModifiedDateTime[:10])
			}
			return nil
		},
	}
}

func newCnoteNewCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "new <title>",
		Short: "Create a new note (content from stdin)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			title := args[0]
			content, err := io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return fmt.Errorf("read stdin: %w", err)
			}
			if len(bytes.TrimSpace(content)) == 0 {
				return fmt.Errorf("no content provided (stdin is empty)")
			}

			filename := title + ".html"
			oc := onedrive.NewClient(auth.GetAccessToken)
			return oc.WriteFile(cmd.Context(), "Notes", filename, "text/html", content)
		},
	}
}

func newCnoteAppendCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "append <title>",
		Short: "Append content to an existing note (content from stdin)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			title := args[0]
			filename := title + ".html"

			oc := onedrive.NewClient(auth.GetAccessToken)
			existing, err := oc.ReadFile(cmd.Context(), "Notes", filename)
			if err != nil {
				return fmt.Errorf("read note: %w", err)
			}

			appendContent, err := io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return fmt.Errorf("read stdin: %w", err)
			}
			if len(bytes.TrimSpace(appendContent)) == 0 {
				return fmt.Errorf("no content provided (stdin is empty)")
			}

			combined := append(existing, appendContent...)
			return oc.WriteFile(cmd.Context(), "Notes", filename, "text/html", combined)
		},
	}
}
