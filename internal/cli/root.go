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
	OneDrive    func() OneDriveClient
}

type Updater interface {
	Update(ctx context.Context, current version.Info) (update.Result, error)
}

type OneDriveClient interface {
	ListFolder(ctx context.Context, folderPath string) ([]onedrive.Item, error)
	ReadFileByID(ctx context.Context, itemID string) ([]byte, error)
	ReadFile(ctx context.Context, folderPath, filename string) ([]byte, error)
	WriteFile(ctx context.Context, folderPath, filename, contentType string, content []byte) error
	CreateShareLink(ctx context.Context, folderPath, filename string) (string, error)
	GetUserInfo(ctx context.Context) (onedrive.UserInfo, error)
}

func oneDriveClient(deps Dependencies) OneDriveClient {
	if deps.OneDrive != nil {
		return deps.OneDrive()
	}
	return onedrive.NewClient(auth.GetAccessToken)
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
	root.AddCommand(newTVCommand(deps))
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
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("unknown date command %q", args[0])
			}
			return cmd.Help()
		},
	}
	cmd.AddCommand(newDateHolidayCommand(deps))
	return cmd
}

func newDateHolidayCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "holiday [YYYY-MM-DD]",
		Short: "Get date holiday",
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
			body, err := deps.Service.DateHoliday(cmd.Context(), date)
			if err != nil {
				return err
			}
			return writeDateHoliday(deps.Stdout, body)
		},
	}
}

type dateHolidayView struct {
	Date      string `json:"date"`
	IsHoliday bool   `json:"is_holiday"`
	Name      string `json:"name"`
	Type      int    `json:"type"`
	Week      int    `json:"week"`
	Wage      int    `json:"wage"`
	Target    string `json:"target"`
}

func writeDateHoliday(out io.Writer, body []byte) error {
	var h dateHolidayView
	if err := json.Unmarshal(body, &h); err != nil {
		return err
	}
	lines := []string{
		"日期: " + h.Date,
		"星期: " + dateWeekName(h.Week),
		"状态: " + dateHolidayStatus(h),
	}
	if h.Name != "" && h.Type != 1 {
		lines = append(lines, "名称: "+h.Name)
	}
	if h.Wage > 1 {
		lines = append(lines, fmt.Sprintf("薪资倍数: %d", h.Wage))
	}
	if h.Target != "" {
		lines = append(lines, "目标假期: "+h.Target)
	}
	_, err := io.WriteString(out, strings.Join(lines, "\n")+"\n")
	return err
}

func dateHolidayStatus(h dateHolidayView) string {
	switch {
	case h.Type == 3:
		return "调休补班"
	case h.Type == 1:
		return "周末休息"
	case h.Type == 2 || h.IsHoliday:
		return "休息日"
	default:
		return "工作日"
	}
}

func dateWeekName(week int) string {
	switch week {
	case 1:
		return "周一"
	case 2:
		return "周二"
	case 3:
		return "周三"
	case 4:
		return "周四"
	case 5:
		return "周五"
	case 6:
		return "周六"
	case 7:
		return "周日"
	default:
		return fmt.Sprintf("%d", week)
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
			oc := oneDriveClient(deps)
			items, err := oc.ListFolder(cmd.Context(), "路书")
			if err != nil {
				return err
			}
			if len(items) == 0 {
				_, err := deps.Stdout.Write([]byte("No roadbooks found.\n"))
				return err
			}
			var results []roadbookListRow
			for _, item := range items {
				title := ""
				content, err := oc.ReadFileByID(cmd.Context(), item.ID)
				if err == nil {
					var raw string
					if json.Unmarshal(content, &raw) == nil {
						content = []byte(raw)
					}
					var payload struct {
						Title string `json:"title"`
					}
					if json.Unmarshal(content, &payload) == nil && payload.Title != "" {
						title = payload.Title
					}
				}
				name := strings.TrimSuffix(item.Name, ".json")
				results = append(results, roadbookListRow{
					Name:     name,
					Title:    title,
					Modified: item.LastModifiedDateTime[:10],
				})
			}

			return renderRoadbookListTable(deps.Stdout, results)
		},
	}
}

type roadbookListRow struct {
	Name     string
	Title    string
	Modified string
}

