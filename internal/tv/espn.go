package tv

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ESPN site API. league codes:
//   fifa.world           - FIFA World Cup (men)
//   fifa.wwc             - FIFA Women's World Cup
//   fifa.worldq.afc      - World Cup Qualifying - AFC (亚洲区)
//   fifa.friendly        - international friendlies (men)
//   fifa.friendly.w      - international friendlies (women)
//   fifa.olympics        - Olympics men
//   fifa.w_olympics      - Olympics women
const espnScoreboardBase = "https://site.api.espn.com/apis/site/v2/sports/soccer"

type espnTime struct {
	time.Time
}

func (t *espnTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		return nil
	}
	layouts := []string{
		"2006-01-02T15:04Z",
		"2006-01-02T15:04:05Z",
		time.RFC3339,
	}
	var lastErr error
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, s)
		if err == nil {
			t.Time = parsed
			return nil
		}
		lastErr = err
	}
	return lastErr
}

type espnFetcher struct {
	HTTPClient *http.Client
	BaseURL    string
	League     League
	LeagueName string
	LeagueIDs  []string
	Filter     func(espnEvent) bool
}

func (f *espnFetcher) league() League { return f.League }

func (f *espnFetcher) base() string {
	if f.BaseURL != "" {
		return f.BaseURL
	}
	return espnScoreboardBase
}

func (f *espnFetcher) fetch(ctx context.Context, q Query) ([]Match, error) {
	dates := espnDateRanges(q)
	var all []Match
	var firstErr error
	successCount := 0
	totalCalls := 0
	for _, leagueID := range f.LeagueIDs {
		for _, d := range dates {
			totalCalls++
			matches, err := f.fetchOne(ctx, leagueID, d)
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			successCount++
			all = append(all, matches...)
		}
	}
	if successCount == 0 && firstErr != nil {
		return nil, firstErr
	}
	return dedupMatches(all), nil
}

