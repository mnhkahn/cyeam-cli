package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/mnhkahn/cyeam-cli/internal/auth"
	"github.com/mnhkahn/cyeam-cli/internal/onedrive"
	"github.com/mnhkahn/cyeam-cli/internal/output"
	"github.com/mnhkahn/cyeam-cli/internal/phonetic"
	"github.com/mnhkahn/cyeam-cli/internal/update"
	"github.com/mnhkahn/cyeam-cli/internal/version"
	"github.com/spf13/cobra"
)

type Service interface {
	DateHoliday(ctx context.Context, date string) ([]byte, error)
	RoadbookShare(ctx context.Context, body []byte) ([]byte, error)
	RoadbookGet(ctx context.Context, id string) ([]byte, error)
	MoGuwen(ctx context.Context, text string, aiCompose bool, font string) ([]byte, error)
	MoCharDetail(ctx context.Context, char string, font string) ([]byte, error)
	MoCharComposition(ctx context.Context, char string, font string) ([]byte, error)
	MoCharCompose(ctx context.Context, char string, font string) ([]byte, error)
	MoOCR(ctx context.Context, filename string, body []byte) ([]byte, error)
	NewsGet(ctx context.Context, date string) ([]byte, error)
}

type Dependencies struct {
	Service     Service
	Stdout      io.Writer
	Now         func() time.Time
	VersionInfo func() version.Info
	Updater     Updater
	OneDrive    func() OneDriveClient
	Phonetic    phonetic.Fetcher
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

var (
	pendingUpdate output.UpdateNotice
	pendingSkills output.SkillsNotice
	mu            sync.Mutex
)

func setupNotices(deps Dependencies) {
	if !update.ShouldCheck() {
		return
	}
	current := deps.VersionInfo().Version

	if latest, ok := update.CheckCached(current); ok {
		mu.Lock()
		pendingUpdate = output.UpdateNotice{
			Current: current,
			Latest:  latest,
			Message: fmt.Sprintf("cyeam %s available, current %s, run: cyeam update", latest, current),
			Command: "cyeam update",
		}
		mu.Unlock()
	}

	go func() {
		if latest, ok := update.RefreshCache(current); ok {
			mu.Lock()
			pendingUpdate = output.UpdateNotice{
				Current: current,
				Latest:  latest,
				Message: fmt.Sprintf("cyeam %s available, current %s, run: cyeam update", latest, current),
				Command: "cyeam update",
			}
			mu.Unlock()
		}
	}()
}

func consumeNotice() *output.Notice {
	mu.Lock()
	defer mu.Unlock()
	if pendingUpdate.Current == "" && pendingSkills.Current == "" {
		return nil
	}
	n := output.Notice{}
	if pendingUpdate.Current != "" {
		u := pendingUpdate
		n.Update = &u
		pendingUpdate = output.UpdateNotice{}
	}
	if pendingSkills.Current != "" {
		s := pendingSkills
		n.Skills = &s
		pendingSkills = output.SkillsNotice{}
	}
	return &n
}

func syncSkills(ctx context.Context, out io.Writer) bool {
	if _, err := exec.LookPath("npx"); err != nil {
		fmt.Fprintf(out, "skill sync skipped: npx not found (%v)\n", err)
		return false
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "npx", skillsAddArgs()...)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(output.String())
		if msg == "" {
			msg = err.Error()
		}
		fmt.Fprintf(out, "skill sync failed: %v: %s\n", err, msg)
		return false
	}
	fmt.Fprintf(out, "skills synced\n")
	return true
}

func skillsAddArgs() []string {
	return []string{"skills", "add", "mnhkahn/cyeam-cli", "-g", "-y"}
}

func setSkillsSynced(version string) {
	_ = update.SaveState(&update.State{
		LatestVersion: version,
		CheckedAt:     time.Now().Unix(),
	})
}

