package cli

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/mnhkahn/cyeam-cli/internal/output"
	"github.com/mnhkahn/cyeam-cli/internal/trello"
	"github.com/spf13/cobra"
)

func newTrelloCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "trello", Short: "Manage Trello boards and task cards"}
	cmd.AddCommand(newTrelloBoardsCommand(), newTrelloListsCommand(), newTrelloCardsCommand(), newTrelloCardCommand(), newTrelloWebhookCommand())
	return cmd
}

func getTrelloClient() (*trello.Client, error) { return trello.NewFromEnv() }
func writeTrelloJSON(cmd *cobra.Command, data []byte) error {
	return output.WriteJSON(cmd.OutOrStdout(), data)
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
	var rawCards []json.RawMessage
	if err := json.Unmarshal(data, &rawCards); err != nil {
		return nil, err
	}
	location := now.Location()
	day := now.In(location).Format("2006-01-02")
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
		if !card.Closed && due.In(location).Format("2006-01-02") == day {
			result = append(result, raw)
		}
	}
	return json.Marshal(result)
}

func newTrelloCardCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "card", Short: "Create, update, move, and inspect cards"}
	cmd.AddCommand(newTrelloCreateCardCommand(), newTrelloUpdateCardCommand(), newTrelloMoveCardCommand(), newTrelloAttachCardCommand(), newTrelloAttachmentsCommand(), newTrelloActionsCommand())
	return cmd
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
