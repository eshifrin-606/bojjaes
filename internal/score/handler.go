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

// FetchFunc produces the stat line to score. It takes only a context because
// player, season, and week are still constants.
type FetchFunc func(ctx context.Context) (StatLine, error)

// Handler scores whatever fetch returns and serves it as JSON.
func Handler(fetch FetchFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stats, err := fetch(r.Context())
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