func newSkillsCommand(deps Dependencies) *cobra.Command {
	syncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync cyeam AI agent skills",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if syncSkills(cmd.Context(), deps.Stdout) {
				setSkillsSynced(deps.VersionInfo().Version)
			}
			return nil
		},
	}
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Manage AI agent skills",
	}
	cmd.AddCommand(syncCmd)
	return cmd
}

func WriteError(out io.Writer, err error) {
	enc := json.NewEncoder(out)
	_ = enc.Encode(output.ErrorEnvelope{
		OK:     false,
		Error:  output.ErrorInfo{Type: "error", Message: err.Error()},
		Notice: consumeNotice(),
	})
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
	if deps.Phonetic == nil {
		deps.Phonetic = phonetic.NewClient()
	}

	originalStdout := deps.Stdout
	var buf bytes.Buffer
	deps.Stdout = &buf

	var pretty bool

	root := &cobra.Command{
		Use:           "cyeam",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			setupNotices(deps)
			return nil
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			notice := consumeNotice()
			// `update` and `login` stream directly to stdout/stderr and must not
			// be wrapped in (or trailed by) the JSON envelope.
			if pretty || cmd.Name() == "update" || cmd.Name() == "login" {
				_, err := originalStdout.Write(buf.Bytes())
				return err
			}
			var env output.Envelope
			if cmd.Name() == "news" || (cmd.Parent() != nil && cmd.Parent().Name() == "news") {
				env = output.Envelope{OK: true, Data: json.RawMessage(buf.Bytes()), Notice: notice}
			} else {
				env = output.Envelope{OK: true, Data: string(buf.Bytes()), Notice: notice}
			}
			enc := json.NewEncoder(originalStdout)
			enc.SetEscapeHTML(false)
			return enc.Encode(env)
		},
	}
	root.SetOut(&buf)
	root.PersistentFlags().BoolVar(&pretty, "pretty", false, "human-readable output (omit JSON envelope)")

	helpFunc := root.HelpFunc()
	root.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		cmd.SetOut(originalStdout)
		helpFunc(cmd, args)
		cmd.SetOut(&buf)
	})

	root.AddCommand(newVersionCommand(deps))
	root.AddCommand(newUpdateCommand(deps))
	root.AddCommand(newDateCommand(deps))
	root.AddCommand(newMoCommand(deps))
	root.AddCommand(newRoadbookCommand(deps))
	root.AddCommand(newLoginCommand(deps))
	root.AddCommand(newLogoutCommand(deps))
	root.AddCommand(newWhoamiCommand(deps))
	root.AddCommand(newCnoteCommand(deps))
	root.AddCommand(newTVCommand(deps))
	root.AddCommand(newNewsCommand(deps))
	root.AddCommand(newSkillsCommand(deps))
	root.AddCommand(newPinyinCommand())
	root.AddCommand(newPhoneticCommand(deps.Phonetic))
	root.AddCommand(newAICommand())
	root.AddCommand(newPDFCommand())
	root.AddCommand(newImageCommand())
	root.AddCommand(newMailCommand())
	root.AddCommand(newTrelloCommand())
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
		Long:  "Update cyeam binary and sync AI agent skills via npx skills add",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if deps.Updater == nil {
				return fmt.Errorf("updater is required")
			}
			fmt.Fprintln(os.Stderr, "==> Checking for updates")
			current := deps.VersionInfo()
			result, err := deps.Updater.Update(cmd.Context(), current)
			if err != nil {
				return err
			}
			if result.Updated {
				fmt.Fprintf(os.Stderr, "==> Updated %s → %s\n", result.OldVersion, result.NewVersion)
			} else {
				fmt.Fprintf(os.Stderr, "==> Already up to date (%s)\n", result.NewVersion)
			}
			_, err = deps.Stdout.Write([]byte(result.String()))
			if err != nil {
				return err
			}

			if !result.Updated {
				fmt.Fprintln(os.Stderr, "==> Syncing skills")
				if ok := syncSkills(cmd.Context(), deps.Stdout); ok {
					setSkillsSynced(current.Version)
				}
				return nil
			}

			fmt.Fprintln(os.Stderr, "==> Syncing skills")
			if syncSkills(cmd.Context(), deps.Stdout) {
				setSkillsSynced(result.NewVersion)
			}
			return nil
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
			cmd.SetOut(deps.Stdout)
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
	cmd.AddCommand(newRoadbookCSVCommand(deps))
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

