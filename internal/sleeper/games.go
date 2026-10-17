package sleeper

import (
	"context"
	"fmt"
	"time"

	"github.com/eshifrin/bojjaes/internal/games"
	"github.com/eshifrin/bojjaes/internal/sleeper/internal/httpjson"
)

type gameRow struct {
	Status    string `json:"status"`
	StartTime int64  `json:"start_time"` // epoch milliseconds
	Metadata  struct {
		HomeTeam string `json:"home_team"`
		AwayTeam string `json:"away_team"`
		Quarter  string `json:"quarter"`
	} `json:"metadata"`
}

// WeekGames reads a week's games from the scores endpoint, which, unlike the
// schedule one, carries kickoff time and quarter. It translates Sleeper's
// fields only; display wording is the page's choice.
//
// An unplayed or out-of-season week answers 200 with `[]`, which is an empty
// week, not an error.
func (c Client) WeekGames(ctx context.Context, season, week int) (games.Week, error) {
	start := time.Now()
	url := fmt.Sprintf("%s/scores/nfl/regular/%d/%d", c.BaseURL, season, week)

	var rows []gameRow
	if err := httpjson.Get(ctx, url, fmt.Sprintf("sleeper games for season %d week %d", season, week), &rows); err != nil {
		return games.Week{}, err
	}
	gs := make([]games.Game, 0, len(rows))
	for _, row := range rows {
		// A game without both teams cannot be found by ForTeam, so it is
		// dropped rather than failing the week.
		if row.Metadata.HomeTeam == "" || row.Metadata.AwayTeam == "" {
			continue
		}
		gs = append(gs, row.game())
	}
	c.logf("sleeper games %d w%d: %d games in %s", season, week, len(gs), time.Since(start))
	return games.NewWeek(season, week, gs), nil
}

func (r gameRow) game() games.Game {
	return games.Game{
		Home:    r.Metadata.HomeTeam,
		Away:    r.Metadata.AwayTeam,
		State:   gameState(r.Status),
		Status:  r.Status,
		Kickoff: time.UnixMilli(r.StartTime),
		Period:  r.Metadata.Quarter,
	}
}

func gameState(status string) games.State {
	switch status {
	case "pre_game":
		return games.Upcoming
	case "in_game":
		return games.InProgress
	case "complete":
		return games.Complete
	}
	return games.Other
}
