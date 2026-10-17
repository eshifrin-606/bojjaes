package plays

import (
	"context"
	"fmt"

	"github.com/eshifrin/bojjaes/internal/sleeper/internal/httpjson"
)

type play struct {
	ID        string    `json:"play_id"`
	UpdatedAt int64     `json:"updated_at"`
	Metadata  playMeta  `json:"metadata"`
	PlayStats []playRow `json:"play_stats"`
}

type playMeta struct {
	Description string `json:"description"`
	Team        string `json:"team"`
	Opponent    string `json:"opponent"`
	QuarterName string `json:"quarter_name"`
	Minutes     int    `json:"time_remaining_minutes"`
	Seconds     int    `json:"time_remaining_seconds"`
}

type playRow struct {
	PlayerID string             `json:"player_id"`
	Stats    map[string]float64 `json:"stats"`
}

// fetchRecentPlays reads the limit newest plays of a week, newest first.
func fetchRecentPlays(ctx context.Context, baseURL string, season, week, limit int) ([]play, error) {
	url := fmt.Sprintf("%s/plays/nfl/recent?season_type=regular&season=%d&week=%d&limit=%d", baseURL, season, week, limit)

	var plays []play
	if err := httpjson.Get(ctx, url, fmt.Sprintf("sleeper plays for season %d week %d", season, week), &plays); err != nil {
		return nil, err
	}
	return plays, nil
}
