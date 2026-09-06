// Package sleeper adapts Sleeper's weekly stats REST API to the neutral types
// in internal/score. It is the only package in which a Sleeper stat key or
// payload shape appears; nothing here may be imported by a scoring or serving
// package.
package sleeper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/eshifrin/bojjaes/internal/score"
)

// BaseURL is the live REST host; tests pass an httptest.Server URL.
const BaseURL = "https://api.sleeper.app"

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
func statLineFrom(weekly map[string]map[string]float64, playerID string, season, week int) (score.StatLine, bool) {
	// A null entry decodes to a nil map, which reads every stat as zero just as
	// convincingly as a missing player does.
	raw, ok := weekly[playerID]
	if !ok || raw == nil {
		return score.StatLine{}, false
	}

	// A stat the player did not record is absent from their entry, which reads
	// as zero — the same thing it means. A present entry with no mapped stats
	// is therefore a real scoreless line, not an absence.
	stat := func(key string) int { return int(raw[key]) }

	// Sacks are credited in halves, and the payload is already float64, so
	// this reader converts nothing where stat truncates.
	statFloat := func(key string) float64 { return raw[key] }

	return score.StatLine{
		PlayerID: playerID,
		Season:   season,
		Week:     week,
		// pass_yd, not pass_rush_yd: the latter is passing plus rushing yards
		// combined, and it appears on running backs too.
		PassYd: stat("pass_yd"),
		RushYd: stat("rush_yd"),
		RecYd:  stat("rec_yd"),
		// The touchdown keys are an allowlist. The rest belong to someone
		// else: pass_int_td to the quarterback who threw the pick-six,
		// kr_td/pr_td/def_td to team rows, td/anytime_tds to mixed
		// aggregates. anytime_tds reads like a shortcut for the whole rule
		// and is not one — it omits defensive touchdowns.
		PassTD:   stat("pass_td"),
		RushTD:   stat("rush_td"),
		RecTD:    stat("rec_td"),
		TD40Plus: stat("pass_td_40p") + stat("rush_td_40p") + stat("rec_td_40p"),
		TwoPt:    stat("pass_2pt") + stat("rush_2pt") + stat("rec_2pt"),
		// pass_int is interceptions thrown, and pays the passer -3. The
		// defender's side of the same play is idp_int, below.
		PassInt: stat("pass_int"),
		FumLost: stat("fum_lost"),

		// idp_int, not the generic int: idp_int is the key observed on a real
		// interception, carried by the week 14 pick-six defender alongside
		// idp_def_td. int is named by Sleeper's documentation but unverified
		// here.
		IntCaught: stat("idp_int"),
		DefTD:     stat("idp_def_td"),
		ReturnTD:  stat("st_td"),

		// Only recoveries that were turnovers pay, and no single key holds
		// them: idp_fum_rec alone misses the special-teams ones. fum_rec is
		// not part of the sum — it holds own-team recoveries, which are not
		// turnovers, and adding it is what would make this term raw.
		FumRec: stat("idp_fum_rec") + stat("st_fum_rec") + stat("def_st_fum_rec"),

		// idp_sack is the sack recorded; pass_sack is the same play from the
		// sacked quarterback's side.
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

// FetchWeek reads one season and week's stats and returns them as a
// provider-neutral snapshot.
//
// Every entry in the payload is transformed, not only the ones a caller will
// ask for: that is what lets the decoded map's lifetime end in this function,
// so no Sleeper shape escapes the package.
func FetchWeek(ctx context.Context, baseURL string, season, week int) (score.Week, error) {
	weekly, err := fetchWeekly(ctx, baseURL, season, week)
	if err != nil {
		return score.Week{}, err
	}

	players := make(map[string]score.StatLine, len(weekly))
	for playerID := range weekly {
		line, ok := statLineFrom(weekly, playerID, season, week)
		if !ok {
			// A null entry is not a scoreless week; leaving it out of the map
			// keeps absence absent.
			continue
		}
		players[playerID] = line
	}
	return score.NewWeek(season, week, players), nil
}

// Client is a handle on one Sleeper host. Its Week method satisfies the
// week-source interfaces the serving packages declare for themselves.
//
// It lives here rather than in main because it is provider state — a base URL —
// and holding it here means each composition root wires a value instead of
// redeclaring the same adapter. Nothing in this package names the interfaces it
// happens to satisfy, so the dependency still points one way.
type Client struct {
	BaseURL string
}

func (c Client) Week(ctx context.Context, season, week int) (score.Week, error) {
	return FetchWeek(ctx, c.BaseURL, season, week)
}
