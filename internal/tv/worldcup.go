package tv

import "context"

type WorldCupFetcher struct {
	inner *espnFetcher
}

func NewWorldCupFetcher() *WorldCupFetcher {
	return &WorldCupFetcher{
		inner: &espnFetcher{
			League:     LeagueWorldCup,
			LeagueName: "世界杯",
			LeagueIDs:  []string{"fifa.world", "fifa.wwc"},
		},
	}
}

func (f *WorldCupFetcher) League() League { return LeagueWorldCup }

func (f *WorldCupFetcher) Fetch(ctx context.Context, q Query) ([]Match, error) {
	return f.inner.fetch(ctx, q)
}
