// Package sleeper adapts Sleeper's weekly stats and scores REST endpoints to the
// neutral types in internal/score and internal/games. It and its subpackages
// are the only place a Sleeper stat key or payload shape appears; nothing here
// may be imported by a scoring or serving package.
package sleeper

import (
	"context"
	"time"

	"github.com/eshifrin/bojjaes/internal/games"
	"github.com/eshifrin/bojjaes/internal/nflweek"
	"github.com/eshifrin/bojjaes/internal/score"
	"github.com/eshifrin/bojjaes/internal/sleeper/plays"
)

// BaseURL is the live REST host; tests pass an httptest.Server URL.
const BaseURL = "https://api.sleeper.com"

// V1BaseURL is the documented v1 host, which serves the NFL state endpoint
// that BaseURL does not.
const V1BaseURL = "https://api.sleeper.app/v1"

// Client is a handle on one Sleeper host. Its WeekStats and WeekGames methods
// satisfy the source interfaces the serving packages declare for themselves.
//
// It lives here rather than in main because it is provider state — a base URL —
// and holding it here means each composition root wires a value instead of
// redeclaring the same adapter. Nothing in this package names the interfaces it
// happens to satisfy, so the dependency still points one way.
type Client struct {
	BaseURL   string
	V1BaseURL string
	Plays     *plays.Store
	Logf      func(format string, args ...any) // nil is silent
}

func (c Client) WeekStats(ctx context.Context, season, week int) (score.WeekStats, error) {
	start := time.Now()
	players, ids, err := fetchPlayerStats(ctx, c.BaseURL, season, week)
	if err != nil {
		return score.WeekStats{}, err
	}
	c.logf("sleeper stats %d w%d: %d rows in %s", season, week, len(ids), time.Since(start))
	c.addForcedFumbles(ctx, season, week, ids, players)
	return score.NewWeekStats(season, week, players), nil
}

func (c Client) WeekGames(ctx context.Context, season, week int) (games.Week, error) {
	start := time.Now()
	gs, err := fetchWeekGames(ctx, c.BaseURL, season, week)
	if err != nil {
		return games.Week{}, err
	}
	c.logf("sleeper games %d w%d: %d games in %s", season, week, len(gs), time.Since(start))
	return games.NewWeek(season, week, gs), nil
}

func (c Client) CurrentWeek(ctx context.Context) (nflweek.Current, error) {
	start := time.Now()
	row, err := fetchState(ctx, c.V1BaseURL)
	if err != nil {
		return nflweek.Current{}, err
	}
	current, err := row.current()
	if err != nil {
		return nflweek.Current{}, err
	}
	c.logf("sleeper state: %s %s w%d in %s", row.Season, row.SeasonType, row.Week, time.Since(start))
	return current, nil
}

func (c Client) addForcedFumbles(ctx context.Context, season, week int, ids plays.Identities, players map[string]score.StatLine) {
	if c.Plays == nil {
		return
	}
	forced := c.Plays.ForcedFumbles(ctx, season, week, ids)
	for id, n := range forced {
		line, ok := players[id]
		if !ok {
			line = score.StatLine{PlayerID: id, Season: season, Week: week}
		}
		line.FFTurnover = n
		players[id] = line
	}
}

func (c Client) logf(format string, args ...any) {
	if c.Logf != nil {
		c.Logf(format, args...)
	}
}
