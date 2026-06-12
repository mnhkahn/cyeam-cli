package tv

import (
	"context"
	"strings"
)

// CNFootballFetcher 覆盖中国男足 / 女足国家队的国际比赛：
// 友谊赛、世预赛（亚洲区）、亚洲杯、东亚杯、奥运会等多 league 合并去重。
type CNFootballFetcher struct {
	inner *espnFetcher
}

func NewCNFootballFetcher() *CNFootballFetcher {
	return &CNFootballFetcher{
		inner: &espnFetcher{
			League:     LeagueCNFootball,
			LeagueName: "国足",
			LeagueIDs: []string{
				"fifa.friendly",
				"fifa.friendly.w",
				"fifa.worldq.afc",
				"fifa.worldq.afc.w",
				"afc.asian_cup",
				"afc.asian_cup.w",
				"fifa.world",
				"fifa.wwc",
				"fifa.olympics",
				"fifa.w_olympics",
			},
			Filter: cnTeamFilter,
		},
	}
}

func (f *CNFootballFetcher) League() League { return LeagueCNFootball }

func (f *CNFootballFetcher) Fetch(ctx context.Context, q Query) ([]Match, error) {
	return f.inner.fetch(ctx, q)
}

func cnTeamFilter(e espnEvent) bool {
	if len(e.Competitions) == 0 {
		return false
	}
	for _, c := range e.Competitions[0].Competitors {
		name := strings.ToLower(c.Team.DisplayName)
		if strings.Contains(name, "china") {
			return true
		}
	}
	return false
}