func newRoadbookCSVCommand(deps Dependencies) *cobra.Command {
	var title string
	cmd := &cobra.Command{
		Use:   "csv",
		Short: "Generate a roadbook from CSV text (stdin)",
		Args:  cobra.NoArgs,
		Long: `Read CSV from stdin, save to OneDrive, and generate a shareable roadbook link.
Requires login (cyeam login) first.

CSV format: 名称,地址,类型,日期,备注
 - 类型: 景点/餐饮/住宿/起点/其他
 - 日期: Day1, Day2... 或 5.1, 5.2...

Example:
  printf "%s\n" "故宫博物院,北京市东城区景山前街4号,景点,Day1,上午" | cyeam roadbook csv`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if deps.Service == nil {
				return fmt.Errorf("service is required")
			}
			csvData, err := io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return err
			}
			if len(bytes.TrimSpace(csvData)) == 0 {
				return fmt.Errorf("no CSV data provided (stdin is empty)")
			}
			items := parseCSVToItems(csvData)
			if len(items) == 0 {
				return fmt.Errorf("no valid items found in CSV")
			}

			// Build roadbook payload
			payload := map[string]interface{}{
				"title": title,
				"items": items,
			}
			payloadJSON, err := json.Marshal(payload)
			if err != nil {
				return err
			}

			// Generate fileId from sha256 hash (matches web's saveToOneDrive)
			h := sha256.Sum256(payloadJSON)
			fileID := "p_" + hex.EncodeToString(h[:])[:14]

			// Save to OneDrive 路书 folder
			oc := oneDriveClient(deps)
			filename := fileID + ".json"
			if err := oc.WriteFile(cmd.Context(), "路书", filename, "application/json", payloadJSON); err != nil {
				return fmt.Errorf("save to OneDrive: %w", err)
			}

			// Share via API to get short link
			sharePayload, err := json.Marshal(map[string]interface{}{"data": payload})
			if err != nil {
				return err
			}
			resp, err := deps.Service.RoadbookShare(cmd.Context(), sharePayload)
			if err != nil {
				return err
			}
			return output.WriteJSON(deps.Stdout, resp)
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "roadbook title")
	return cmd
}

type csvItem struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Type    string `json:"type"`
	Day     string `json:"day"`
	Note    string `json:"note"`
}

var (
	dayNoteDayRegex  = regexp.MustCompile(`^(?i)(Day\d+)\s*`)
	dayNoteDateRegex = regexp.MustCompile(`^(\d{1,2}\.\d{1,2})\s*`)
)

