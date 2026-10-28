package trello

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const ReviewPrefix = "作业批改：\n"

// ReviewOptions describes one publication. Reuse ReviewID and StatePath on retry.
type ReviewOptions struct {
	CardID, ReviewID, Type, Text, File, StatePath string
	Grade                                         string
	Provisional                                   bool
	RetryUncertain                                bool
	HasErrors                                     *bool
}

type ReviewResult struct {
	ReviewID     string `json:"review_id"`
	CardID       string `json:"card_id"`
	AttachmentID string `json:"attachment_id,omitempty"`
	ActionID     string `json:"action_id"`
	StatePath    string `json:"state_file"`
	Status       string `json:"status,omitempty"`
	Grade        string `json:"grade,omitempty"`
	Provisional  bool   `json:"provisional,omitempty"`
}

type reviewState struct {
	ReviewResult
	Type          string   `json:"type"`
	Fingerprint   string   `json:"fingerprint"`
	Name          string   `json:"name,omitempty"`
	Comment       string   `json:"comment,omitempty"`
	Phase         string   `json:"phase"`
	Baseline      []string `json:"baseline,omitempty"`
	HasErrors     *bool    `json:"has_errors,omitempty"`
	StatusApplied bool     `json:"status_applied,omitempty"`
	GradeApplied  bool     `json:"grade_applied,omitempty"`
	TargetListID  string   `json:"target_list_id,omitempty"`
}

type reviewPayload struct {
	HasErrors    *bool  `json:"has_errors,omitempty"`
	Grade        string `json:"grade,omitempty"`
	Provisional  bool   `json:"provisional,omitempty"`
	Version      int    `json:"version"`
	Type         string `json:"type"`
	Text         string `json:"text,omitempty"`
	AttachmentID string `json:"attachment_id,omitempty"`
}

type reviewAttachment struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	MIMEType string `json:"mimeType"`
	IsUpload bool   `json:"isUpload"`
}

type reviewComment struct {
	ID   string `json:"id"`
	Data struct {
		Text string `json:"text"`
	} `json:"data"`
}

