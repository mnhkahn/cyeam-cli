package cli

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mnhkahn/cyeam-cli/internal/auth"
	"github.com/mnhkahn/cyeam-cli/internal/output"
	"github.com/mnhkahn/cyeam-cli/internal/trello"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newTrelloCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "trello", Short: "Manage Trello boards, lists, and task cards"}
	cmd.AddCommand(
		newTrelloLoginCommand(),
		newTrelloStatusCommand(),
		newTrelloLogoutCommand(),
		newTrelloBoardsCommand(),
		newTrelloListsCommand(),
		newTrelloCardsCommand(),
		newTrelloStatusChangesCommand(),
		newTrelloHomeworkCommand(),
		newTrelloBoardCommand(),
		newTrelloListCommand(),
		newTrelloCardCommand(),
		newTrelloWebhookCommand(),
	)
	return cmd
}

func getTrelloClient() (*trello.Client, error) { return trello.NewDefault() }
func writeTrelloJSON(cmd *cobra.Command, data []byte) error {
	return output.WriteJSON(cmd.OutOrStdout(), data)
}

func newTrelloLoginCommand() *cobra.Command {
	var apiKey string
	cmd := &cobra.Command{Use: "login", Short: "Authorize Trello and store credentials securely", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		apiKey = strings.TrimSpace(apiKey)
		if apiKey == "" {
			apiKey = strings.TrimSpace(os.Getenv("TRELLO_API_KEY"))
		}
		if apiKey == "" {
			if stored, err := trello.LoadCredentials(); err == nil {
				apiKey = stored.APIKey
			}
		}
		if apiKey == "" {
			return fmt.Errorf("API key is required: cyeam trello login --key <api-key>")
		}

		authorizationURL := trello.AuthorizationURL(apiKey)
		fmt.Fprintf(os.Stdout, "Authorize cyeam in Trello:\n%s\n\n", authorizationURL)
		if err := auth.OpenBrowser(authorizationURL); err != nil {
			fmt.Fprintf(os.Stderr, "Could not open a browser automatically: %v\n", err)
		}
		token, err := readSecretLine(os.Stdin, os.Stdout, "Paste the token shown after authorization: ")
		if err != nil {
			return err
		}
		credentials := trello.Credentials{APIKey: apiKey, Token: strings.TrimSpace(token)}
		if credentials.Token == "" {
			return fmt.Errorf("token is required")
		}
		client := trello.New(credentials)
		member, err := client.Member(cmd.Context())
		if err != nil {
			return fmt.Errorf("validate Trello credentials: %w", err)
		}
		if err := trello.StoreCredentials(credentials); err != nil {
			return fmt.Errorf("store Trello credentials: %w", err)
		}
		var identity struct {
			FullName string `json:"fullName"`
			Username string `json:"username"`
		}
		_ = json.Unmarshal(member, &identity)
		if identity.Username != "" {
			fmt.Fprintf(os.Stdout, "Trello login successful: %s (@%s)\n", identity.FullName, identity.Username)
		} else {
			fmt.Fprintln(os.Stdout, "Trello login successful.")
		}
		if os.Getenv("TRELLO_API_KEY") != "" || os.Getenv("TRELLO_TOKEN") != "" {
			fmt.Fprintln(os.Stdout, "Note: unset TRELLO_API_KEY and TRELLO_TOKEN to use the newly stored credentials.")
		}
		return nil
	}}
	cmd.Flags().StringVar(&apiKey, "key", "", "Trello Power-Up API key")
	return cmd
}

func newTrelloStatusCommand() *cobra.Command {
	return &cobra.Command{Use: "status", Short: "Show Trello login status", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		credentials, source, err := trello.ResolveCredentials()
		if err != nil {
			data, _ := json.Marshal(map[string]any{"logged_in": false, "message": err.Error()})
			return writeTrelloJSON(cmd, data)
		}
		member, err := trello.New(credentials).Member(cmd.Context())
		if err != nil {
			return fmt.Errorf("validate stored Trello credentials: %w", err)
		}
		var identity any
		if err := json.Unmarshal(member, &identity); err != nil {
			return err
		}
		data, _ := json.Marshal(map[string]any{"logged_in": true, "source": source, "member": identity})
		return writeTrelloJSON(cmd, data)
	}}
}

