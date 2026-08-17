package score

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// ScoreResponse echoes the stats alongside the total, so a wrong player or a
// bad week shows in the output instead of hiding behind a plausible number.
type ScoreResponse struct {
	Stats  StatLine `json:"stats"`
	Points float64  `json:"points"`
}

// BatchRequest asks for one season and week. The season type is implicitly
// regular — the league scores regular-season play.
type BatchRequest struct {
	Season    int      `json:"season"`
	Week      int      `json:"week"`
	PlayerIDs []string `json:"player_ids"`
}

// BatchResponse splits scored players from players the payload had nothing for.
//
// The split is what keeps the two apart: NoStats is a list of IDs with nowhere
// to put a number, so an absent player cannot acquire a point total. Points
// carries no omitempty for the mirror-image reason — a real 0.0 must survive.
type BatchResponse struct {
	Season  int             `json:"season"`
	Week    int             `json:"week"`
	Scores  []ScoreResponse `json:"scores"`
	NoStats []string        `json:"no_stats"`
}

// newBatchResponse starts both buckets non-nil so they serialize as `[]`
// rather than `null`.
func newBatchResponse(season, week int) BatchResponse {
	return BatchResponse{
		Season:  season,
		Week:    week,
		Scores:  []ScoreResponse{},
		NoStats: []string{},
	}
}

// Handler scores the fixed target player and week, as a smoke test over the
// same path POST /scores takes.
//
// Absence is a failure here, though the transform now reports it as an
// ordinary result: the target is a settled historical week in which the player
// is known present, so a missing entry means the fetch or the transform broke.
// The conversion lives in this handler because only this endpoint can make it.
// fetchTarget scores the target player as a one-element batch, turning his
// absence into an error the endpoint reports as a 502.
func fetchTarget(ctx context.Context, baseURL string) (StatLine, error) {
	weekly, err := fetchWeekly(ctx, baseURL, TargetSeason, TargetWeek)
	if err != nil {
		return StatLine{}, err
	}

	stats, ok := statLineFrom(weekly, NacuaPlayerID, TargetSeason, TargetWeek)
	if !ok {
		return StatLine{}, fmt.Errorf("player %s not found in sleeper stats for season %d week %d",
			NacuaPlayerID, TargetSeason, TargetWeek)
	}
	return stats, nil
}

func Handler(baseURL string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stats, err := fetchTarget(r.Context(), baseURL)
		if err != nil {
			// The response body reaches whoever made the request; the log
			// reaches whoever is running the server.
			log.Printf("fetching stats: %v", err)

			// Never fall through to scoring the zero value: 0.0 served with a
			// 200 is indistinguishable from a real scoreless week.
			http.Error(w, fmt.Sprintf("scoring failed: %v", err), http.StatusBadGateway)
			return
		}

		resp := ScoreResponse{Stats: stats, Points: Points(stats)}

		fmt.Printf("player %s season %d week %d: %.1f points (%+v)\n",
			resp.Stats.PlayerID, resp.Stats.Season, resp.Stats.Week, resp.Points, resp.Stats)

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			// Headers are already out; nothing to do but say so in the log.
			log.Printf("encoding response: %v", err)
		}
	})
}