// ReviewFileType uses an explicit mapping so MIME types are stable across systems.
func ReviewFileType(kind, filename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	images := map[string]string{".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".png": "image/png", ".webp": "image/webp", ".gif": "image/gif"}
	audio := map[string]string{".m4a": "audio/mp4", ".mp3": "audio/mpeg", ".wav": "audio/wav", ".aac": "audio/aac", ".ogg": "audio/ogg", ".opus": "audio/ogg", ".amr": "audio/amr", ".3gp": "audio/3gpp"}
	types := images
	if kind == "audio" {
		types = audio
	} else if kind != "image" {
		return "", fmt.Errorf("unknown review type %q", kind)
	}
	if value := types[ext]; value != "" {
		return value, nil
	}
	return "", fmt.Errorf("unsupported %s extension %q", kind, ext)
}

func saveReviewState(path string, state *reviewState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".review-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err = file.Write(data); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(file.Name(), path)
}

func (c *Client) reviewComments(ctx context.Context, cardID string) ([]reviewComment, error) {
	var all []reviewComment
	before := ""
	seen := map[string]bool{}
	for {
		query := url.Values{"filter": {"commentCard"}, "limit": {"1000"}}
		if before != "" {
			query.Set("before", before)
		}
		data, err := c.get(ctx, "cards/"+cardID+"/actions", query)
		if err != nil {
			return nil, err
		}
		var page []reviewComment
		if err := json.Unmarshal(data, &page); err != nil {
			return nil, err
		}
		all = append(all, page...)
		if len(page) < 1000 {
			return all, nil
		}
		before = page[len(page)-1].ID
		if before == "" || seen[before] {
			return nil, fmt.Errorf("comment pagination did not advance")
		}
		seen[before] = true
	}
}

// PublishReview appends a native attachment/comment and optionally applies the explicit review outcome.
// Durable intent checkpoints precede POSTs; uncertain writes are reconciled before retry.
func (c *Client) PublishReview(ctx context.Context, opts ReviewOptions) (ReviewResult, error) {
	var empty ReviewResult
	if (opts.Grade != "" && !ValidGrade(opts.Grade)) || (opts.Provisional && opts.Grade == "") {
		return empty, fmt.Errorf("--grade must be 100, 优, 良, 中 or 差; --provisional requires --grade")
	}
	if opts.Grade == "100" {
		if opts.Provisional || (opts.HasErrors != nil && *opts.HasErrors) {
			return empty, fmt.Errorf("--grade 100 means fully correct; do not combine it with --provisional or --has-errors=true")
		}
		passed := false
		opts.HasErrors = &passed
	}
	if opts.CardID == "" || strings.ContainsAny(opts.CardID, "/?#") {
		return empty, fmt.Errorf("a card ID is required")
	}
	if opts.ReviewID == "" || strings.ContainsAny(opts.ReviewID, "/\\") || opts.StatePath == "" {
		return empty, fmt.Errorf("review ID and state path are required")
	}
	var content []byte
	var mimeType, name string
	if opts.Type == "text" {
		if strings.TrimSpace(opts.Text) == "" || !utf8.ValidString(opts.Text) || opts.File != "" {
			return empty, fmt.Errorf("text review requires nonempty UTF-8 text and no file")
		}
		content = []byte(opts.Text)
	} else {
		if opts.Text != "" {
			return empty, fmt.Errorf("media reviews cannot contain text")
		}
		var err error
		mimeType, err = ReviewFileType(opts.Type, opts.File)
		if err != nil {
			return empty, err
		}
		f, err := os.Open(opts.File)
		if err != nil {
			return empty, err
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil {
			return empty, err
		}
		if !info.Mode().IsRegular() || info.Size() == 0 || info.Size() > 50<<20 {
			return empty, fmt.Errorf("review file must be a nonempty regular file of at most 50 MiB")
		}
		content, err = io.ReadAll(io.LimitReader(f, (50<<20)+1))
		if err != nil {
			return empty, err
		}
		if len(content) == 0 || len(content) > 50<<20 {
			return empty, fmt.Errorf("review file must be nonempty and at most 50 MiB")
		}
		name = "homework-review-" + opts.Type + "-" + opts.ReviewID + strings.ToLower(filepath.Ext(opts.File))
	}
	sum := sha256.Sum256(content)
	fingerprint := hex.EncodeToString(sum[:])
	if err := os.MkdirAll(filepath.Dir(opts.StatePath), 0700); err != nil {
		return empty, err
	}
	lock, err := os.OpenFile(opts.StatePath+".lock", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return empty, fmt.Errorf("lock review state (ensure no process is using it before removing a stale .lock): %w", err)
	}
	lock.Close()
	defer os.Remove(opts.StatePath + ".lock")
	state := reviewState{ReviewResult: ReviewResult{ReviewID: opts.ReviewID, CardID: opts.CardID, StatePath: opts.StatePath, Grade: opts.Grade, Provisional: opts.Provisional}, Type: opts.Type, Fingerprint: fingerprint, Name: name, Phase: "ready", HasErrors: opts.HasErrors}
	data, err := os.ReadFile(opts.StatePath)
	if err == nil {
		var saved reviewState
		if err := json.Unmarshal(data, &saved); err != nil {
			return empty, fmt.Errorf("decode review state: %w", err)
		}
		state = saved
		if state.ReviewID != opts.ReviewID || state.CardID != opts.CardID || state.Type != opts.Type || state.Fingerprint != fingerprint || state.Name != name || !sameReviewOutcome(state.HasErrors, opts.HasErrors) || state.Grade != opts.Grade || state.Provisional != opts.Provisional {
			return empty, fmt.Errorf("review ID already belongs to different content or card; use a new --review-id")
		}
	} else if !os.IsNotExist(err) {
		return empty, err
	}
	if state.ActionID != "" {
		return c.finishReview(ctx, &state)
	}
	save := func() error { return saveReviewState(opts.StatePath, &state) }
	if err := save(); err != nil {
		return empty, err
	}
	uncertain := func() error {
		return fmt.Errorf("previous %s outcome is uncertain; no matching record found; retry with the same --review-id later, or use --retry-uncertain only after confirming the write failed", state.Phase)
	}
	if opts.Type != "text" {
		raw, err := c.Attachments(ctx, opts.CardID)
		if err != nil {
			return empty, err
		}
		var attachments []reviewAttachment
		if err := json.Unmarshal(raw, &attachments); err != nil {
			return empty, err
		}
		var found *reviewAttachment
		for i := range attachments {
			att := &attachments[i]
			if (state.AttachmentID != "" && att.ID == state.AttachmentID) || att.Name == name {
				if found != nil {
					return empty, fmt.Errorf("multiple review attachments found; inspect card before retrying")
				}
				found = att
			}
		}
		if found != nil {
			if found.ID == "" || !found.IsUpload || found.MIMEType != mimeType || found.Name != name {
				return empty, fmt.Errorf("review attachment is not a native upload of the expected type")
			}
			state.AttachmentID = found.ID
		} else {
			if state.AttachmentID != "" {
				return empty, fmt.Errorf("previous review attachment is missing; start a new review")
			}
			if state.Phase == "uploading" && !opts.RetryUncertain {
				return empty, uncertain()
			}
			state.Phase = "uploading"
			if err := save(); err != nil {
				return empty, err
			}
			raw, err := c.attachFileContent(ctx, opts.CardID, bytes.NewReader(content), name, mimeType)
			if err != nil {
				return empty, fmt.Errorf("upload review (retry same review ID to reconcile): %w", err)
			}
			var att reviewAttachment
			if err := json.Unmarshal(raw, &att); err != nil {
				return empty, err
			}
			if att.ID == "" || !att.IsUpload || att.MIMEType != mimeType || att.Name != name {
				return empty, fmt.Errorf("upload response lacks a native attachment ID or expected MIME type; retry same review ID to reconcile")
			}
			state.AttachmentID = att.ID
		}
		if state.Phase != "publishing" {
			state.Phase = "uploaded"
		}
		if err := save(); err != nil {
			return empty, err
		}
	}
	payload, err := json.Marshal(reviewPayload{Version: 1, Type: opts.Type, Text: opts.Text, AttachmentID: state.AttachmentID, HasErrors: opts.HasErrors, Grade: opts.Grade, Provisional: opts.Provisional})
	if err != nil {
		return empty, err
	}
	state.Comment = ReviewPrefix + string(payload)
	comments, err := c.reviewComments(ctx, opts.CardID)
	if err != nil {
		return empty, err
	}
	if state.Phase == "publishing" {
		baseline := map[string]bool{}
		for _, id := range state.Baseline {
			baseline[id] = true
		}
		for _, comment := range comments {
			if comment.ID != "" && comment.Data.Text == state.Comment && !baseline[comment.ID] {
				if state.ActionID != "" {
					return empty, fmt.Errorf("multiple matching comments found; inspect card before retrying")
				}
				state.ActionID = comment.ID
			}
		}
		if state.ActionID != "" {
			if err := save(); err != nil {
				return empty, err
			}
			return c.finishReview(ctx, &state)
		}
		if !opts.RetryUncertain {
			return empty, uncertain()
		}
	}
	state.Baseline = nil
	for _, comment := range comments {
		if comment.Data.Text == state.Comment {
			state.Baseline = append(state.Baseline, comment.ID)
		}
	}
	state.Phase = "publishing"
	if err := save(); err != nil {
		return empty, err
	}
	raw, err := c.request(ctx, http.MethodPost, "cards/"+opts.CardID+"/actions/comments", nil, strings.NewReader(url.Values{"text": {state.Comment}}.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return empty, fmt.Errorf("publish review (retry same review ID to reconcile): %w", err)
	}
	var action struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &action); err != nil {
		return empty, err
	}
	if action.ID == "" {
		return empty, fmt.Errorf("publication response has no action ID; retry same review ID to reconcile")
	}
	state.ActionID = action.ID
	if err := save(); err != nil {
		return empty, err
	}
	return c.finishReview(ctx, &state)
}

func sameReviewOutcome(a, b *bool) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && *a == *b)
}

