package tv

import (
	"testing"
	"time"
)

func TestFilterAndSortMatches(t *testing.T) {
	start := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)
	match := func(hours int, status Status, team string) Match {
		return Match{League: LeagueNBA, Start: start.Add(time.Duration(hours) * time.Hour), Status: status,
			Home: Team{Name: team, Abbr: "LAL"}, Away: Team{Name: "Denver Nuggets", Abbr: "DEN"},
			Broadcasts: []Broadcast{{Name: "腾讯体育", URL: "https://nba.qq.com/"}}}
	}
	input := []Match{match(3, StatusScheduled, "Los Angeles Lakers"), match(-1, StatusScheduled, "Los Angeles Lakers"),
		match(1, StatusFinished, "Los Angeles Lakers"), match(2, StatusLive, "Los Angeles Lakers"), match(25, StatusScheduled, "Los Angeles Lakers")}
	q := Query{From: start, To: start.Add(24 * time.Hour), Team: "lal", Source: "腾讯"}
	got := FilterAndSort(input, q)
	if len(got) != 2 || got[0].Status != StatusLive || got[1].Start != start.Add(3*time.Hour) {
		t.Fatalf("matches=%+v", got)
	}
	q.IncludeFinished = true
	if len(FilterAndSort(input, q)) != 3 {
		t.Fatal("finished match not included")
	}
	q.Team = "NUGGETS"
	if len(FilterAndSort(input, q)) != 3 {
		t.Fatal("away team case-insensitive filter failed")
	}
	q.Source = "CCTV5"
	if len(FilterAndSort(input, q)) != 0 {
		t.Fatal("source filter failed")
	}
}

func TestBroadcastRulesAndPreservingProvidedBroadcasts(t *testing.T) {
	matches := []Match{
		{League: LeagueNBA, Stage: "Finals"}, {League: LeagueNBA, Stage: "Playoffs"},
		{League: LeagueNBA, Stage: "Regular"}, {League: LeagueWorldCup, Stage: "Final"},
		{League: LeagueCNFootball}, {League: LeagueNBA, Broadcasts: []Broadcast{{Name: "Provided", URL: "https://example.com"}}},
	}
	AttachBroadcasts(matches)
	for i, expected := range []int{3, 2, 1, 4, 2, 1} {
		if len(matches[i].Broadcasts) != expected {
			t.Fatalf("match=%d broadcasts=%+v", i, matches[i].Broadcasts)
		}
		for _, b := range matches[i].Broadcasts {
			if b.URL == "" {
				t.Fatalf("missing URL: %+v", b)
			}
		}
	}
	if matches[5].Broadcasts[0].Name != "Provided" {
		t.Fatal("provided broadcasts were replaced")
	}
}

func TestChineseTeamFilters(t *testing.T) {
	for _, tc := range []struct{ query, name string }{
		{"湖人", "Los Angeles Lakers"}, {"快船", "LA Clippers"},
		{"中国", "China"}, {"中国队", "China PR"}, {"国足", "China"},
	} {
		m := Match{Home: Team{Name: tc.name}}
		if !matchTeam(m, tc.query) {
			t.Errorf("%s did not match %s", tc.query, tc.name)
		}
		if matchTeam(Match{Home: Team{Name: "Unrelated"}}, tc.query) {
			t.Errorf("%s matched unrelated team", tc.query)
		}
	}
}