func newTrelloLogoutCommand() *cobra.Command {
	return &cobra.Command{Use: "logout", Short: "Delete stored Trello credentials", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if err := trello.DeleteCredentials(); err != nil {
			return err
		}
		message := "Stored Trello credentials deleted."
		if os.Getenv("TRELLO_API_KEY") != "" || os.Getenv("TRELLO_TOKEN") != "" {
			message += " Environment variables are still set and continue to override stored credentials."
		}
		_, err := fmt.Fprintln(cmd.OutOrStdout(), message)
		return err
	}}
}

func readSecretLine(in *os.File, out io.Writer, prompt string) (string, error) {
	if _, err := fmt.Fprint(out, prompt); err != nil {
		return "", err
	}
	if term.IsTerminal(int(in.Fd())) {
		data, err := term.ReadPassword(int(in.Fd()))
		_, _ = fmt.Fprintln(out)
		return string(data), err
	}
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && len(line) == 0 {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func newTrelloBoardsCommand() *cobra.Command {
	return &cobra.Command{Use: "boards", Short: "List accessible boards", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		client, err := getTrelloClient()
		if err != nil {
			return err
		}
		data, err := client.Boards(cmd.Context())
		if err != nil {
			return err
		}
		return writeTrelloJSON(cmd, data)
	}}
}

func newTrelloListsCommand() *cobra.Command {
	return &cobra.Command{Use: "lists <board-id>", Short: "List a board's status lists", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getTrelloClient()
		if err != nil {
			return err
		}
		data, err := client.Lists(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		return writeTrelloJSON(cmd, data)
	}}
}

func newTrelloCardsCommand() *cobra.Command {
	var listID, boardID string
	var today bool
	cmd := &cobra.Command{Use: "cards", Short: "List cards from one list or board", RunE: func(cmd *cobra.Command, _ []string) error {
		if (listID == "") == (boardID == "") {
			return fmt.Errorf("set exactly one of --list or --board")
		}
		client, err := getTrelloClient()
		if err != nil {
			return err
		}
		var data []byte
		if listID != "" {
			data, err = client.ListCards(cmd.Context(), listID)
		} else {
			data, err = client.BoardCards(cmd.Context(), boardID)
		}
		if err != nil {
			return err
		}
		if today {
			data, err = filterTodayCards(data, time.Now())
			if err != nil {
				return err
			}
		}
		return writeTrelloJSON(cmd, data)
	}}
	cmd.Flags().StringVar(&listID, "list", "", "list ID")
	cmd.Flags().StringVar(&boardID, "board", "", "board ID")
	cmd.Flags().BoolVar(&today, "today", false, "only cards due today in the local timezone")
	return cmd
}

func filterTodayCards(data []byte, now time.Time) ([]byte, error) {
	return filterCardsDueOn(data, now.In(now.Location()))
}

// filterCardsDueOn keeps open cards whose due date falls on day's local date.
func filterCardsDueOn(data []byte, day time.Time) ([]byte, error) {
	var rawCards []json.RawMessage
	if err := json.Unmarshal(data, &rawCards); err != nil {
		return nil, err
	}
	location := day.Location()
	day = day.In(location)
	result := make([]json.RawMessage, 0)
	for _, raw := range rawCards {
		var card struct {
			Due    string `json:"due"`
			Closed bool   `json:"closed"`
		}
		if err := json.Unmarshal(raw, &card); err != nil {
			return nil, err
		}
		due, err := time.Parse(time.RFC3339, card.Due)
		if err != nil {
			continue
		}
		if !card.Closed && due.In(location).Format("2006-01-02") == day.Format("2006-01-02") {
			result = append(result, raw)
		}
	}
	return json.Marshal(result)
}

type trelloStatusChange struct {
	CardID         string `json:"card_id"`
	CardName       string `json:"card_name"`
	FromListID     string `json:"from_list_id"`
	FromListName   string `json:"from_list_name"`
	ToListID       string `json:"to_list_id"`
	ToListName     string `json:"to_list_name"`
	ChangedAt      string `json:"changed_at"`
	ChangedAtLocal string `json:"changed_at_local"`
	MemberID       string `json:"member_id,omitempty"`
	MemberName     string `json:"member_name,omitempty"`
}

func parseTrelloStatusChanges(data []byte, toListID string, location *time.Location) ([]trelloStatusChange, error) {
	var actions []struct {
		Type string `json:"type"`
		Date string `json:"date"`
		Data struct {
			Card struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"card"`
			ListBefore struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"listBefore"`
			ListAfter struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"listAfter"`
		} `json:"data"`
		MemberCreator struct {
			ID       string `json:"id"`
			FullName string `json:"fullName"`
			Username string `json:"username"`
		} `json:"memberCreator"`
	}
	if err := json.Unmarshal(data, &actions); err != nil {
		return nil, err
	}
	result := make([]trelloStatusChange, 0, len(actions))
	for _, action := range actions {
		if action.Type != "updateCard" || action.Data.ListBefore.ID == "" || action.Data.ListAfter.ID == "" {
			continue
		}
		if toListID != "" && action.Data.ListAfter.ID != toListID {
			continue
		}
		memberName := action.MemberCreator.FullName
		if memberName == "" {
			memberName = action.MemberCreator.Username
		}
		changedAtLocal := ""
		if changedAt, err := time.Parse(time.RFC3339Nano, action.Date); err == nil {
			changedAtLocal = changedAt.In(location).Format(time.RFC3339)
		}
		result = append(result, trelloStatusChange{
			CardID:         action.Data.Card.ID,
			CardName:       action.Data.Card.Name,
			FromListID:     action.Data.ListBefore.ID,
			FromListName:   action.Data.ListBefore.Name,
			ToListID:       action.Data.ListAfter.ID,
			ToListName:     action.Data.ListAfter.Name,
			ChangedAt:      action.Date,
			ChangedAtLocal: changedAtLocal,
			MemberID:       action.MemberCreator.ID,
			MemberName:     memberName,
		})
	}
	return result, nil
}

func localDayRange(day string, now time.Time) (time.Time, time.Time, error) {
	location := now.Location()
	if day == "" {
		day = now.In(location).Format(time.DateOnly)
	}
	start, err := time.ParseInLocation(time.DateOnly, day, location)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("--date must use YYYY-MM-DD: %w", err)
	}
	return start, start.AddDate(0, 0, 1), nil
}

func newTrelloStatusChangesCommand() *cobra.Command {
	var boardID, day, toListID string
	var limit int
	cmd := &cobra.Command{Use: "status-changes", Short: "List card status changes for one local day", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if boardID == "" {
			return fmt.Errorf("--board is required")
		}
		if limit < 1 || limit > 1000 {
			return fmt.Errorf("--limit must be between 1 and 1000")
		}
		start, end, err := localDayRange(day, time.Now())
		if err != nil {
			return err
		}
		client, err := getTrelloClient()
		if err != nil {
			return err
		}
		data, err := client.BoardStatusChanges(cmd.Context(), boardID, start.Format(time.RFC3339), end.Format(time.RFC3339), limit)
		if err != nil {
			return err
		}
		changes, err := parseTrelloStatusChanges(data, toListID, start.Location())
		if err != nil {
			return err
		}
		body, err := json.Marshal(changes)
		if err != nil {
			return err
		}
		return writeTrelloJSON(cmd, body)
	}}
	cmd.Flags().StringVar(&boardID, "board", "", "board ID")
	cmd.Flags().StringVar(&day, "date", "", "local date in YYYY-MM-DD; defaults to today")
	cmd.Flags().StringVar(&toListID, "to-list", "", "only changes moved into this list ID")
	cmd.Flags().IntVar(&limit, "limit", 1000, "maximum status changes (1-1000)")
	return cmd
}

// trelloWithRetry reruns fn on failure, which keeps one flaky request from
// aborting the whole homework report.
func trelloWithRetry[T any](ctx context.Context, fn func(context.Context) (T, error)) (T, error) {
	var zero T
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		var result T
		result, err = fn(ctx)
		if err == nil || ctx.Err() != nil {
			return result, err
		}
		if attempt+1 < 3 {
			select {
			case <-ctx.Done():
				return zero, ctx.Err()
			case <-time.After(time.Duration(attempt+1) * 2 * time.Second):
			}
		}
	}
	return zero, err
}

func sanitizeTrelloFilename(name string) string {
	name = strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':':
			return '_'
		}
		return r
	}, strings.TrimSpace(name))
	if name == "" || name == "." || name == ".." {
		return "_"
	}
	return name
}

type trelloHomeworkAttachment struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	MIMEType         string `json:"mime_type,omitempty"`
	HomeworkPhotoURL string `json:"homework_photo_url,omitempty"`
	SavedTo          string `json:"saved_to,omitempty"`
	SizeBytes        int    `json:"size_bytes,omitempty"`
	Error            string `json:"error,omitempty"`
}

type trelloHomeworkCard struct {
	ID          string                     `json:"id"`
	Name        string                     `json:"name"`
	ListID      string                     `json:"list_id"`
	ListName    string                     `json:"list_name"`
	Due         string                     `json:"due,omitempty"`
	DueComplete bool                       `json:"due_complete"`
	URL         string                     `json:"url,omitempty"`
	Attachments []trelloHomeworkAttachment `json:"attachments"`
	Error       string                     `json:"error,omitempty"`
}

type trelloHomeworkReport struct {
	Date    string               `json:"date"`
	BoardID string               `json:"board_id"`
	Dir     string               `json:"dir"`
	Cards   []trelloHomeworkCard `json:"cards"`
}

func newTrelloHomeworkCommand() *cobra.Command {
	var boardID, day, dir string
	var maxBytes int64
	var maxWidth int
	cmd := &cobra.Command{Use: "homework", Short: "One-shot report of one day's homework cards with attachments downloaded", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if boardID == "" {
			return fmt.Errorf("--board is required")
		}
		start, _, err := localDayRange(day, time.Now())
		if err != nil {
			return err
		}
		date := start.Format(time.DateOnly)
		if dir == "" {
			dir = "homework-" + date
		}
		client, err := getTrelloClient()
		if err != nil {
			return err
		}
		// 单个请求设超时，避免不稳定链路把整个命令挂死；失败由 trelloWithRetry 重试。
		client.HTTPClient = &http.Client{Timeout: 60 * time.Second}
		ctx := cmd.Context()

		listsData, err := trelloWithRetry(ctx, func(ctx context.Context) ([]byte, error) {
			return client.Lists(ctx, boardID)
		})
		if err != nil {
			return err
		}
		var lists []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(listsData, &lists); err != nil {
			return err
		}
		listNames := make(map[string]string, len(lists))
		for _, list := range lists {
			listNames[list.ID] = list.Name
		}

		cardsData, err := trelloWithRetry(ctx, func(ctx context.Context) ([]byte, error) {
			return client.BoardCards(ctx, boardID)
		})
		if err != nil {
			return err
		}
		cardsData, err = filterCardsDueOn(cardsData, start)
		if err != nil {
			return err
		}
		var cards []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Due         string `json:"due"`
			DueComplete bool   `json:"dueComplete"`
			IDList      string `json:"idList"`
			URL         string `json:"url"`
		}
		if err := json.Unmarshal(cardsData, &cards); err != nil {
			return err
		}

		report := trelloHomeworkReport{Date: date, BoardID: boardID, Dir: dir, Cards: make([]trelloHomeworkCard, 0, len(cards))}
		for _, card := range cards {
			entry := trelloHomeworkCard{
				ID:          card.ID,
				Name:        card.Name,
				ListID:      card.IDList,
				ListName:    listNames[card.IDList],
				Due:         card.Due,
				DueComplete: card.DueComplete,
				URL:         card.URL,
				Attachments: make([]trelloHomeworkAttachment, 0),
			}
			if entry.ListName == "" {
				entry.ListName = card.IDList
			}
			attachmentsData, err := trelloWithRetry(ctx, func(ctx context.Context) ([]byte, error) {
				return client.Attachments(ctx, card.ID)
			})
			if err != nil {
				entry.Error = fmt.Sprintf("list attachments: %v", err)
				report.Cards = append(report.Cards, entry)
				continue
			}
			var attachments []struct {
				ID       string `json:"id"`
				Name     string `json:"name"`
				MIMEType string `json:"mimeType"`
				URL      string `json:"url"`
			}
			if err := json.Unmarshal(attachmentsData, &attachments); err != nil {
				entry.Error = fmt.Sprintf("parse attachments: %v", err)
				report.Cards = append(report.Cards, entry)
				continue
			}
			for _, attachment := range attachments {
				item := trelloHomeworkAttachment{ID: attachment.ID, Name: attachment.Name, MIMEType: attachment.MIMEType, HomeworkPhotoURL: attachment.URL}
				download, err := trelloWithRetry(ctx, func(ctx context.Context) (trello.AttachmentDownload, error) {
					return client.DownloadAttachmentSized(ctx, card.ID, attachment.ID, maxWidth, maxBytes)
				})
				if err != nil {
					item.Error = err.Error()
					entry.Attachments = append(entry.Attachments, item)
					continue
				}
				if download.MIMEType != "" {
					item.MIMEType = download.MIMEType
				}
				cardDir := filepath.Join(dir, sanitizeTrelloFilename(card.Name))
				if err := os.MkdirAll(cardDir, 0755); err != nil {
					item.Error = err.Error()
					entry.Attachments = append(entry.Attachments, item)
					continue
				}
				filename := sanitizeTrelloFilename(download.Name)
				savedTo := filepath.Join(cardDir, filename)
				if _, err := os.Stat(savedTo); err == nil {
					savedTo = filepath.Join(cardDir, sanitizeTrelloFilename(download.ID+"-"+download.Name))
				}
				if err := output.WriteFile(savedTo, download.Data); err != nil {
					item.Error = err.Error()
				} else {
					item.SavedTo = savedTo
					item.SizeBytes = len(download.Data)
				}
				entry.Attachments = append(entry.Attachments, item)
			}
			report.Cards = append(report.Cards, entry)
		}
		if pretty, _ := cmd.Flags().GetBool("pretty"); pretty {
			_, err := fmt.Fprint(cmd.OutOrStdout(), formatTrelloHomeworkReport(report))
			return err
		}
		body, err := json.Marshal(report)
		if err != nil {
			return err
		}
		return writeTrelloJSON(cmd, body)
	}}
	cmd.Flags().StringVar(&boardID, "board", "", "board ID")
	cmd.Flags().StringVar(&day, "date", "", "local date in YYYY-MM-DD; defaults to today")
	cmd.Flags().StringVar(&dir, "dir", "", "directory to save attachments; defaults to homework-<date>")
	cmd.Flags().IntVar(&maxWidth, "max-width", 1200, "download the largest image preview up to this width; 0 downloads the original")
	cmd.Flags().Int64Var(&maxBytes, "max-bytes", 25<<20, "maximum attachment size to download")
	return cmd
}

// formatTrelloHomeworkReport 按 --pretty 约定把作业报告渲染成人可读文本。
func formatTrelloHomeworkReport(report trelloHomeworkReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "日期: %s\n目录: %s\n卡片数: %d\n", report.Date, report.Dir, len(report.Cards))
	for _, card := range report.Cards {
		fmt.Fprintf(&b, "\n[%s] %s\n", card.ListName, card.Name)
		if card.Error != "" {
			fmt.Fprintf(&b, "  错误: %s\n", card.Error)
			continue
		}
		if len(card.Attachments) == 0 {
			b.WriteString("  (无附件)\n")
		}
		for _, att := range card.Attachments {
			if att.Error != "" {
				fmt.Fprintf(&b, "  作业照片: %s (下载失败: %s)\n", att.Name, att.Error)
				continue
			}
			fmt.Fprintf(&b, "  作业照片: %s (%d bytes)\n", att.Name, att.SizeBytes)
			if att.SavedTo != "" {
				fmt.Fprintf(&b, "    本地: %s\n", att.SavedTo)
			}
			if att.HomeworkPhotoURL != "" {
				fmt.Fprintf(&b, "    链接: %s\n", att.HomeworkPhotoURL)
			}
		}
	}
	return b.String()
}

func newTrelloCardCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "card", Short: "Create, update, move, and inspect cards"}
	cmd.AddCommand(
		newTrelloGetCardCommand(),
		newTrelloCreateCardCommand(),
		newTrelloUpdateCardCommand(),
		newTrelloMoveCardCommand(),
		newTrelloDeleteCardCommand(),
		newTrelloAttachCardCommand(),
		newTrelloAttachmentsCommand(),
		newTrelloAttachmentCommand(),
		newTrelloActionsCommand(),
	)
	return cmd
}

func newTrelloGetCardCommand() *cobra.Command {
	return &cobra.Command{Use: "get <card-id>", Short: "Get a card's detail", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getTrelloClient()
		if err != nil {
			return err
		}
		data, err := client.Card(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		return writeTrelloJSON(cmd, data)
	}}
}

func newTrelloDeleteCardCommand() *cobra.Command {
	return &cobra.Command{Use: "delete <card-id>", Short: "Permanently delete a card", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getTrelloClient()
		if err != nil {
			return err
		}
		data, err := client.DeleteCard(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		return writeTrelloJSON(cmd, data)
	}}
}

func newTrelloCreateCardCommand() *cobra.Command {
	var listID, name, desc, due, labels string
	cmd := &cobra.Command{Use: "create", Short: "Create a task card", RunE: func(cmd *cobra.Command, _ []string) error {
		if listID == "" || name == "" {
			return fmt.Errorf("--list and --name are required")
		}
		fields := url.Values{"idList": {listID}, "name": {name}}
		if desc != "" {
			fields.Set("desc", desc)
		}
		if due != "" {
			fields.Set("due", due)
		}
		if labels != "" {
			fields.Set("idLabels", labels)
		}
		client, err := getTrelloClient()
		if err != nil {
			return err
		}
		data, err := client.CreateCard(cmd.Context(), fields)
		if err != nil {
			return err
		}
		return writeTrelloJSON(cmd, data)
	}}
	cmd.Flags().StringVar(&listID, "list", "", "target list ID")
	cmd.Flags().StringVar(&name, "name", "", "card title")
	cmd.Flags().StringVar(&desc, "desc", "", "card description")
	cmd.Flags().StringVar(&due, "due", "", "RFC3339 deadline")
	cmd.Flags().StringVar(&labels, "labels", "", "comma-separated label IDs")
	return cmd
}

func newTrelloUpdateCardCommand() *cobra.Command {
	var name, desc, due string
	var complete bool
	cmd := &cobra.Command{Use: "update <card-id>", Short: "Update a card", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		fields := url.Values{}
		for _, field := range []struct{ name, value string }{{"name", name}, {"desc", desc}, {"due", due}} {
			if cmd.Flags().Changed(field.name) {
				fields.Set(field.name, field.value)
			}
		}
		if cmd.Flags().Changed("complete") {
			fields.Set("dueComplete", strconv.FormatBool(complete))
		}
		if len(fields) == 0 {
			return fmt.Errorf("provide at least one field to update")
		}
		client, err := getTrelloClient()
		if err != nil {
			return err
		}
		data, err := client.UpdateCard(cmd.Context(), args[0], fields)
		if err != nil {
			return err
		}
		return writeTrelloJSON(cmd, data)
	}}
	cmd.Flags().StringVar(&name, "name", "", "new title")
	cmd.Flags().StringVar(&desc, "desc", "", "new description")
	cmd.Flags().StringVar(&due, "due", "", "RFC3339 deadline; empty clears it")
	cmd.Flags().BoolVar(&complete, "complete", false, "mark due date complete or incomplete")
	return cmd
}

func newTrelloMoveCardCommand() *cobra.Command {
	var listID string
	cmd := &cobra.Command{Use: "move <card-id>", Short: "Move a card to a status list", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if listID == "" {
			return fmt.Errorf("--list is required")
		}
		client, err := getTrelloClient()
		if err != nil {
			return err
		}
		data, err := client.UpdateCard(cmd.Context(), args[0], url.Values{"idList": {listID}})
		if err != nil {
			return err
		}
		return writeTrelloJSON(cmd, data)
	}}
	cmd.Flags().StringVar(&listID, "list", "", "destination list ID")
	return cmd
}

func newTrelloAttachCardCommand() *cobra.Command {
	var file, name string
	cmd := &cobra.Command{Use: "attach <card-id>", Short: "Upload a result photo or file", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if file == "" {
			return fmt.Errorf("--file is required")
		}
		client, err := getTrelloClient()
		if err != nil {
			return err
		}
		data, err := client.AttachFile(cmd.Context(), args[0], file, name)
		if err != nil {
			return err
		}
		return writeTrelloJSON(cmd, data)
	}}
	cmd.Flags().StringVar(&file, "file", "", "local file path")
	cmd.Flags().StringVar(&name, "name", "", "attachment display name")
	return cmd
}

func newTrelloAttachmentsCommand() *cobra.Command {
	return &cobra.Command{Use: "attachments <card-id>", Short: "List a card's attachments", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getTrelloClient()
		if err != nil {
			return err
		}
		data, err := client.Attachments(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		return writeTrelloJSON(cmd, data)
	}}
}

func newTrelloAttachmentCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "attachment", Short: "Read an attachment's content"}
	cmd.AddCommand(newTrelloGetAttachmentContentCommand())
	return cmd
}

func newTrelloGetAttachmentContentCommand() *cobra.Command {
	var maxBytes int64
	var maxWidth int
	var outFile string
	cmd := &cobra.Command{Use: "get <card-id> <attachment-id>", Short: "Download an attachment", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getTrelloClient()
		if err != nil {
			return err
		}
		item, err := client.DownloadAttachmentSized(cmd.Context(), args[0], args[1], maxWidth, maxBytes)
		if err != nil {
			return err
		}
		body, err := formatTrelloAttachment(item, outFile)
		if err != nil {
			return err
		}
		return writeTrelloJSON(cmd, body)
	}}
	cmd.Flags().IntVar(&maxWidth, "max-width", 0, "download the largest image preview up to this width (WebP, much smaller); 0 downloads the original")
	cmd.Flags().Int64Var(&maxBytes, "max-bytes", 25<<20, "maximum attachment size to return")
	cmd.Flags().StringVarP(&outFile, "out", "o", "", "save attachment bytes directly to this file")
	return cmd
}

func formatTrelloAttachment(item trello.AttachmentDownload, outFile string) ([]byte, error) {
	result := map[string]interface{}{
		"id":           item.ID,
		"name":         item.Name,
		"mime_type":    item.MIMEType,
		"content_type": item.ContentType,
		"size_bytes":   len(item.Data),
	}
	if outFile != "" {
		if err := output.WriteFile(outFile, item.Data); err != nil {
			return nil, err
		}
		result["saved_to"] = outFile
	} else {
		result["base64"] = base64.StdEncoding.EncodeToString(item.Data)
	}
	return json.Marshal(result)
}

func newTrelloActionsCommand() *cobra.Command {
	var limit int
	cmd := &cobra.Command{Use: "actions <card-id>", Short: "List card history", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getTrelloClient()
		if err != nil {
			return err
		}
		data, err := client.Actions(cmd.Context(), args[0], limit)
		if err != nil {
			return err
		}
		return writeTrelloJSON(cmd, data)
	}}
	cmd.Flags().IntVar(&limit, "limit", 50, "maximum actions")
	return cmd
}

func newTrelloWebhookCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "webhook", Short: "Manage Trello webhooks"}
	var callbackURL, modelID, description string
	create := &cobra.Command{Use: "create", Short: "Register a webhook", RunE: func(cmd *cobra.Command, _ []string) error {
		if callbackURL == "" || modelID == "" {
			return fmt.Errorf("--callback-url and --model-id are required")
		}
		fields := url.Values{"callbackURL": {callbackURL}, "idModel": {modelID}}
		if description != "" {
			fields.Set("description", description)
		}
		client, err := getTrelloClient()
		if err != nil {
			return err
		}
		data, err := client.CreateWebhook(cmd.Context(), fields)
		if err != nil {
			return err
		}
		return writeTrelloJSON(cmd, data)
	}}
	create.Flags().StringVar(&callbackURL, "callback-url", "", "public HTTPS callback URL")
	create.Flags().StringVar(&modelID, "model-id", "", "board, list, card, or member ID to watch")
	create.Flags().StringVar(&description, "description", "", "webhook description")
	deleteCmd := &cobra.Command{Use: "delete <webhook-id>", Short: "Delete a webhook", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getTrelloClient()
		if err != nil {
			return err
		}
		data, err := client.DeleteWebhook(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		return writeTrelloJSON(cmd, data)
	}}
	cmd.AddCommand(create, deleteCmd)
	return cmd
}

// ---------- Board CRUD ----------

func newTrelloBoardCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "board", Short: "Get, create, update, and delete boards"}
	cmd.AddCommand(newTrelloGetBoardCommand(), newTrelloCreateBoardCommand(), newTrelloUpdateBoardCommand(), newTrelloDeleteBoardCommand())
	return cmd
}

func newTrelloGetBoardCommand() *cobra.Command {
	return &cobra.Command{Use: "get <board-id>", Short: "Get a board's detail", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getTrelloClient()
		if err != nil {
			return err
		}
		data, err := client.Board(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		return writeTrelloJSON(cmd, data)
	}}
}

func newTrelloCreateBoardCommand() *cobra.Command {
	var name, desc, orgID string
	var defaultLists bool
	cmd := &cobra.Command{Use: "create", Short: "Create a new board", RunE: func(cmd *cobra.Command, _ []string) error {
		if name == "" {
			return fmt.Errorf("--name is required")
		}
		fields := url.Values{"name": {name}}
		if desc != "" {
			fields.Set("desc", desc)
		}
		if orgID != "" {
			fields.Set("idOrganization", orgID)
		}
		// Trello 默认会创建 To Do / Doing / Done 三个 List；通过 defaultLists=false 可禁用。
		if cmd.Flags().Changed("default-lists") {
			fields.Set("defaultLists", strconv.FormatBool(defaultLists))
		}
		client, err := getTrelloClient()
		if err != nil {
			return err
		}
		data, err := client.CreateBoard(cmd.Context(), fields)
		if err != nil {
			return err
		}
		return writeTrelloJSON(cmd, data)
	}}
	cmd.Flags().StringVar(&name, "name", "", "board name")
	cmd.Flags().StringVar(&desc, "desc", "", "board description")
	cmd.Flags().StringVar(&orgID, "org", "", "workspace (organization) ID")
	cmd.Flags().BoolVar(&defaultLists, "default-lists", true, "create the default To Do/Doing/Done lists")
	return cmd
}

func newTrelloUpdateBoardCommand() *cobra.Command {
	var name, desc string
	var closed bool
	cmd := &cobra.Command{Use: "update <board-id>", Short: "Update a board", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		fields := url.Values{}
		if cmd.Flags().Changed("name") {
			fields.Set("name", name)
		}
		if cmd.Flags().Changed("desc") {
			fields.Set("desc", desc)
		}
		if cmd.Flags().Changed("closed") {
			fields.Set("closed", strconv.FormatBool(closed))
		}
		if len(fields) == 0 {
			return fmt.Errorf("provide at least one field to update")
		}
		client, err := getTrelloClient()
		if err != nil {
			return err
		}
		data, err := client.UpdateBoard(cmd.Context(), args[0], fields)
		if err != nil {
			return err
		}
		return writeTrelloJSON(cmd, data)
	}}
	cmd.Flags().StringVar(&name, "name", "", "new name")
	cmd.Flags().StringVar(&desc, "desc", "", "new description")
	cmd.Flags().BoolVar(&closed, "closed", false, "archive (true) or reopen (false) the board")
	return cmd
}

func newTrelloDeleteBoardCommand() *cobra.Command {
	return &cobra.Command{Use: "delete <board-id>", Short: "Permanently delete a board", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getTrelloClient()
		if err != nil {
			return err
		}
		data, err := client.DeleteBoard(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		return writeTrelloJSON(cmd, data)
	}}
}

// ---------- List CRUD ----------

func newTrelloListCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "list", Short: "Get, create, update, and archive lists"}
	cmd.AddCommand(newTrelloGetListCommand(), newTrelloCreateListCommand(), newTrelloUpdateListCommand(), newTrelloArchiveListCommand())
	return cmd
}

func newTrelloGetListCommand() *cobra.Command {
	return &cobra.Command{Use: "get <list-id>", Short: "Get a list's detail", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getTrelloClient()
		if err != nil {
			return err
		}
		data, err := client.List(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		return writeTrelloJSON(cmd, data)
	}}
}

func newTrelloCreateListCommand() *cobra.Command {
	var boardID, name, pos string
	cmd := &cobra.Command{Use: "create", Short: "Create a new list on a board", RunE: func(cmd *cobra.Command, _ []string) error {
		if boardID == "" || name == "" {
			return fmt.Errorf("--board and --name are required")
		}
		fields := url.Values{"idBoard": {boardID}, "name": {name}}
		if pos != "" {
			fields.Set("pos", pos)
		}
		client, err := getTrelloClient()
		if err != nil {
			return err
		}
		data, err := client.CreateList(cmd.Context(), fields)
		if err != nil {
			return err
		}
		return writeTrelloJSON(cmd, data)
	}}
	cmd.Flags().StringVar(&boardID, "board", "", "target board ID")
	cmd.Flags().StringVar(&name, "name", "", "list name")
	cmd.Flags().StringVar(&pos, "pos", "", "position: top, bottom, or numeric")
	return cmd
}

func newTrelloUpdateListCommand() *cobra.Command {
	var name, pos, boardID string
	var closed bool
	cmd := &cobra.Command{Use: "update <list-id>", Short: "Update a list", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		fields := url.Values{}
		if cmd.Flags().Changed("name") {
			fields.Set("name", name)
		}
		if cmd.Flags().Changed("pos") {
			fields.Set("pos", pos)
		}
		if cmd.Flags().Changed("board") {
			fields.Set("idBoard", boardID)
		}
		if cmd.Flags().Changed("closed") {
			fields.Set("closed", strconv.FormatBool(closed))
		}
		if len(fields) == 0 {
			return fmt.Errorf("provide at least one field to update")
		}
		client, err := getTrelloClient()
		if err != nil {
			return err
		}
		data, err := client.UpdateList(cmd.Context(), args[0], fields)
		if err != nil {
			return err
		}
		return writeTrelloJSON(cmd, data)
	}}
	cmd.Flags().StringVar(&name, "name", "", "new name")
	cmd.Flags().StringVar(&pos, "pos", "", "new position: top, bottom, or numeric")
	cmd.Flags().StringVar(&boardID, "board", "", "move to a different board")
	cmd.Flags().BoolVar(&closed, "closed", false, "archive (true) or reopen (false) the list")
	return cmd
}

func newTrelloArchiveListCommand() *cobra.Command {
	// Trello REST 不提供 List 的真删除，这里以归档代替；如需恢复，用 `list update --closed=false`。
	return &cobra.Command{Use: "archive <list-id>", Short: "Archive a list (Trello has no hard delete for lists)", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getTrelloClient()
		if err != nil {
			return err
		}
		data, err := client.ArchiveList(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		return writeTrelloJSON(cmd, data)
	}}
}
