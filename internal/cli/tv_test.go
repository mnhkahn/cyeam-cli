package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mnhkahn/cyeam-cli/internal/tv"
	"github.com/spf13/cobra"
)

type fakeTVFetcher struct {
	league  tv.League
	matches []tv.Match
	err     error
	query   *tv.Query
}

func (f fakeTVFetcher) League() tv.League { return f.league }
func (f fakeTVFetcher) Fetch(_ context.Context, q tv.Query) ([]tv.Match, error) {
	if f.query != nil {
		*f.query = q
	}
	return f.matches, f.err
}

func TestTVSubcommandsAndFlags(t *testing.T) {
	now := time.Date(2026, 10, 15, 20, 0, 0, 0, time.UTC)
	cases := []struct {
		name     string
		command  func(Dependencies, tvDeps) *cobra.Command
		args     []string
		from, to string
		finished bool
	}{
		{"list defaults", newTVListCommand, []string{"--json"}, "2026-10-16", "2026-10-18", false},
		{"today defaults", newTVTodayCommand, []string{"--json"}, "2026-10-16", "2026-10-16", true},
		{"today explicit excludes finished", newTVTodayCommand, []string{"--json", "--include-finished=false"}, "2026-10-16", "2026-10-16", false},
		{"today UTC", newTVTodayCommand, []string{"--json", "--tz", "UTC"}, "2026-10-15", "2026-10-15", true},
		{"tomorrow", newTVTomorrowCommand, []string{"--json"}, "2026-10-17", "2026-10-17", false},
		{"next defaults", newTVNextCommand, []string{"--json"}, "2026-10-16", "2026-10-29", false},
		{"next explicit days", newTVNextCommand, []string{"--json", "--days", "2"}, "2026-10-16", "2026-10-17", false},
		{"days cap", newTVListCommand, []string{"--json", "--days", "100"}, "2026-10-16", "2026-10-29", false},
		{"explicit dates", newTVListCommand, []string{"--json", "--from", "2026-03-01", "--to", "2026-03-03", "--include-finished"}, "2026-03-01", "2026-03-03", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := new(bytes.Buffer)
			var q tv.Query
			td := tvDeps{Now: func() time.Time { return now }, Fetchers: func() []tv.Fetcher {
				return []tv.Fetcher{fakeTVFetcher{league: tv.LeagueNBA, query: &q}}
			}}
			cmd := tc.command(Dependencies{Stdout: out}, td)
			cmd.SetArgs(tc.args)
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			if q.From.Format(time.DateOnly) != tc.from || q.To.Format(time.DateOnly) != tc.to || q.IncludeFinished != tc.finished {
				t.Fatalf("query=%+v", q)
			}
			if strings.TrimSpace(out.String()) != "[]" {
				t.Fatalf("output=%q", out)
			}
		})
	}
}

