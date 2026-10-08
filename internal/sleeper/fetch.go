package sleeper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// BaseURL is the live REST host; tests pass an httptest.Server URL.
const BaseURL = "https://api.sleeper.com"

// http.DefaultClient has no timeout, so a stalled upstream would hang the
// request forever. The budget covers the whole exchange, body included.
var sleeperClient = &http.Client{Timeout: 15 * time.Second}

type weeklyRow struct {
	PlayerID string             `json:"player_id"`
	Stats    map[string]float64 `json:"stats"`
	// Team is the player's team in that week's game; player.team is their
	// current one, which is wrong for a player traded since.
	Team   string `json:"team"`
	Player struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	} `json:"player"`
}

// fetchWeekly reads Sleeper's regular-season weekly stats rows and indexes
// them by player ID.
//
// One call serves any number of players: the endpoint returns every player in
// the league regardless, roughly two megabytes (~280 KB gzipped) whether one
// line is wanted or a whole roster.
//
// An empty payload is not an error — an unplayed week returns 200 with `[]`.
func fetchWeekly(ctx context.Context, baseURL string, season, week int) (map[string]map[string]float64, identities, error) {
	url := fmt.Sprintf("%s/stats/nfl/%d/%d?season_type=regular", baseURL, season, week)

	// Sleeper sends every stat as a JSON number, so decode as float64 across
	// the board and convert at the boundary.
	var rows []weeklyRow
	if err := getJSON(ctx, url, fmt.Sprintf("sleeper stats for season %d week %d", season, week), &rows); err != nil {
		return nil, nil, err
	}
	weekly, ids := indexWeekly(rows)
	return weekly, ids, nil
}

func indexWeekly(rows []weeklyRow) (map[string]map[string]float64, identities) {
	weekly := make(map[string]map[string]float64, len(rows))
	ids := make(identities, len(rows))
	for _, row := range rows {
		weekly[row.PlayerID] = row.Stats
		ids[row.PlayerID] = identity{name: abbrevName(row.Player.FirstName, row.Player.LastName), team: row.Team}
	}
	return weekly, ids
}

// fetchRecentPlays reads the limit newest plays of a week, newest first.
func fetchRecentPlays(ctx context.Context, baseURL string, season, week, limit int) ([]play, error) {
	url := fmt.Sprintf("%s/plays/nfl/recent?season_type=regular&season=%d&week=%d&limit=%d", baseURL, season, week, limit)

	var plays []play
	if err := getJSON(ctx, url, fmt.Sprintf("sleeper plays for season %d week %d", season, week), &plays); err != nil {
		return nil, err
	}
	return plays, nil
}

// getJSON decodes the 200 response body of a GET into out. what names the
// lookup in every error, so a failed call says which season and week it was.
func getJSON(ctx context.Context, url, what string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("building request for %s: %w", what, err)
	}

	resp, err := sleeperClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching %s: %w", what, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetching %s: status %s", what, resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding %s: %w", what, err)
	}
	return nil
}
