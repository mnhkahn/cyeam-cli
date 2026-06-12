package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mnhkahn/cyeam-cli/internal/tv"
)

func renderTVTable(out io.Writer, matches []tv.Match, loc *time.Location, color bool) error {
	t := cliTable{
		Headers: []tableCell{
			{text: "开始时间", visible: "开始时间"},
			{text: "联赛", visible: "联赛"},
			{text: "比赛", visible: "比赛"},
			{text: "阶段", visible: "阶段"},
			{text: "转播源", visible: "转播源"},
		},
		Color: color,
	}
	for _, m := range matches {
		when := formatMatchTime(m.Start.In(loc))
		league := m.LeagueName
		if league == "" {
			league = m.League.DisplayName()
		}
		matchup := fmt.Sprintf("%s vs %s", m.Home.Name, m.Away.Name)
		stage := m.Stage
		if m.Round != "" {
			if stage != "" {
				stage = stage + " " + m.Round
			} else {
				stage = m.Round
			}
		}
		if stage == "" {
			stage = "-"
		}
		text, visible := formatBroadcasts(m.Broadcasts)
		t.Rows = append(t.Rows, []tableCell{
			{text: when, visible: when},
			{text: league, visible: league},
			{text: matchup, visible: matchup},
			{text: stage, visible: stage},
			{text: text, visible: visible},
		})
	}
	return renderTable(out, t)
}

var tvWeekChinese = []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}

func formatMatchTime(t time.Time) string {
	return fmt.Sprintf("%02d-%02d %s %02d:%02d", t.Month(), t.Day(), tvWeekChinese[int(t.Weekday())], t.Hour(), t.Minute())
}

func formatBroadcasts(bs []tv.Broadcast) (string, string) {
	if len(bs) == 0 {
		return "-", "-"
	}
	var textParts, visibleParts []string
	for _, b := range bs {
		if b.URL != "" {
			textParts = append(textParts, terminalHyperlink(b.URL, b.Name))
		} else {
			textParts = append(textParts, b.Name)
		}
		visibleParts = append(visibleParts, b.Name)
	}
	return strings.Join(textParts, "、"), strings.Join(visibleParts, "、")
}

type tvJSONBroadcast struct {
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
	URL  string `json:"url,omitempty"`
}

type tvJSONMatch struct {
	League     string            `json:"league"`
	LeagueName string            `json:"league_name"`
	Stage      string            `json:"stage,omitempty"`
	Round      string            `json:"round,omitempty"`
	Start      string            `json:"start"`
	Home       tv.Team           `json:"home"`
	Away       tv.Team           `json:"away"`
	Venue      string            `json:"venue,omitempty"`
	Status     string            `json:"status"`
	Broadcasts []tvJSONBroadcast `json:"broadcasts,omitempty"`
}

func writeTVJSON(out io.Writer, matches []tv.Match) error {
	view := make([]tvJSONMatch, 0, len(matches))
	for _, m := range matches {
		bs := make([]tvJSONBroadcast, 0, len(m.Broadcasts))
		for _, b := range m.Broadcasts {
			bs = append(bs, tvJSONBroadcast{Name: b.Name, Type: b.Type, URL: b.URL})
		}
		view = append(view, tvJSONMatch{
			League:     string(m.League),
			LeagueName: m.LeagueName,
			Stage:      m.Stage,
			Round:      m.Round,
			Start:      m.Start.Format(time.RFC3339),
			Home:       m.Home,
			Away:       m.Away,
			Venue:      m.Venue,
			Status:     string(m.Status),
			Broadcasts: bs,
		})
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(view)
}
