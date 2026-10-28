package trello

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

const GradeFieldName = "打分结果"

func ValidGrade(grade string) bool {
	return grade == "100" || grade == "优" || grade == "良" || grade == "中" || grade == "差"
}

// The first line is reserved score metadata. Never rewrite assignment text below it.
func withReviewGrade(description, grade string, provisional bool) string {
	value := grade
	if provisional {
		value = "暂评·" + grade
	}
	line := GradeFieldName + "：" + value
	first, rest, hasNewline := strings.Cut(description, "\n")
	if strings.HasPrefix(first, GradeFieldName+"：") || strings.HasPrefix(first, GradeFieldName+":") {
		if hasNewline {
			return line + "\n" + rest
		}
		return line
	}
	if description == "" {
		return line
	}
	return line + "\n" + description
}

func (c *Client) applyReviewGrade(ctx context.Context, state *reviewState) error {
	if state.Grade == "" || state.GradeApplied {
		return nil
	}
	raw, err := c.get(ctx, "cards/"+state.CardID, url.Values{"fields": {"desc"}})
	if err != nil {
		return err
	}
	var card struct {
		Description *string `json:"desc"`
	}
	if err := json.Unmarshal(raw, &card); err != nil {
		return err
	}
	if card.Description == nil {
		return fmt.Errorf("review published but card description is missing; retry same --review-id")
	}
	description := withReviewGrade(*card.Description, state.Grade, state.Provisional)
	if description != *card.Description {
		if _, err := c.UpdateCard(ctx, state.CardID, url.Values{"desc": {description}}); err != nil {
			return fmt.Errorf("review published but grade update failed; retry same --review-id: %w", err)
		}
	}
	state.GradeApplied = true
	return saveReviewState(state.StatePath, state)
}