func parseCSVToItems(data []byte) []csvItem {
	r := csv.NewReader(bytes.NewReader(data))
	r.LazyQuotes = true
	r.TrimLeadingSpace = true
	records, err := r.ReadAll()
	if err != nil {
		return nil
	}
	var items []csvItem
	for _, cols := range records {
		if len(cols) < 3 {
			continue
		}
		name := strings.TrimSpace(cols[0])
		address := strings.TrimSpace(cols[1])
		if name == "" || (name == "名称" && address == "地址") {
			continue
		}
		typ := strings.TrimSpace(cols[2])
		switch typ {
		case "景点", "餐饮", "住宿", "起点", "其他":
		default:
			typ = "其他"
		}

		day := "Day1"
		note := ""
		if len(cols) > 3 {
			dayPart := strings.TrimSpace(cols[3])
			if m := dayNoteDayRegex.FindStringSubmatch(dayPart); m != nil {
				day = m[1]
				note = strings.TrimSpace(dayPart[len(m[0]):])
			} else if m := dayNoteDateRegex.FindStringSubmatch(dayPart); m != nil {
				day = m[1]
				note = strings.TrimSpace(dayPart[len(m[0]):])
			} else {
				note = dayPart
			}
		}
		if len(cols) > 4 {
			extra := strings.TrimSpace(cols[4])
			if extra != "" {
				if note != "" {
					note += " "
				}
				note += extra
			}
		}

		items = append(items, csvItem{
			Name:    name,
			Address: address,
			Type:    typ,
			Day:     day,
			Note:    note,
		})
	}
	return items
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
	var font string
	cmd := &cobra.Command{
		Use:   "guwen <text>",
		Short: "Generate guwen glyph data",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if deps.Service == nil {
				return fmt.Errorf("service is required")
			}
			body, err := deps.Service.MoGuwen(cmd.Context(), args[0], aiCompose, font)
			if err != nil {
				return err
			}
			return output.WriteJSON(deps.Stdout, body)
		},
	}
	cmd.Flags().BoolVar(&aiCompose, "ai-compose", false, "enable AI composition for missing glyphs")
	cmd.Flags().StringVar(&font, "font", "行书", "font type: 行书, 楷书, 隶书 etc.")
	return cmd
}

func newMoCharCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "char",
		Short: "Mo character utilities",
	}
	{
		var font string
		subCmd := &cobra.Command{
			Use:   "detail <char>",
			Short: "Get glyph candidates",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				if deps.Service == nil {
					return fmt.Errorf("service is required")
				}
				body, err := deps.Service.MoCharDetail(cmd.Context(), args[0], font)
				if err != nil {
					return err
				}
				return output.WriteJSON(deps.Stdout, body)
			},
		}
		subCmd.Flags().StringVar(&font, "font", "行书", "font type: 行书, 楷书, 隶书 etc.")
		cmd.AddCommand(subCmd)
	}
	{
		var font string
		subCmd := &cobra.Command{
			Use:   "composition <char>",
			Short: "Get character composition",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				if deps.Service == nil {
					return fmt.Errorf("service is required")
				}
				body, err := deps.Service.MoCharComposition(cmd.Context(), args[0], font)
				if err != nil {
					return err
				}
				return output.WriteJSON(deps.Stdout, body)
			},
		}
		subCmd.Flags().StringVar(&font, "font", "行书", "font type: 行书, 楷书, 隶书 etc.")
		cmd.AddCommand(subCmd)
	}
	cmd.AddCommand(newMoCharComposeCommand(deps))
	return cmd
}

func newMoCharComposeCommand(deps Dependencies) *cobra.Command {
	var out string
	var font string
	cmd := &cobra.Command{
		Use:   "compose <char>",
		Short: "Compose a character image",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if deps.Service == nil {
				return fmt.Errorf("service is required")
			}
			if out == "" {
				return fmt.Errorf("--out is required")
			}
			body, err := deps.Service.MoCharCompose(cmd.Context(), args[0], font)
			if err != nil {
				return err
			}
			return output.WriteFile(out, body)
		},
	}
	cmd.Flags().StringVar(&out, "out", "", "output PNG path")
	cmd.Flags().StringVar(&font, "font", "行书", "font type: 行书, 楷书, 隶书 etc.")
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
			// Write the link/code straight to the real stdout (not the envelope
			// buffer) so a streaming caller sees them while polling continues.
			return auth.Login(cmd.Context(), os.Stdout, os.Stderr)
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

func newNewsCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "news",
		Short: "Geek news and AI news from cyeam.com",
	}
	cmd.AddCommand(newNewsListCommand(deps))
	cmd.AddCommand(newNewsGetCommand(deps))
	return cmd
}