func TestTVLeagueSelectionDoesNotFetchUnrequestedSources(t *testing.T) {
	var q tv.Query
	td := tvDeps{Fetchers: func() []tv.Fetcher {
		return []tv.Fetcher{fakeTVFetcher{league: tv.LeagueNBA, query: &q}, fakeTVFetcher{league: tv.LeagueWorldCup, err: errors.New("must not be fetched")}}
	}}
	out := new(bytes.Buffer)
	cmd := newTVListCommand(Dependencies{Stdout: out}, td)
	cmd.SetArgs([]string{"--league", "nba", "--team", "LAL", "--source", "腾讯", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if len(q.Leagues) != 1 || q.Leagues[0] != tv.LeagueNBA || q.Team != "LAL" || q.Source != "腾讯" {
		t.Fatalf("query=%+v", q)
	}
}

func TestTVNextSortsAndIncludesWarningsInText(t *testing.T) {
	now := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)
	out := new(bytes.Buffer)
	td := tvDeps{Now: func() time.Time { return now }, Fetchers: func() []tv.Fetcher {
		return []tv.Fetcher{fakeTVFetcher{league: tv.LeagueNBA, err: errors.New("partial failure"), matches: []tv.Match{
			{League: tv.LeagueNBA, Start: now.Add(2 * time.Hour), Home: tv.Team{Name: "Later"}},
			{League: tv.LeagueNBA, Start: now.Add(time.Hour), Home: tv.Team{Name: "Earlier"}},
		}}}
	}}
	if err := runTVNext(context.Background(), Dependencies{Stdout: out}, td, tvFlags{noColor: true}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "partial failure") || !strings.Contains(out.String(), "Earlier") || strings.Contains(out.String(), "Later") {
		t.Fatalf("output=%s", out)
	}
}

func TestTVFailuresAreErrorsInAllOutputModes(t *testing.T) {
	for _, next := range []bool{false, true} {
		for _, jsonOut := range []bool{false, true} {
			out := new(bytes.Buffer)
			deps := Dependencies{Stdout: out}
			td := tvDeps{Fetchers: func() []tv.Fetcher {
				return []tv.Fetcher{fakeTVFetcher{league: tv.LeagueNBA, err: errors.New("http 403")}}
			}}
			f := tvFlags{jsonOut: jsonOut}
			var err error
			if next {
				err = runTVNext(context.Background(), deps, td, f)
			} else {
				err = runTVList(context.Background(), deps, td, f)
			}
			if !errors.Is(err, tv.ErrSourceUnavailable) || !strings.Contains(err.Error(), "http 403") || out.Len() != 0 {
				t.Fatalf("next=%v json=%v err=%v output=%q", next, jsonOut, err, out.String())
			}
		}
	}
}

func TestTVPartialResultsRemainVisibleWithWarnings(t *testing.T) {
	now := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)
	for _, next := range []bool{false, true} {
		out := new(bytes.Buffer)
		td := tvDeps{Now: func() time.Time { return now }, Fetchers: func() []tv.Fetcher {
			return []tv.Fetcher{fakeTVFetcher{league: tv.LeagueNBA,
				matches: []tv.Match{{League: tv.LeagueNBA, Start: now.Add(time.Hour), Status: tv.StatusScheduled}},
				err:     errors.New("one date failed")}}
		}}
		f := tvFlags{jsonOut: true}
		var err error
		if next {
			err = runTVNext(context.Background(), Dependencies{Stdout: out}, td, f)
		} else {
			err = runTVList(context.Background(), Dependencies{Stdout: out}, td, f)
		}
		var result struct {
			Matches  []tvJSONMatch
			Partial  bool
			Warnings []struct {
				League  string
				Message string
			}
		}
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(out.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if len(result.Matches) != 1 || !result.Partial || len(result.Warnings) != 1 || result.Warnings[0].Message != "one date failed" {
			t.Fatalf("next=%v result=%s", next, out)
		}
	}
}

func TestTVNextEmptyJSONIsAnArray(t *testing.T) {
	out := new(bytes.Buffer)
	td := tvDeps{Fetchers: func() []tv.Fetcher { return []tv.Fetcher{fakeTVFetcher{league: tv.LeagueNBA}} }}
	if err := runTVNext(context.Background(), Dependencies{Stdout: out}, td, tvFlags{jsonOut: true}); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) != "[]" {
		t.Fatalf("output=%q", out.String())
	}
}

func TestTVDaysAndDateValidation(t *testing.T) {
	td := tvDeps{Now: func() time.Time { return time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC) }}
	q, _, err := buildTVQuery(td, tvFlags{days: 3})
	if err != nil {
		t.Fatal(err)
	}
	if q.To.Sub(q.From) != 72*time.Hour-time.Nanosecond {
		t.Fatalf("window=%v", q.To.Sub(q.From))
	}
	if _, _, err := buildTVQuery(td, tvFlags{from: "2026-10-16", to: "2026-10-15"}); err == nil {
		t.Fatal("reversed dates accepted")
	}
}
