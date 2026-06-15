package tv

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const nbaScheduleURL = "https://cdn.nba.com/static/json/staticData/scheduleLeagueV2.json"

type NBAFetcher struct {
	HTTPClient *http.Client
	URL        string
}

func NewNBAFetcher() *NBAFetcher {
	return &NBAFetcher{
		HTTPClient: &http.Client{Timeout: 60 * time.Second},
		URL:        nbaScheduleURL,
	}
}

func (f *NBAFetcher) League() League { return LeagueNBA }

type nbaScheduleResp struct {
	LeagueSchedule struct {
		GameDates []struct {
			Games []nbaGame `json:"games"`
		} `json:"gameDates"`
	} `json:"leagueSchedule"`
}

type nbaGame struct {
	GameID         string `json:"gameId"`
	GameCode       string `json:"gameCode"`
	GameStatus     int    `json:"gameStatus"`
	GameStatusText string `json:"gameStatusText"`
	GameDateTimeUTC string `json:"gameDateTimeUTC"`
	GameLabel      string `json:"gameLabel"`
	GameSubLabel   string `json:"gameSubLabel"`
	SeriesText     string `json:"seriesText"`
	WeekName       string `json:"weekName"`
	HomeTeam       nbaTeam `json:"homeTeam"`
	AwayTeam       nbaTeam `json:"awayTeam"`
	ArenaName      string  `json:"arenaName"`
	ArenaCity      string  `json:"arenaCity"`
}

type nbaTeam struct {
	TeamID       int    `json:"teamId"`
	TeamCity     string `json:"teamCity"`
	TeamName     string `json:"teamName"`
	TeamTricode  string `json:"teamTricode"`
	Score        int    `json:"score"`
}

func (f *NBAFetcher) Fetch(ctx context.Context, q Query) ([]Match, error) {
	url := f.URL
	if url == "" {
		url = nbaScheduleURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (cyeam-cli)")
	req.Header.Set("Referer", "https://www.nba.com/")
	req.Header.Set("Origin", "https://www.nba.com")
	resp, err := f.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("nba: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("nba: http %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return parseNBASchedule(body, q)
}

func parseNBASchedule(body []byte, q Query) ([]Match, error) {
	var sched nbaScheduleResp
	if err := json.Unmarshal(body, &sched); err != nil {
		return nil, fmt.Errorf("nba decode: %w", err)
	}
	var out []Match
	for _, gd := range sched.LeagueSchedule.GameDates {
		for _, g := range gd.Games {
			if g.HomeTeam.TeamName == "" && g.AwayTeam.TeamName == "" {
				continue
			}
			start, err := time.Parse("2006-01-02T15:04:05Z", g.GameDateTimeUTC)
			if err != nil {
				continue
			}
			if !q.From.IsZero() && start.Before(q.From.Add(-12*time.Hour)) {
				continue
			}
			if !q.To.IsZero() && start.After(q.To.Add(12*time.Hour)) {
				continue
			}
			stage := nbaStage(g.GameLabel, g.SeriesText)
			out = append(out, Match{
				League:     LeagueNBA,
				LeagueName: nbaLeagueName(stage),
				Stage:      stage,
				Round:      strings.TrimSpace(g.GameSubLabel),
				Start:      start,
				Home: Team{
					Name: nbaTeamName(g.HomeTeam),
					Abbr: g.HomeTeam.TeamTricode,
				},
				Away: Team{
					Name: nbaTeamName(g.AwayTeam),
					Abbr: g.AwayTeam.TeamTricode,
				},
				HomeScore: strconv.Itoa(g.HomeTeam.Score),
				AwayScore: strconv.Itoa(g.AwayTeam.Score),
				Status:    nbaStatus(g.GameStatus),
			})
		}
	}
	return out, nil
}

func nbaTeamName(t nbaTeam) string {
	if name, ok := nbaTeamCN[t.TeamTricode]; ok {
		return name
	}
	if t.TeamCity != "" && t.TeamName != "" {
		return t.TeamCity + " " + t.TeamName
	}
	if t.TeamName != "" {
		return t.TeamName
	}
	return t.TeamTricode
}

func nbaStage(label, series string) string {
	l := strings.ToLower(label)
	switch {
	case strings.Contains(l, "finals") && !strings.Contains(l, "conference"):
		return "Finals"
	case strings.Contains(l, "conference finals") || strings.Contains(l, "semifinals") || strings.Contains(l, "first round") || strings.Contains(l, "playoffs"):
		return "Playoffs"
	case strings.Contains(l, "play-in") || strings.Contains(l, "play in"):
		return "Play-In"
	case strings.Contains(l, "preseason"):
		return "Preseason"
	case strings.Contains(l, "all-star") || strings.Contains(l, "all star"):
		return "All-Star"
	case strings.Contains(l, "in-season") || strings.Contains(l, "cup"):
		return "Cup"
	}
	if series != "" {
		return "Playoffs"
	}
	return "Regular"
}

func nbaLeagueName(stage string) string {
	switch stage {
	case "Finals":
		return "NBA 总决赛"
	case "Playoffs":
		return "NBA 季后赛"
	case "Play-In":
		return "NBA 附加赛"
	case "Preseason":
		return "NBA 季前赛"
	case "All-Star":
		return "NBA 全明星"
	case "Cup":
		return "NBA 杯赛"
	default:
		return "NBA 常规赛"
	}
}

func nbaStatus(s int) Status {
	switch s {
	case 1:
		return StatusScheduled
	case 2:
		return StatusLive
	case 3:
		return StatusFinished
	default:
		return StatusScheduled
	}
}

var nbaTeamCN = map[string]string{
	"ATL": "老鹰", "BOS": "凯尔特人", "BKN": "篮网", "CHA": "黄蜂",
	"CHI": "公牛", "CLE": "骑士", "DAL": "独行侠", "DEN": "掘金",
	"DET": "活塞", "GSW": "勇士", "HOU": "火箭", "IND": "步行者",
	"LAC": "快船", "LAL": "湖人", "MEM": "灰熊", "MIA": "热火",
	"MIL": "雄鹿", "MIN": "森林狼", "NOP": "鹈鹕", "NYK": "尼克斯",
	"OKC": "雷霆", "ORL": "魔术", "PHI": "76 人", "PHX": "太阳",
	"POR": "开拓者", "SAC": "国王", "SAS": "马刺", "TOR": "猛龙",
	"UTA": "爵士", "WAS": "奇才",
}