func (f *espnFetcher) fetchOne(ctx context.Context, leagueID, dates string) ([]Match, error) {
	u := fmt.Sprintf("%s/%s/scoreboard", f.base(), leagueID)
	if dates != "" {
		u += "?dates=" + url.QueryEscape(dates)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "cyeam-cli")
	resp, err := f.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("espn %s: %w", leagueID, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("espn %s: http %d", leagueID, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return f.parse(body)
}

func (f *espnFetcher) client() *http.Client {
	if f.HTTPClient != nil {
		return f.HTTPClient
	}
	return &http.Client{Timeout: 10 * time.Second}
}

type espnScoreboard struct {
	Events []espnEvent `json:"events"`
}

type espnEvent struct {
	ID         string    `json:"id"`
	Date       espnTime  `json:"date"`
	Name       string    `json:"name"`
	ShortName  string    `json:"shortName"`
	Season     struct {
		Type int    `json:"type"`
		Slug string `json:"slug"`
	} `json:"season"`
	Status struct {
		Type struct {
			State       string `json:"state"`
			Completed   bool   `json:"completed"`
			Description string `json:"description"`
		} `json:"type"`
	} `json:"status"`
	Competitions []struct {
		Venue struct {
			FullName string `json:"fullName"`
		} `json:"venue"`
		Notes []struct {
			Headline string `json:"headline"`
		} `json:"notes"`
		Competitors []espnCompetitor `json:"competitors"`
	} `json:"competitions"`
}

type espnCompetitor struct {
	HomeAway string `json:"homeAway"`
	Team     struct {
		ID           string `json:"id"`
		DisplayName  string `json:"displayName"`
		Name         string `json:"name"`
		Abbreviation string `json:"abbreviation"`
	} `json:"team"`
}

func (f *espnFetcher) parse(body []byte) ([]Match, error) {
	var sb espnScoreboard
	if err := json.Unmarshal(body, &sb); err != nil {
		return nil, fmt.Errorf("espn decode: %w", err)
	}
	var out []Match
	for _, e := range sb.Events {
		if f.Filter != nil && !f.Filter(e) {
			continue
		}
		if len(e.Competitions) == 0 || len(e.Competitions[0].Competitors) < 2 {
			continue
		}
		comp := e.Competitions[0]
		home, away := splitCompetitors(comp.Competitors)
		stage := ""
		if len(comp.Notes) > 0 {
			stage = comp.Notes[0].Headline
		}
		out = append(out, Match{
			League:     f.League,
			LeagueName: f.LeagueName,
			Stage:      stage,
			Start:      e.Date.Time,
			Home:       Team{Name: localizeTeam(home.Team.DisplayName), Abbr: home.Team.Abbreviation},
			Away:       Team{Name: localizeTeam(away.Team.DisplayName), Abbr: away.Team.Abbreviation},
			Venue:      comp.Venue.FullName,
			Status:     espnStatus(e.Status.Type.State),
		})
	}
	return out, nil
}

func splitCompetitors(cs []espnCompetitor) (home, away espnCompetitor) {
	for _, c := range cs {
		if c.HomeAway == "home" {
			home = c
		} else {
			away = c
		}
	}
	if home.Team.ID == "" {
		home = cs[0]
	}
	if away.Team.ID == "" && len(cs) > 1 {
		away = cs[1]
	}
	return
}

func espnStatus(state string) Status {
	switch strings.ToLower(state) {
	case "pre":
		return StatusScheduled
	case "in":
		return StatusLive
	case "post":
		return StatusFinished
	}
	return StatusScheduled
}

func espnDateRanges(q Query) []string {
	if q.From.IsZero() && q.To.IsZero() {
		return []string{""}
	}
	from := q.From
	to := q.To
	if from.IsZero() {
		from = time.Now().UTC()
	}
	if to.IsZero() {
		to = from.AddDate(0, 0, 7)
	}
	if !to.After(from) {
		return []string{from.Format("20060102")}
	}
	return []string{from.Format("20060102") + "-" + to.Format("20060102")}
}

func dedupMatches(in []Match) []Match {
	seen := make(map[string]struct{}, len(in))
	out := make([]Match, 0, len(in))
	for _, m := range in {
		key := fmt.Sprintf("%s|%s|%s|%s", m.League, m.Start.UTC().Format(time.RFC3339), m.Home.Name, m.Away.Name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, m)
	}
	return out
}

func localizeTeam(name string) string {
	if cn, ok := teamLocalize[name]; ok {
		return cn
	}
	return name
}

var teamLocalize = map[string]string{
	"China PR":     "中国",
	"China":        "中国",
	"Chinese Taipei": "中国台北",
	"Hong Kong":    "中国香港",
	"Argentina":    "阿根廷",
	"Brazil":       "巴西",
	"France":       "法国",
	"Germany":      "德国",
	"Spain":        "西班牙",
	"Portugal":     "葡萄牙",
	"England":      "英格兰",
	"Italy":        "意大利",
	"Netherlands":  "荷兰",
	"Belgium":      "比利时",
	"Croatia":      "克罗地亚",
	"Morocco":      "摩洛哥",
	"Mexico":       "墨西哥",
	"Uruguay":      "乌拉圭",
	"Colombia":     "哥伦比亚",
	"Japan":        "日本",
	"South Korea":  "韩国",
	"Korea Republic": "韩国",
	"Saudi Arabia": "沙特阿拉伯",
	"Iran":         "伊朗",
	"Australia":    "澳大利亚",
	"United States": "美国",
	"USA":          "美国",
	"Canada":       "加拿大",
	"Qatar":        "卡塔尔",
	"Senegal":      "塞内加尔",
	"Ghana":        "加纳",
	"Cameroon":     "喀麦隆",
	"Tunisia":      "突尼斯",
	"Switzerland":  "瑞士",
	"Denmark":      "丹麦",
	"Poland":       "波兰",
	"Serbia":       "塞尔维亚",
	"Wales":        "威尔士",
	"Ecuador":      "厄瓜多尔",
	"Costa Rica":   "哥斯达黎加",
}