// Checkpoint the comment independently so a failed status update never republishes it.
func (c *Client) finishReview(ctx context.Context, state *reviewState) (ReviewResult, error) {
	if err := c.applyReviewGrade(ctx, state); err != nil {
		return ReviewResult{}, err
	}
	if state.HasErrors == nil || state.StatusApplied {
		return state.ReviewResult, nil
	}
	if state.TargetListID == "" {
		raw, err := c.get(ctx, "cards/"+state.CardID, url.Values{"fields": {"idBoard,idList"}})
		if err != nil {
			return ReviewResult{}, err
		}
		var card struct {
			BoardID string `json:"idBoard"`
			ListID  string `json:"idList"`
		}
		if err := json.Unmarshal(raw, &card); err != nil {
			return ReviewResult{}, err
		}
		if card.BoardID == "" {
			return ReviewResult{}, fmt.Errorf("review published, but card has no board ID; retry with the same --review-id")
		}
		raw, err = c.Lists(ctx, card.BoardID)
		if err != nil {
			return ReviewResult{}, err
		}
		var lists []struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Closed bool   `json:"closed"`
		}
		if err := json.Unmarshal(raw, &lists); err != nil {
			return ReviewResult{}, err
		}
		target := "已完成"
		if *state.HasErrors {
			target = "待改错"
		} else {
			for _, list := range lists {
				if list.ID == card.ListID && list.Name == "已改错" && !list.Closed {
					target = "已改错"
				}
			}
		}
		for _, list := range lists {
			if list.Name == target && !list.Closed {
				state.TargetListID = list.ID
				break
			}
		}
		if state.TargetListID == "" {
			raw, err = c.CreateList(ctx, url.Values{"idBoard": {card.BoardID}, "name": {target}})
			if err != nil {
				return ReviewResult{}, err
			}
			var list struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(raw, &list); err != nil {
				return ReviewResult{}, err
			}
			if list.ID == "" {
				return ReviewResult{}, fmt.Errorf("review published, but list creation returned no ID; retry with the same --review-id")
			}
			state.TargetListID = list.ID
		}
		state.Status = target
		if err := saveReviewState(state.StatePath, state); err != nil {
			return ReviewResult{}, err
		}
	}
	complete := "true"
	if *state.HasErrors {
		complete = "false"
	}
	if _, err := c.UpdateCard(ctx, state.CardID, url.Values{"idList": {state.TargetListID}, "dueComplete": {complete}}); err != nil {
		return ReviewResult{}, fmt.Errorf("review published but status update failed; retry with the same --review-id: %w", err)
	}
	state.StatusApplied = true
	if err := saveReviewState(state.StatePath, state); err != nil {
		return ReviewResult{}, err
	}
	return state.ReviewResult, nil
}