func newNewsListCommand(deps Dependencies) *cobra.Command {
	var from, to string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List geek news by date range",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if deps.Service == nil {
				return fmt.Errorf("service is required")
			}
			f, t := from, to
			if f == "" {
				f = deps.Now().AddDate(0, 0, -7).Format(time.DateOnly)
			}
			if t == "" {
				t = deps.Now().Format(time.DateOnly)
			}
			body, err := deps.Service.NewsGet(cmd.Context(), f)
			if err != nil {
				return err
			}
			return renderNewsList(deps.Stdout, body)
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "start date (YYYY-MM-DD), defaults to 7 days ago")
	cmd.Flags().StringVar(&to, "to", "", "end date (YYYY-MM-DD), defaults to today")
	return cmd
}

func newNewsGetCommand(deps Dependencies) *cobra.Command {
	var date string
	var brief bool
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get full geek news content by date",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if deps.Service == nil {
				return fmt.Errorf("service is required")
			}
			if date == "" {
				return fmt.Errorf("--date is required")
			}
			if _, err := time.Parse(time.DateOnly, date); err != nil {
				return fmt.Errorf("invalid date %q, want YYYY-MM-DD", date)
			}
			body, err := deps.Service.NewsGet(cmd.Context(), date)
			if err != nil {
				return err
			}
			pretty, _ := cmd.Flags().GetBool("pretty")
			if pretty {
				return renderNewsDetailTable(deps.Stdout, body)
			}
			if brief {
				var err error
				body, err = compactNewsBody(body)
				if err != nil {
					return err
				}
			}
			_, err = deps.Stdout.Write(body)
			return err
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "news date (YYYY-MM-DD)")
	cmd.Flags().BoolVar(&brief, "brief", false, "trim news output for LLM processing")
	_ = cmd.MarkFlagRequired("date")
	return cmd
}

type newsAPIResponse struct {
	News   *newsGeekNews `json:"news,omitempty"`
	AINews *newsAINews   `json:"ai_news,omitempty"`
	Date   string        `json:"date"`
	Error  string        `json:"error,omitempty"`
}

type newsGeekNews struct {
	CreateTime int64      `json:"create_time"`
	News       []newsItem `json:"news"`
	Summary    string     `json:"summary"`
}

type newsAINews struct {
	CreateTime int64      `json:"create_time"`
	News       []newsItem `json:"news"`
}

type newsItem struct {
	Link        string `json:"link"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Image       string `json:"image"`
	CreateTime  int64  `json:"create_time"`
}

type compactNewsAPIResponse struct {
	News   *compactNewsGeekNews `json:"news,omitempty"`
	AINews *compactNewsAINews   `json:"ai_news,omitempty"`
	Date   string               `json:"date"`
	Error  string               `json:"error,omitempty"`
}

type compactNewsGeekNews struct {
	CreateTime int64             `json:"create_time,omitempty"`
	News       []compactNewsItem `json:"news"`
	Summary    string            `json:"summary,omitempty"`
}

type compactNewsAINews struct {
	CreateTime int64             `json:"create_time,omitempty"`
	News       []compactNewsItem `json:"news"`
}

type compactNewsItem struct {
	Title       string `json:"title"`
	Link        string `json:"link,omitempty"`
	Description string `json:"description,omitempty"`
	CreateTime  int64  `json:"create_time,omitempty"`
}

func compactNewsBody(body []byte) ([]byte, error) {
	var resp newsAPIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	out := compactNewsAPIResponse{
		Date:  resp.Date,
		Error: resp.Error,
	}
	if resp.News != nil {
		out.News = &compactNewsGeekNews{
			CreateTime: resp.News.CreateTime,
			Summary:    compactNewsText(resp.News.Summary, 300),
			News:       compactNewsItems(resp.News.News),
		}
	}
	if resp.AINews != nil {
		out.AINews = &compactNewsAINews{
			CreateTime: resp.AINews.CreateTime,
			News:       compactNewsItems(resp.AINews.News),
		}
	}

	return json.Marshal(out)
}

func compactNewsItems(items []newsItem) []compactNewsItem {
	out := make([]compactNewsItem, 0, len(items))
	for _, item := range items {
		out = append(out, compactNewsItem{
			Title:       item.Title,
			Link:        item.Link,
			Description: compactNewsText(item.Description, 200),
			CreateTime:  item.CreateTime,
		})
	}
	return out
}

func compactNewsText(s string, maxRunes int) string {
	s = regexp.MustCompile(`!\[[^\]]*\]\([^)]+\)`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`).ReplaceAllString(s, "$1")
	s = regexp.MustCompile(`https?://\S+`).ReplaceAllString(s, "")
	s = strings.Join(strings.Fields(s), " ")
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	if maxRunes <= 3 {
		return string(runes[:maxRunes])
	}
	return string(runes[:maxRunes-3]) + "..."
}

