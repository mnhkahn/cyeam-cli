package tv

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type League string

const (
	LeagueNBA         League = "nba"
	LeagueWorldCup    League = "worldcup"
	LeagueCNFootball  League = "cn-football"
)

func KnownLeagues() []League {
	return []League{LeagueNBA, LeagueWorldCup, LeagueCNFootball}
}

func ParseLeague(s string) (League, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "nba":
		return LeagueNBA, nil
	case "worldcup", "world-cup", "fifa", "wc":
		return LeagueWorldCup, nil
	case "cn-football", "cnfootball", "china-football", "guozu", "国足":
		return LeagueCNFootball, nil
	default:
		return "", fmt.Errorf("unknown league %q", s)
	}
}

func (l League) DisplayName() string {
	switch l {
	case LeagueNBA:
		return "NBA"
	case LeagueWorldCup:
		return "世界杯"
	case LeagueCNFootball:
		return "国足"
	default:
		return string(l)
	}
}

type Status string

const (
	StatusScheduled Status = "scheduled"
	StatusLive      Status = "live"
	StatusFinished  Status = "finished"
)

type Team struct {
	Name string `json:"name"`
	Abbr string `json:"abbr,omitempty"`
}

type Broadcast struct {
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
	URL  string `json:"url,omitempty"`
}

type Match struct {
	League     League      `json:"league"`
	LeagueName string      `json:"league_name"`
	Stage      string      `json:"stage,omitempty"`
	Round      string      `json:"round,omitempty"`
	Start      time.Time   `json:"start"`
	Home       Team        `json:"home"`
	Away       Team        `json:"away"`
	HomeScore  string      `json:"home_score,omitempty"`
	AwayScore  string      `json:"away_score,omitempty"`
	Venue      string      `json:"venue,omitempty"`
	Status     Status      `json:"status"`
	Broadcasts []Broadcast `json:"broadcasts,omitempty"`
}

type Query struct {
	Leagues          []League
	From             time.Time
	To               time.Time
	Team             string
	Source           string
	IncludeFinished  bool
	Location         *time.Location
}

type Fetcher interface {
	League() League
	Fetch(ctx context.Context, q Query) ([]Match, error)
}

type FetchResult struct {
	League  League
	Matches []Match
	Err     error
}

func FetchAll(ctx context.Context, fetchers []Fetcher, q Query) []FetchResult {
	results := make([]FetchResult, len(fetchers))
	var wg sync.WaitGroup
	for i, f := range fetchers {
		wg.Add(1)
		go func(i int, f Fetcher) {
			defer wg.Done()
			ms, err := f.Fetch(ctx, q)
			results[i] = FetchResult{League: f.League(), Matches: ms, Err: err}
		}(i, f)
	}
	wg.Wait()
	return results
}

func FilterAndSort(matches []Match, q Query) []Match {
	out := make([]Match, 0, len(matches))
	for _, m := range matches {
		if !q.From.IsZero() && m.Start.Before(q.From) {
			continue
		}
		if !q.To.IsZero() && m.Start.After(q.To) {
			continue
		}
		if !q.IncludeFinished && m.Status == StatusFinished {
			continue
		}
		if q.Team != "" && !matchTeam(m, q.Team) {
			continue
		}
		if q.Source != "" && !matchSource(m, q.Source) {
			continue
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start.Before(out[j].Start) })
	return out
}

func matchTeam(m Match, needle string) bool {
	needle = strings.ToLower(needle)
	for _, t := range []Team{m.Home, m.Away} {
		if strings.Contains(strings.ToLower(t.Name), needle) ||
			strings.Contains(strings.ToLower(t.Abbr), needle) {
			return true
		}
	}
	return false
}

func matchSource(m Match, needle string) bool {
	needle = strings.ToLower(needle)
	for _, b := range m.Broadcasts {
		if strings.Contains(strings.ToLower(b.Name), needle) {
			return true
		}
	}
	return false
}

var ErrSourceUnavailable = errors.New("data source unavailable")
