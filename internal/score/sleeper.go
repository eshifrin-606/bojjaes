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

// fetchWeekly reads Sleeper's regular-season weekly stats aggregate, keyed by
// player ID.
//
// One call serves any number of players: the endpoint returns every player in
// the league regardless, roughly half a megabyte whether one line is wanted or
// a whole roster.
//
// An empty payload is not an error — an unplayed week returns 200 with `{}`.
func fetchWeekly(ctx context.Context, baseURL string, season, week int) (map[string]map[string]float64, error) {
	url := fmt.Sprintf("%s/v1/stats/nfl/regular/%d/%d", baseURL, season, week)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("building sleeper request for season %d week %d: %w", season, week, err)
	}

	resp, err := sleeperClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching sleeper stats for season %d week %d: %w", season, week, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching sleeper stats for season %d week %d: status %s", season, week, resp.Status)
	}

	// Sleeper sends every stat as a JSON number, so decode as float64 across
	// the board and convert at the boundary.
	var weekly map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&weekly); err != nil {
		return nil, fmt.Errorf("decoding sleeper stats for season %d week %d: %w", season, week, err)
	}
	return weekly, nil
}

// statLineFrom maps one player out of a decoded weekly payload, reporting
// false when that player has no entry.
//
// Absence is a value rather than an error because the payload cannot say why a
// player is missing: not yet kicked off, inactive, and unknown ID all look
// identical. Only the caller can decide what absence means for its request.
func statLineFrom(weekly map[string]map[string]float64, playerID string, season, week int) (StatLine, bool) {
	// A null entry decodes to a nil map, which reads every stat as zero just as
	// convincingly as a missing player does.
	raw, ok := weekly[playerID]
	if !ok || raw == nil {
		return StatLine{}, false
	}

	// A stat the player did not record is absent from their entry, which reads
	// as zero — the same thing it means. A present entry with no mapped stats
	// is therefore a real scoreless line, not an absence.
	stat := func(key string) int { return int(raw[key]) }

	// Sacks are credited in halves, and the payload is already float64, so
	// this reader converts nothing where stat truncates.
	statFloat := func(key string) float64 { return raw[key] }

	return StatLine{
		PlayerID: playerID,
		Season:   season,
		Week:     week,
		// pass_yd, not pass_rush_yd: the latter is passing plus rushing yards
		// combined, and it appears on running backs too.
		PassYd:   stat("pass_yd"),
		RushYd:   stat("rush_yd"),
		RecYd:    stat("rec_yd"),
		PassTD:   stat("pass_td"),
		RushTD:   stat("rush_td"),
		RecTD:    stat("rec_td"),
		TD40Plus: stat("pass_td_40p") + stat("rush_td_40p") + stat("rec_td_40p"),
		TwoPt:    stat("pass_2pt") + stat("rush_2pt") + stat("rec_2pt"),
		// pass_int is interceptions thrown. Sleeper's int and idp_int are
		// interceptions caught, which belong to a defender.
		PassInt: stat("pass_int"),
		FumLost: stat("fum_lost"),

		Sack: statFloat("idp_sack"),

		FGMade: stat("fgm"),
		XPMade: stat("xpm"),
		// fgm_50p is used alone rather than summing fgm_50_59 and fgm_60p:
		// it equals their sum on every entry of the verified week, so the
		// sum would add a way for the three keys to disagree and nothing
		// else.
		FG50Plus: stat("fgm_50p"),
	}, true
}
