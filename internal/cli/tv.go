package cli

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mnhkahn/cyeam-cli/internal/tv"
	"github.com/spf13/cobra"
)

type tvFlags struct {
	leagues         []string
	team            string
	source          string
	days            int
	from            string
	to              string
	tz              string
	jsonOut         bool
	includeFinished bool
	noColor         bool
}

type tvDeps struct {
	Fetchers func() []tv.Fetcher
	Now      func() time.Time
}

func newTVCommand(deps Dependencies) *cobra.Command {
	td := tvDeps{
		Fetchers: defaultTVFetchers,
		Now:      deps.Now,
	}
	cmd := &cobra.Command{
		Use:   "tv",
		Short: "TV / streaming schedule for NBA, World Cup, China national football",
	}
	cmd.AddCommand(newTVListCommand(deps, td))
	cmd.AddCommand(newTVTodayCommand(deps, td))
	cmd.AddCommand(newTVTomorrowCommand(deps, td))
	cmd.AddCommand(newTVNextCommand(deps, td))
	return cmd
}

func defaultTVFetchers() []tv.Fetcher {
	return []tv.Fetcher{
		tv.NewNBAFetcher(),
		tv.NewWorldCupFetcher(),
		tv.NewCNFootballFetcher(),
	}
}

func tvAddFlags(cmd *cobra.Command, f *tvFlags) {
	cmd.Flags().StringSliceVarP(&f.leagues, "league", "l", nil, "filter by league: nba, worldcup, cn-football (repeatable)")
	cmd.Flags().StringVarP(&f.team, "team", "t", "", "filter by team name or abbreviation")
	cmd.Flags().StringVar(&f.source, "source", "", "filter by broadcaster name (e.g. CCTV5)")
	cmd.Flags().IntVarP(&f.days, "days", "d", 3, "show matches in the next N days (max 14)")
	cmd.Flags().StringVar(&f.from, "from", "", "start date YYYY-MM-DD (overrides --days start)")
	cmd.Flags().StringVar(&f.to, "to", "", "end date YYYY-MM-DD")
	cmd.Flags().StringVar(&f.tz, "tz", "Asia/Shanghai", "timezone for display")
	cmd.Flags().BoolVar(&f.jsonOut, "json", false, "output JSON")
	cmd.Flags().BoolVar(&f.includeFinished, "include-finished", false, "include finished matches")
	cmd.Flags().BoolVar(&f.noColor, "no-color", false, "disable color output")
}

func newTVListCommand(deps Dependencies, td tvDeps) *cobra.Command {
	f := &tvFlags{}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List upcoming matches",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTVList(cmd.Context(), deps, td, *f)
		},
	}
	tvAddFlags(cmd, f)
	return cmd
}

func newTVTodayCommand(deps Dependencies, td tvDeps) *cobra.Command {
	f := &tvFlags{}
	cmd := &cobra.Command{
		Use:   "today",
		Short: "List today's matches",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fc := *f
			now := time.Now
			if td.Now != nil {
				now = td.Now
			}
			loc := time.UTC
			if l, err := time.LoadLocation(f.tz); err == nil {
				loc = l
			}
			today := now().In(loc).Format(time.DateOnly)
			fc.from = today
			fc.to = today
			return runTVList(cmd.Context(), deps, td, fc)
		},
	}
	tvAddFlags(cmd, f)
	return cmd
}

func newTVTomorrowCommand(deps Dependencies, td tvDeps) *cobra.Command {
	f := &tvFlags{}
	cmd := &cobra.Command{
		Use:   "tomorrow",
		Short: "List tomorrow's matches",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fc := *f
			now := time.Now
			if td.Now != nil {
				now = td.Now
			}
			loc := time.UTC
			if l, err := time.LoadLocation(f.tz); err == nil {
				loc = l
			}
			tomorrow := now().In(loc).AddDate(0, 0, 1).Format(time.DateOnly)
			fc.from = tomorrow
			fc.to = tomorrow
			return runTVList(cmd.Context(), deps, td, fc)
		},
	}
	tvAddFlags(cmd, f)
	return cmd
}

func newTVNextCommand(deps Dependencies, td tvDeps) *cobra.Command {
	f := &tvFlags{}
	cmd := &cobra.Command{
		Use:   "next",
		Short: "Show the next upcoming match",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fc := *f
			if !cmd.Flags().Changed("days") {
				fc.days = 14
			}
			return runTVNext(cmd.Context(), deps, td, fc)
		},
	}
	tvAddFlags(cmd, f)
	return cmd
}