func renderRoadbookListTable(out io.Writer, rows []roadbookListRow) error {
	t := cliTable{
		Headers: []tableCell{
			{text: "标题", visible: "标题"},
			{text: "修改时间", visible: "修改时间"},
			{text: "链接", visible: "链接"},
		},
		Color: true,
	}
	for _, row := range rows {
		title := row.Title
		if title == "" {
			title = "-"
		}
		title = truncateDisplayWidth(title, 32)
		link := terminalHyperlink(roadbookURL(row.Name), "链接")
		t.Rows = append(t.Rows, []tableCell{
			{text: title, visible: title},
			{text: row.Modified, visible: row.Modified},
			{text: link, visible: "链接"},
		})
	}

	return renderTable(out, t)
}

func roadbookURL(id string) string {
	return "https://www.cyeam.com/tool/roadbook?id=" + id
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
			fmt.Fprint(deps.Stdout, loginStatusLine(token, time.Now()))
			oc := oneDriveClient(deps)
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

func loginStatusLine(token auth.TokenSet, now time.Time) string {
	expiry := time.Unix(token.Expiry, 0)
	refreshStatus := "auto-refresh unavailable; run `cyeam login` again to enable it"
	if token.RefreshToken != "" {
		refreshStatus = "auto-refresh enabled"
	}
	if now.After(expiry) {
		return fmt.Sprintf("Status: logged in (access token expired %s; %s)\n", expiry.Format("2006-01-02 15:04:05"), refreshStatus)
	}
	return fmt.Sprintf("Status: logged in (access token valid until %s; %s)\n", expiry.Format("2006-01-02 15:04:05"), refreshStatus)
}

func newCnoteCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cnote",
		Short: "CNote - cloud notes on OneDrive",
	}
	cmd.AddCommand(newCnoteListCommand(deps))
	cmd.AddCommand(newCnoteGetCommand(deps))
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
			oc := oneDriveClient(deps)
			items, err := oc.ListFolder(cmd.Context(), "Notes")
			if err != nil {
				return err
			}
			if len(items) == 0 {
				_, err := deps.Stdout.Write([]byte("No notes found.\n"))
				return err
			}
			var rows []cnoteListRow
			for _, item := range items {
				rows = append(rows, cnoteListRow{
					Name:     strings.TrimSuffix(item.Name, ".html"),
					Modified: item.LastModifiedDateTime[:10],
					WebURL:   item.WebURL,
				})
			}
			return renderCnoteListTable(deps.Stdout, rows)
		},
	}
}

type cnoteListRow struct {
	Name     string
	Modified string
	WebURL   string
}

func renderCnoteListTable(out io.Writer, rows []cnoteListRow) error {
	t := cliTable{
		Headers: []tableCell{
			{text: "文件名", visible: "文件名"},
			{text: "修改时间", visible: "修改时间"},
			{text: "链接", visible: "链接"},
		},
		Color: true,
	}
	for _, row := range rows {
		name := truncateDisplayWidth(row.Name, 32)
		link := "-"
		linkVisible := "-"
		if row.WebURL != "" {
			link = terminalHyperlink(row.WebURL, "打开")
			linkVisible = "打开"
		}
		t.Rows = append(t.Rows, []tableCell{
			{text: name, visible: name},
			{text: row.Modified, visible: row.Modified},
			{text: link, visible: linkVisible},
		})
	}
	return renderTable(out, t)
}

func newCnoteGetCommand(deps Dependencies) *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "get <title>",
		Short: "Read a note as terminal-friendly text",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch format {
			case "markdown", "text":
			default:
				return fmt.Errorf("unsupported format %q, want markdown or text", format)
			}

			filename := args[0] + ".html"
			content, err := oneDriveClient(deps).ReadFile(cmd.Context(), "Notes", filename)
			if err != nil {
				return fmt.Errorf("read note: %w", err)
			}

			_, err = io.WriteString(deps.Stdout, formatCnoteHTML(content, format)+"\n")
			return err
		},
	}
	cmd.Flags().StringVar(&format, "format", "markdown", "output format: markdown, text")
	return cmd
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
			oc := oneDriveClient(deps)
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

			oc := oneDriveClient(deps)
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
