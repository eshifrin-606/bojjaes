package sleeper

import (
	"context"
	"fmt"
	"strconv"

	"github.com/eshifrin/bojjaes/internal/nflweek"
	"github.com/eshifrin/bojjaes/internal/sleeper/internal/httpjson"
)

type stateRow struct {
	Season     string `json:"season"` // sent as text, e.g. "2026"
	SeasonType string `json:"season_type"`
	Week       int    `json:"week"`
}

// fetchState reads the current NFL state from the v1 host; it takes no season
// or week because it answers which ones are current.
func fetchState(ctx context.Context, v1BaseURL string) (stateRow, error) {
	var row stateRow
	err := httpjson.Get(ctx, v1BaseURL+"/state/nfl", "sleeper nfl state", &row)
	return row, err
}

func (r stateRow) current() (nflweek.Current, error) {
	season, err := strconv.Atoi(r.Season)
	if err != nil {
		return nflweek.Current{}, fmt.Errorf("reading sleeper nfl state season: %w", err)
	}
	return nflweek.New(season, r.Week, statePhase(r.SeasonType)), nil
}

func statePhase(seasonType string) nflweek.Phase {
	switch seasonType {
	case "pre":
		return nflweek.Preseason
	case "regular":
		return nflweek.Regular
	case "post":
		return nflweek.Postseason
	case "off":
		return nflweek.Offseason
	}
	return nflweek.Other
}