func renderNewsList(out io.Writer, body []byte) error {
	var resp newsAPIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}
	if resp.Error != "" {
		_, err := io.WriteString(out, "error: "+resp.Error+"\n")
		return err
	}

	if resp.News != nil && len(resp.News.News) > 0 {
		if _, err := io.WriteString(out, "技术动向 "+resp.Date+"\n"); err != nil {
			return err
		}
		if err := renderNewsItemTable(out, resp.News.News); err != nil {
			return err
		}
	}

	if resp.AINews != nil && len(resp.AINews.News) > 0 {
		aiDate := time.Unix(resp.AINews.CreateTime, 0).Format("2006-01-02")
		if resp.AINews.CreateTime == 0 {
			aiDate = resp.Date
		}
		if _, err := io.WriteString(out, "\nAI 资讯 "+aiDate+"\n"); err != nil {
			return err
		}
		if err := renderNewsItemTable(out, resp.AINews.News); err != nil {
			return err
		}
	}

	return nil
}

func renderNewsItemTable(out io.Writer, items []newsItem) error {
	t := cliTable{
		Headers: []tableCell{
			{text: "标题", visible: "标题"},
			{text: "图片", visible: "图片"},
			{text: "描述", visible: "描述"},
		},
		Color: true,
	}
	const (
		maxTitleWidth = 30
		maxDescWidth  = 40
	)
	for _, item := range items {
		title := truncateDisplayWidth(item.Title, maxTitleWidth)
		imageVisible := "无"
		imageText := "无"
		if item.Image != "" {
			imageVisible = "有图"
			imageText = terminalHyperlink(item.Image, "有图")
		}
		desc := truncateDisplayWidth(item.Description, maxDescWidth)
		titleCell := title
		titleVisible := title
		if item.Link != "" {
			titleCell = terminalHyperlink(item.Link, title)
		}
		t.Rows = append(t.Rows, []tableCell{
			{text: titleCell, visible: titleVisible},
			{text: imageText, visible: imageVisible},
			{text: desc, visible: desc},
		})
	}
	return renderTable(out, t)
}

func renderNewsDetail(out io.Writer, body []byte) error {
	_, err := out.Write(body)
	return err
}

func renderNewsDetailTable(out io.Writer, body []byte) error {
	var resp newsAPIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}
	if resp.Error != "" {
		_, err := io.WriteString(out, "error: "+resp.Error+"\n")
		return err
	}

	if resp.News != nil {
		fmt.Fprintf(out, "技术动向 %s\n", resp.Date)
		if resp.News.Summary != "" {
			fmt.Fprintf(out, "总结: %s\n\n", resp.News.Summary)
		}
		if err := renderNewsItemTable(out, resp.News.News); err != nil {
			return err
		}
		fmt.Fprintln(out)
	}

	if resp.AINews != nil && len(resp.AINews.News) > 0 {
		aiDate := time.Unix(resp.AINews.CreateTime, 0).Format("2006-01-02")
		if resp.AINews.CreateTime == 0 {
			aiDate = resp.Date
		}
		fmt.Fprintf(out, "AI 资讯 %s\n", aiDate)
		if err := renderNewsItemTable(out, resp.AINews.News); err != nil {
			return err
		}
	}

	return nil
}
