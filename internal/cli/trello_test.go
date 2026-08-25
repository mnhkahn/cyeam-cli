package cli

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mnhkahn/cyeam-cli/internal/trello"
)

func TestFilterTodayCardsUsesLocalDeadlineDate(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*60*60)
	now := time.Date(2026, 7, 11, 9, 0, 0, 0, loc)
	input := []byte(`[
  {"id":"today","due":"2026-07-11T00:30:00Z","closed":false},
  {"id":"tomorrow","due":"2026-07-11T18:00:00Z","closed":false},
  {"id":"closed","due":"2026-07-11T00:30:00Z","closed":true}
]`)
	output, err := filterTodayCards(input, now)
	if err != nil {
		t.Fatal(err)
	}
	var cards []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(output, &cards); err != nil {
		t.Fatal(err)
	}
	if len(cards) != 1 || cards[0].ID != "today" {
		t.Fatalf("cards = %s", output)
	}
}

func TestLocalDayRangeUsesLocalTimezone(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*60*60)
	now := time.Date(2026, 7, 29, 9, 0, 0, 0, loc)
	start, end, err := localDayRange("", now)
	if err != nil {
		t.Fatal(err)
	}
	if got := start.Format(time.RFC3339); got != "2026-07-29T00:00:00+08:00" {
		t.Fatalf("start = %s", got)
	}
	if got := end.Format(time.RFC3339); got != "2026-07-30T00:00:00+08:00" {
		t.Fatalf("end = %s", got)
	}
}

func TestParseTrelloStatusChangesFiltersCompletionList(t *testing.T) {
	input := []byte(`[
  {
    "type":"updateCard",
    "date":"2026-07-29T02:30:00.000Z",
    "data":{
      "card":{"id":"card-1","name":"完成日报"},
      "listBefore":{"id":"doing","name":"进行中"},
      "listAfter":{"id":"done","name":"已完成"}
    },
    "memberCreator":{"id":"member-1","fullName":"张三","username":"zhangsan"}
  },
  {
    "type":"updateCard",
    "date":"2026-07-29T03:00:00.000Z",
    "data":{
      "card":{"id":"card-2","name":"继续处理"},
      "listBefore":{"id":"todo","name":"待处理"},
      "listAfter":{"id":"doing","name":"进行中"}
    }
  }
]`)
	location := time.FixedZone("UTC+8", 8*60*60)
	changes, err := parseTrelloStatusChanges(input, "done", location)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 {
		t.Fatalf("changes = %#v", changes)
	}
	change := changes[0]
	if change.CardID != "card-1" || change.ToListName != "已完成" || change.ChangedAt != "2026-07-29T02:30:00.000Z" || change.ChangedAtLocal != "2026-07-29T10:30:00+08:00" || change.MemberName != "张三" {
		t.Fatalf("change = %#v", change)
	}
}

func TestFormatTrelloAttachmentWritesOriginalBytes(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "homework.jpg")
	body, err := formatTrelloAttachment(trello.AttachmentDownload{
		ID:          "attachment-1",
		Name:        "homework.jpg",
		MIMEType:    "image/jpeg",
		ContentType: "image/jpeg",
		Data:        []byte("jpeg-bytes"),
	}, outFile)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(saved) != "jpeg-bytes" {
		t.Fatalf("saved = %q", saved)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatal(err)
	}
	if result["saved_to"] != outFile || result["base64"] != nil {
		t.Fatalf("result = %#v", result)
	}
}

func TestFormatTrelloAttachmentDefaultsToBase64(t *testing.T) {
	body, err := formatTrelloAttachment(trello.AttachmentDownload{
		ID:   "attachment-1",
		Name: "homework.jpg",
		Data: []byte("jpeg-bytes"),
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatal(err)
	}
	if result["base64"] != "anBlZy1ieXRlcw==" || result["saved_to"] != nil {
		t.Fatalf("result = %#v", result)
	}
}

func TestFilterCardsDueOnMatchesExplicitDate(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*60*60)
	day := time.Date(2026, 7, 10, 0, 0, 0, 0, loc)
	input := []byte(`[
  {"id":"on-day","due":"2026-07-10T15:59:59Z","closed":false},
  {"id":"next-day","due":"2026-07-10T16:00:00Z","closed":false},
  {"id":"day-after","due":"2026-07-11T16:00:00Z","closed":false}
]`)
	output, err := filterCardsDueOn(input, day)
	if err != nil {
		t.Fatal(err)
	}
	var cards []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(output, &cards); err != nil {
		t.Fatal(err)
	}
	if len(cards) != 1 || cards[0].ID != "on-day" {
		t.Fatalf("cards = %s", output)
	}
}

func TestTrelloWithRetryRetriesUntilSuccess(t *testing.T) {
	attempts := 0
	result, err := trelloWithRetry(t.Context(), func(_ context.Context) (string, error) {
		attempts++
		if attempts < 3 {
			return "", errors.New("temporary failure")
		}
		return "ok", nil
	})
	if err != nil || result != "ok" || attempts != 3 {
		t.Fatalf("result = %q, err = %v, attempts = %d", result, err, attempts)
	}
}

func TestTrelloWithRetryGivesUpAfterThreeAttempts(t *testing.T) {
	attempts := 0
	_, err := trelloWithRetry(t.Context(), func(_ context.Context) (string, error) {
		attempts++
		return "", errors.New("always failing")
	})
	if err == nil || attempts != 3 {
		t.Fatalf("err = %v, attempts = %d", err, attempts)
	}
}

func TestSanitizeTrelloFilename(t *testing.T) {
	cases := map[string]string{
		"数学作业.jpg":   "数学作业.jpg",
		" a/b\\c:d ": "a_b_c_d",
		"":           "_",
		"..":         "_",
	}
	for input, want := range cases {
		if got := sanitizeTrelloFilename(input); got != want {
			t.Fatalf("sanitizeTrelloFilename(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestFormatTrelloHomeworkReport(t *testing.T) {
	report := trelloHomeworkReport{
		Date: "2026-08-24",
		Dir:  "homework-2026-08-24",
		Cards: []trelloHomeworkCard{
			{
				Name:     "口算（第37天）",
				ListName: "已完成",
				Attachments: []trelloHomeworkAttachment{
					{
						Name:      "photo.jpg",
						HomeworkPhotoURL: "https://trello.com/1/cards/card-1/attachments/att-1/download/photo.jpg",
						SavedTo:   "homework-2026-08-24/口算（第37天）/photo.webp",
						SizeBytes: 48538,
					},
				},
			},
			{
				Name:     "数学（第37天）",
				ListName: "待完成",
				Attachments: []trelloHomeworkAttachment{
					{Name: "photo2.jpg", Error: "download failed"},
				},
			},
			{Name: "语文（第37天）", ListName: "已完成", Attachments: []trelloHomeworkAttachment{}},
		},
	}
	got := formatTrelloHomeworkReport(report)
	for _, want := range []string{
		"日期: 2026-08-24",
		"卡片数: 3",
		"[已完成] 口算（第37天）",
		"作业照片: photo.jpg (48538 bytes)",
		"本地: homework-2026-08-24/口算（第37天）/photo.webp",
		"链接: https://trello.com/1/cards/card-1/attachments/att-1/download/photo.jpg",
		"作业照片: photo2.jpg (下载失败: download failed)",
		"[已完成] 语文（第37天）\n  (无附件)",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("report missing %q:\n%s", want, got)
		}
	}
}