func runTVList(ctx context.Context, deps Dependencies, td tvDeps, f tvFlags) error {
	q, loc, err := buildTVQuery(td, f)
	if err != nil {
		return err
	}
	matches, errs := collectMatches(ctx, td, q)
	tv.AttachBroadcasts(matches)
	matches = tv.FilterAndSort(matches, q)

	if f.jsonOut {
		return writeTVJSON(deps.Stdout, matches)
	}
	for _, e := range errs {
		fmt.Fprintf(deps.Stdout, "warning: %s data unavailable: %v\n", e.League, e.Err)
	}
	if len(matches) == 0 {
		_, err := io.WriteString(deps.Stdout, "暂无符合条件的比赛。\n")
		return err
	}
	return renderTVTable(deps.Stdout, matches, loc, !f.noColor)
}

func runTVNext(ctx context.Context, deps Dependencies, td tvDeps, f tvFlags) error {
	q, loc, err := buildTVQuery(td, f)
	if err != nil {
		return err
	}
	matches, errs := collectMatches(ctx, td, q)
	tv.AttachBroadcasts(matches)
	matches = tv.FilterAndSort(matches, q)
	if len(matches) == 0 {
		for _, e := range errs {
			fmt.Fprintf(deps.Stdout, "warning: %s data unavailable: %v\n", e.League, e.Err)
		}
		_, err := io.WriteString(deps.Stdout, "暂无即将开始的比赛。\n")
		return err
	}
	matches = matches[:1]
	if f.jsonOut {
		return writeTVJSON(deps.Stdout, matches)
	}
	return renderTVTable(deps.Stdout, matches, loc, !f.noColor)
}

func buildTVQuery(td tvDeps, f tvFlags) (tv.Query, *time.Location, error) {
	now := time.Now
	if td.Now != nil {
		now = td.Now
	}
	tzName := f.tz
	if tzName == "" {
		tzName = "Asia/Shanghai"
	}
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return tv.Query{}, nil, fmt.Errorf("invalid timezone %q: %w", tzName, err)
	}
	var leagues []tv.League
	for _, raw := range f.leagues {
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			l, err := tv.ParseLeague(part)
			if err != nil {
				return tv.Query{}, nil, err
			}
			leagues = append(leagues, l)
		}
	}
	days := f.days
	if days <= 0 {
		days = 3
	}
	if days > 14 {
		days = 14
	}
	startOfDay := now().In(loc)
	startOfDay = time.Date(startOfDay.Year(), startOfDay.Month(), startOfDay.Day(), 0, 0, 0, 0, loc)
	from := startOfDay
	to := from.AddDate(0, 0, days+1).Add(-time.Nanosecond)
	if f.from != "" {
		t, err := time.ParseInLocation(time.DateOnly, f.from, loc)
		if err != nil {
			return tv.Query{}, nil, fmt.Errorf("invalid --from %q", f.from)
		}
		from = t
		to = from.AddDate(0, 0, days+1).Add(-time.Nanosecond)
	}
	if f.to != "" {
		t, err := time.ParseInLocation(time.DateOnly, f.to, loc)
		if err != nil {
			return tv.Query{}, nil, fmt.Errorf("invalid --to %q", f.to)
		}
		to = t.AddDate(0, 0, 1).Add(-time.Second)
	}
	q := tv.Query{
		Leagues:         leagues,
		From:            from,
		To:              to,
		Team:            f.team,
		Source:          f.source,
		IncludeFinished: f.includeFinished,
		Location:        loc,
	}
	return q, loc, nil
}

func collectMatches(ctx context.Context, td tvDeps, q tv.Query) ([]tv.Match, []tv.FetchResult) {
	all := td.Fetchers()
	var selected []tv.Fetcher
	if len(q.Leagues) == 0 {
		selected = all
	} else {
		want := make(map[tv.League]struct{}, len(q.Leagues))
		for _, l := range q.Leagues {
			want[l] = struct{}{}
		}
		for _, f := range all {
			if _, ok := want[f.League()]; ok {
				selected = append(selected, f)
			}
		}
	}
	results := tv.FetchAll(ctx, selected, q)
	var matches []tv.Match
	var errs []tv.FetchResult
	for _, r := range results {
		if r.Err != nil {
			errs = append(errs, r)
			continue
		}
		matches = append(matches, r.Matches...)
	}
	return matches, errs
}
