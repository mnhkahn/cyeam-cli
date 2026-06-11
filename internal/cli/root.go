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
			oc := onedrive.NewClient(auth.GetAccessToken)
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
	table := [][]tableCell{{
		{text: "标题", visible: "标题"},
		{text: "修改时间", visible: "修改时间"},
		{text: "链接", visible: "链接"},
	}}
	for _, row := range rows {
		title := row.Title
		if title == "" {
			title = "-"
		}
		link := terminalHyperlink(roadbookURL(row.Name), "链接")
		table = append(table, []tableCell{
			{text: truncateDisplayWidth(title, 32), visible: truncateDisplayWidth(title, 32)},
			{text: row.Modified, visible: row.Modified},
			{text: link, visible: "链接"},
		})
	}

	widths := tableWidths(table)
	if _, err := fmt.Fprintln(out, tableBorder(widths)); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, tableRow(table[0], widths)); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, tableBorder(widths)); err != nil {
		return err
	}
	for _, row := range table[1:] {
		if _, err := fmt.Fprintln(out, tableRow(row, widths)); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(out, tableBorder(widths))
	return err
}

func roadbookURL(id string) string {
	return "https://www.cyeam.com/tool/roadbook?id=" + id
}

func terminalHyperlink(url string, label string) string {
	return "\033]8;;" + url + "\033\\" + label + "\033]8;;\033\\"
}

type tableCell struct {
	text    string
	visible string
}

func tableWidths(rows [][]tableCell) []int {
	if len(rows) == 0 {
		return nil
	}
	widths := make([]int, len(rows[0]))
	for _, row := range rows {
		for i, cell := range row {
			widths[i] = max(widths[i], displayWidth(cell.visible))
		}
	}
	return widths
}

func tableBorder(widths []int) string {
	var b strings.Builder
	for _, width := range widths {
		b.WriteByte('+')
		b.WriteString(strings.Repeat("-", width+2))
	}
	b.WriteString("+")
	return b.String()
}

func tableRow(row []tableCell, widths []int) string {
	var b strings.Builder
	for i, cell := range row {
		b.WriteString("| ")
		b.WriteString(cell.text)
		b.WriteString(strings.Repeat(" ", widths[i]-displayWidth(cell.visible)))
		b.WriteByte(' ')
	}
	b.WriteString("|")
	return b.String()
}

func truncateDisplayWidth(s string, maxWidth int) string {
	if displayWidth(s) <= maxWidth {
		return s
	}
	if maxWidth <= 3 {
		return takeDisplayWidth(s, maxWidth)
	}
	return takeDisplayWidth(s, maxWidth-3) + "..."
}

func takeDisplayWidth(s string, maxWidth int) string {
	var b strings.Builder
	width := 0
	for _, r := range s {
		next := runeWidth(r)
		if width+next > maxWidth {
			break
		}
		b.WriteRune(r)
		width += next
	}
	return b.String()
}

func displayWidth(s string) int {
	width := 0
	for _, r := range s {
		width += runeWidth(r)
	}
	return width
}

func runeWidth(r rune) int {
	if r == 0 || r < 32 || (r >= 0x7f && r < 0xa0) {
		return 0
	}
	if isWideRune(r) {
		return 2
	}
	return 1
}

func isWideRune(r rune) bool {
	return r >= 0x1100 && (r <= 0x115f ||
		r == 0x2329 || r == 0x232a ||
		(r >= 0x2e80 && r <= 0xa4cf && r != 0x303f) ||
		(r >= 0xac00 && r <= 0xd7a3) ||
		(r >= 0xf900 && r <= 0xfaff) ||
		(r >= 0xfe10 && r <= 0xfe19) ||
		(r >= 0xfe30 && r <= 0xfe6f) ||
		(r >= 0xff00 && r <= 0xff60) ||
		(r >= 0xffe0 && r <= 0xffe6))
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
