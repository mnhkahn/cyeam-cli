package cli

import (
	"io"
	"strings"
	"testing"
)

func TestTrelloReviewFlagValidation(t *testing.T) {
	for _, args := range [][]string{
		{"card-1"},
		{"card-1", "--text", "hi", "--image", "image.png"},
		{"card-1", "--audio", "a.m4a", "--text-file", "text.txt"},
		{"card-1", "--text", "hi", "--review-id", "../escape"},
		{"card-1", "--text", "hi", "--review-id", ""},
		{"card-1", "--text", "hi", "--retry-uncertain"},
	} {
		cmd := newTrelloReviewCommand()
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		cmd.SetArgs(args)
		err := cmd.Execute()
		if err == nil || strings.Contains(err.Error(), "not logged in") {
			t.Fatalf("args %v: %v", args, err)
		}
	}
}

func TestTrelloCardRegistersReview(t *testing.T) {
	cmd, _, err := newTrelloCommand().Find([]string{"card", "review"})
	if err != nil || cmd.Name() != "review" {
		t.Fatal(cmd, err)
	}
}

func TestReviewOutcomeFlagPreservesExplicitFalse(t *testing.T) {
	for _, arg := range []string{"--has-errors=true", "--has-errors=false"} {
		cmd := newTrelloReviewCommand()
		if err := cmd.ParseFlags([]string{arg}); err != nil {
			t.Fatal(err)
		}
		if !cmd.Flags().Changed("has-errors") {
			t.Fatal("explicit false must differ from omission")
		}
	}
}

func TestHomeworkStatusUsesFourLists(t *testing.T) {
	for list, want := range map[string]string{"待完成": "todo", "已完成": "completed", "待改错": "needs_correction", "已改错": "corrected", "其他": "unknown"} {
		if got := trelloHomeworkStatus(list); got != want {
			t.Errorf("%s: %s, want %s", list, got, want)
		}
	}
}
