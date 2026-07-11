package cli

import (
	"encoding/json"
	"testing"
	"time"
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
