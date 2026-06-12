package tv

import (
	_ "embed"
	"encoding/json"
	"strings"
)

//go:embed broadcasts.json
var broadcastsJSON []byte

type broadcastRule struct {
	Match struct {
		League      string `json:"league"`
		Stage       string `json:"stage,omitempty"`
		Team        string `json:"team,omitempty"`
		Competition string `json:"competition,omitempty"`
	} `json:"match"`
	Broadcasts []Broadcast `json:"broadcasts"`
}

type broadcastTable struct {
	Rules []broadcastRule `json:"rules"`
}

var defaultBroadcastTable = mustLoadBroadcasts(broadcastsJSON)

func mustLoadBroadcasts(data []byte) broadcastTable {
	var t broadcastTable
	if err := json.Unmarshal(data, &t); err != nil {
		panic("invalid broadcasts.json: " + err.Error())
	}
	return t
}

func (t broadcastTable) lookup(m Match) []Broadcast {
	var matched []broadcastRule
	for _, r := range t.Rules {
		if !ruleMatches(r, m) {
			continue
		}
		matched = append(matched, r)
	}
	if len(matched) == 0 {
		return nil
	}
	specificity := func(r broadcastRule) int {
		s := 0
		if r.Match.League != "" {
			s++
		}
		if r.Match.Stage != "" {
			s += 2
		}
		if r.Match.Team != "" {
			s += 2
		}
		if r.Match.Competition != "" {
			s += 2
		}
		return s
	}
	best := matched[0]
	for _, r := range matched[1:] {
		if specificity(r) > specificity(best) {
			best = r
		}
	}
	return dedupBroadcasts(best.Broadcasts)
}

func ruleMatches(r broadcastRule, m Match) bool {
	if r.Match.League != "" && !strings.EqualFold(r.Match.League, string(m.League)) {
		return false
	}
	if r.Match.Stage != "" && !strings.EqualFold(r.Match.Stage, m.Stage) {
		return false
	}
	if r.Match.Team != "" {
		t := strings.ToLower(r.Match.Team)
		if !strings.Contains(strings.ToLower(m.Home.Name), t) &&
			!strings.Contains(strings.ToLower(m.Away.Name), t) {
			return false
		}
	}
	return true
}

func dedupBroadcasts(in []Broadcast) []Broadcast {
	seen := make(map[string]struct{}, len(in))
	out := make([]Broadcast, 0, len(in))
	for _, b := range in {
		if _, ok := seen[b.Name]; ok {
			continue
		}
		seen[b.Name] = struct{}{}
		out = append(out, b)
	}
	return out
}

func AttachBroadcasts(matches []Match) {
	for i := range matches {
		if len(matches[i].Broadcasts) > 0 {
			continue
		}
		matches[i].Broadcasts = defaultBroadcastTable.lookup(matches[i])
	}
}
