package tv

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const espnFixture = `{"events":[{"id":"1","date":"2026-10-15T00:30Z","season":{"type":1,"slug":"preseason"},"status":{"type":{"state":"pre"}},"competitions":[{"competitors":[{"homeAway":"home","score":"0","team":{"id":"1","displayName":"Houston Rockets","abbreviation":"HOU"}},{"homeAway":"away","score":"0","team":{"id":"2","displayName":"Oklahoma City Thunder","abbreviation":"OKC"}}]}]}]}`

func testQuery() Query {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	from := time.Date(2026, 10, 15, 0, 0, 0, 0, loc)
	return Query{From: from, To: from.AddDate(0, 0, 1).Add(-time.Nanosecond)}
}

func TestESPNQueriesSingleDatesAndDeduplicates(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		date := r.URL.Query().Get("dates")
		if len(date) != 8 || strings.Contains(date, "-") {
			t.Errorf("expected single date, got %q", date)
			http.Error(w, "bad dates", http.StatusBadRequest)
			return
		}
		calls.Add(1)
		fmt.Fprint(w, espnFixture)
	}))
	defer srv.Close()
	f := &espnFetcher{BaseURL: srv.URL, HTTPClient: srv.Client(), League: LeagueNBA, LeagueIDs: []string{"nba"}}
	matches, err := f.fetch(context.Background(), testQuery())
	if err != nil || len(matches) != 1 || calls.Load() != 3 {
		t.Fatalf("matches=%d calls=%d err=%v", len(matches), calls.Load(), err)
	}
	if matches[0].Stage != "Preseason" || matches[0].Home.Abbr != "HOU" || matches[0].Status != StatusScheduled {
		t.Fatalf("unexpected NBA match: %+v", matches[0])
	}
	if got := FilterAndSort(matches, testQuery()); len(got) != 1 || got[0].Start.In(testQuery().From.Location()).Hour() != 8 {
		t.Fatalf("timezone filtering lost match: %+v", got)
	}
}

func TestESPNKeepsMatchesAndReportsPartialFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("dates") == "20261014" {
			http.Error(w, "unavailable", http.StatusBadGateway)
			return
		}
		fmt.Fprint(w, espnFixture)
	}))
	defer srv.Close()
	f := &espnFetcher{BaseURL: srv.URL, HTTPClient: srv.Client(), LeagueIDs: []string{"test"}}
	matches, err := f.fetch(context.Background(), testQuery())
	if len(matches) != 1 || err == nil || !strings.Contains(err.Error(), "1/3") {
		t.Fatalf("matches=%d err=%v", len(matches), err)
	}
}

func TestESPNCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("cancelled query reached server")
	}))
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f := &espnFetcher{BaseURL: srv.URL, HTTPClient: srv.Client(), LeagueIDs: []string{"test"}}
	_, err := f.fetch(ctx, testQuery())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
}

func TestNBAFallsBackOnForbidden(t *testing.T) {
	var primaryCalls, fallbackCalls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/schedule" {
			primaryCalls.Add(1)
			http.Error(w, "Access Denied", http.StatusForbidden)
			return
		}
		if r.URL.Path != "/nba/scoreboard" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		fallbackCalls.Add(1)
		fmt.Fprint(w, espnFixture)
	}))
	defer srv.Close()
	f := &NBAFetcher{HTTPClient: srv.Client(), URL: srv.URL + "/schedule", FallbackURL: srv.URL}
	matches, err := f.Fetch(context.Background(), testQuery())
	if err != nil || len(matches) != 1 || primaryCalls.Load() != 1 || fallbackCalls.Load() != 3 {
		t.Fatalf("matches=%d primary=%d fallback=%d err=%v", len(matches), primaryCalls.Load(), fallbackCalls.Load(), err)
	}
}

func TestNBADoesNotFallbackWhenScheduleSucceeds(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/schedule" {
			t.Errorf("unexpected fallback: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"leagueSchedule":{"gameDates":[]}}`)
	}))
	defer srv.Close()
	f := &NBAFetcher{HTTPClient: srv.Client(), URL: srv.URL + "/schedule", FallbackURL: srv.URL}
	if _, err := f.Fetch(context.Background(), testQuery()); err != nil {
		t.Fatal(err)
	}
}

func TestNBAReportsBothSourceFailures(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusForbidden)
	}))
	defer srv.Close()
	f := &NBAFetcher{HTTPClient: srv.Client(), URL: srv.URL + "/schedule", FallbackURL: srv.URL}
	_, err := f.Fetch(context.Background(), testQuery())
	if err == nil || !strings.Contains(err.Error(), "nba: http 403") || !strings.Contains(err.Error(), "nba fallback") {
		t.Fatalf("err=%v", err)
	}
}

func TestWorldCupFinalStageAndBroadcasts(t *testing.T) {
	f := &espnFetcher{League: LeagueWorldCup}
	fixture := strings.Replace(espnFixture, `"slug":"preseason"`, `"slug":"final"`, 1)
	matches, err := f.parse([]byte(fixture))
	if err != nil || len(matches) != 1 || matches[0].Stage != "Final" {
		t.Fatalf("matches=%+v err=%v", matches, err)
	}
	AttachBroadcasts(matches)
	if !matchSource(matches[0], "CCTV1") {
		t.Fatalf("final rule did not match: %+v", matches[0].Broadcasts)
	}
}

func TestNBAPlayoffStageClassification(t *testing.T) {
	for _, label := range []string{"Conference Finals", "East Conf Finals", "West Semifinals", "First Round", "Playoffs"} {
		if got := nbaStage(label, ""); got != "Playoffs" {
			t.Errorf("%s -> %s", label, got)
		}
	}
	if got := nbaStage("NBA Finals", ""); got != "Finals" {
		t.Errorf("NBA Finals -> %s", got)
	}
}

func TestNBAFallsBackOnPrimaryTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/schedule" {
			<-r.Context().Done()
			return
		}
		fmt.Fprint(w, espnFixture)
	}))
	defer srv.Close()
	client := srv.Client()
	client.Timeout = 100 * time.Millisecond
	f := &NBAFetcher{HTTPClient: client, URL: srv.URL + "/schedule", FallbackURL: srv.URL}
	matches, err := f.Fetch(context.Background(), testQuery())
	if err != nil || len(matches) != 1 {
		t.Fatalf("matches=%d err=%v", len(matches), err)
	}
}

func TestESPNRejectsInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "not JSON")
	}))
	defer srv.Close()
	f := &espnFetcher{BaseURL: srv.URL, HTTPClient: srv.Client(), LeagueIDs: []string{"test"}}
	matches, err := f.fetch(context.Background(), testQuery())
	if len(matches) != 0 || err == nil || !strings.Contains(err.Error(), "espn decode") {
		t.Fatalf("matches=%d err=%v", len(matches), err)
	}
}
