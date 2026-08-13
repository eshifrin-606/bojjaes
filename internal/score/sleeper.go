package score

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// The one player and week this walking skeleton scores. Verified against
// Sleeper's player index (9493 = Puka Nacua, LAR WR, BYU) and the week 14 box
// score.
const (
	NacuaPlayerID = "9493"
	TargetSeason  = 2025
	TargetWeek    = 14
)

// SleeperBaseURL is the live REST host; tests pass an httptest.Server URL.
const SleeperBaseURL = "https://api.sleeper.app"

// http.DefaultClient has no timeout, so a stalled upstream would hang the
// request forever. The budget covers the whole exchange, body included.
var sleeperClient = &http.Client{Timeout: 15 * time.Second}

// FetchStatLine maps one player's entry out of Sleeper's weekly stats
// aggregate.
//
// The endpoint returns every player in the league — roughly half a megabyte to
// read six numbers. Wasteful, and entirely fine at one request per manual hit.
func FetchStatLine(ctx context.Context, baseURL string, season, week int, playerID string) (StatLine, error) {
	url := fmt.Sprintf("%s/v1/stats/nfl/regular/%d/%d", baseURL, season, week)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return StatLine{}, fmt.Errorf("building sleeper request for season %d week %d: %w", season, week, err)
	}

	resp, err := sleeperClient.Do(req)
	if err != nil {
		return StatLine{}, fmt.Errorf("fetching sleeper stats for season %d week %d: %w", season, week, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return StatLine{}, fmt.Errorf("fetching sleeper stats for season %d week %d: status %s", season, week, resp.Status)
	}

	// Sleeper sends every stat as a JSON number, so decode as float64 across
	// the board and convert at the boundary.
	var weekly map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&weekly); err != nil {
		return StatLine{}, fmt.Errorf("decoding sleeper stats for season %d week %d: %w", season, week, err)
	}

	// A null entry decodes to a nil map, which reads every stat as zero just as
	// convincingly as a missing player does.
	raw, ok := weekly[playerID]
	if !ok || raw == nil {
		return StatLine{}, fmt.Errorf("player %s not found in sleeper stats for season %d week %d", playerID, season, week)
	}

	// A stat the player did not record is absent from their entry, which reads
	// as zero — the same thing it means. Only a missing *player* is an error,
	// since a silent zero score is the failure worth catching.
	stat := func(key string) int { return int(raw[key]) }

	return StatLine{
		PlayerID: playerID,
		Season:   season,
		Week:     week,
		RushYd:   stat("rush_yd"),
		RecYd:    stat("rec_yd"),
		RushTD:   stat("rush_td"),
		RecTD:    stat("rec_td"),
		TD40Plus: stat("rush_td_40p") + stat("rec_td_40p"),
		FumLost:  stat("fum_lost"),
	}, nil
}
