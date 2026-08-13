package score

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// The one player and week this walking skeleton scores. Verified against
// Sleeper's player index (9493 = Puka Nacua, LAR WR, BYU) and the week 14 box
// score. Removing these constants in favour of request parameters is the first
// job of the next change.
const (
	NacuaPlayerID = "9493"
	TargetSeason  = 2025
	TargetWeek    = 14
)

// SleeperBaseURL is the live Sleeper REST host. Tests point FetchStatLine at an
// httptest.Server instead.
const SleeperBaseURL = "https://api.sleeper.app"

// FetchStatLine pulls Sleeper's weekly stats aggregate for a season and week
// and maps one player's entry onto a StatLine.
//
// The endpoint returns every player in the league — roughly half a megabyte to
// read six numbers. That is wasteful and entirely fine at one request per
// manual hit.
func FetchStatLine(baseURL string, season, week int, playerID string) (StatLine, error) {
	url := fmt.Sprintf("%s/v1/stats/nfl/regular/%d/%d", baseURL, season, week)

	resp, err := http.Get(url)
	if err != nil {
		return StatLine{}, fmt.Errorf("fetching sleeper stats for season %d week %d: %w", season, week, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return StatLine{}, fmt.Errorf("fetching sleeper stats for season %d week %d: status %s", season, week, resp.Status)
	}

	// Sleeper sends every stat as a JSON number, so float64 across the board
	// and convert at the boundary.
	var weekly map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&weekly); err != nil {
		return StatLine{}, fmt.Errorf("decoding sleeper stats for season %d week %d: %w", season, week, err)
	}

	raw, ok := weekly[playerID]
	if !ok {
		return StatLine{}, fmt.Errorf("player %s not found in sleeper stats for season %d week %d", playerID, season, week)
	}

	// A stat a player did not record is simply absent from their entry, which
	// reads as zero — the same thing it means. Only a missing *player* is an
	// error, since a silent zero score is the failure worth catching.
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
