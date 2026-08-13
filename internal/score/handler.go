package score

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ScoreResponse is the served shape: the stats we read and the points they
// earned, together. Points are computed here rather than stored on StatLine,
// so the two can never disagree.
type ScoreResponse struct {
	Stats  StatLine `json:"stats"`
	Points float64  `json:"points"`
}

// FetchFunc produces the stat line to score. It takes no arguments because
// player, season, and week are hardcoded for this walking skeleton; widening
// it is the next change's job.
type FetchFunc func() (StatLine, error)

// Handler scores whatever fetch returns and serves it as JSON, echoing the
// stats alongside the total so a wrong player or a bad week is visible in the
// output rather than hidden behind a plausible-looking number.
func Handler(fetch FetchFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stats, err := fetch()
		if err != nil {
			// Never fall through to scoring a zero value here: 0.0 served
			// with a 200 is indistinguishable from a real scoreless week.
			http.Error(w, fmt.Sprintf("scoring failed: %v", err), http.StatusBadGateway)
			return
		}

		resp := ScoreResponse{Stats: stats, Points: Points(stats)}

		fmt.Printf("player %s season %d week %d: %.1f points (%+v)\n",
			resp.Stats.PlayerID, resp.Stats.Season, resp.Stats.Week, resp.Points, resp.Stats)

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			// Headers are already out; nothing to do but say so on stdout.
			fmt.Printf("encoding response: %v\n", err)
		}
	})
}
