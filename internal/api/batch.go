// Package api is the JSON transport over scoring: request and response
// shapes, validation, and the handlers. It depends on internal/score and on
// the StatsSource interface it declares below — never on a provider package.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/eshifrin/bojjaes/internal/score"
)

// StatsSource supplies one season and week's stats. It is declared here, by the
// consumer, so that no provider package is named on this side of the boundary;
// main supplies the implementation.
type StatsSource interface {
	WeekStats(ctx context.Context, season, week int) (score.WeekStats, error)
}

// Request bounds. The roster cap is the league's maximum roster size, which
// comfortably exceeds two full starting lineups; it is a sanity bound, not a
// cost control, since the upstream request is the same size either way.
const (
	maxPlayerIDs = 26

	minSeason = 2009 // Sleeper's stats do not reach further back.
	maxSeason = 2099
	minWeek   = 1
	maxWeek   = 18
)

// BatchHandler scores many players for one season and week from a single fetch
// of the weekly aggregate.
//
// Interim: this endpoint exists so scripts/scores.sh and scripts/fantasycast.sh
// can run, and retires with them when the matchup page replaces them. It is not
// a designed public API.
func BatchHandler(source StatsSource) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req BatchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
			return
		}
		if err := req.validate(); err != nil {
			http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
			return
		}

		weekStats, err := source.WeekStats(r.Context(), req.Season, req.Week)
		if err != nil {
			// The response body reaches whoever made the request; the log
			// reaches whoever is running the server.
			log.Printf("fetching stats: %v", err)

			// Never serve a partial result: it reads as "these players scored
			// nothing" rather than "we do not know".
			http.Error(w, fmt.Sprintf("scoring failed: %v", err), http.StatusBadGateway)
			return
		}

		resp := newBatchResponse(req.Season, req.Week)
		for _, playerID := range req.PlayerIDs {
			stats, ok := weekStats.Player(playerID)
			if !ok {
				// The payload cannot say why a player is missing — not yet
				// kicked off, inactive, and unknown ID look identical — so the
				// ID is reported back rather than diagnosed or zeroed.
				resp.NoStats = append(resp.NoStats, playerID)
				continue
			}
			resp.Scores = append(resp.Scores, ScoreResponse{Stats: stats, Points: score.Points(stats)})
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			// Headers are already out; nothing to do but say so in the log.
			log.Printf("encoding response: %v", err)
		}
	})
}

func (r BatchRequest) validate() error {
	if r.Season < minSeason || r.Season > maxSeason {
		return fmt.Errorf("season %d outside %d-%d", r.Season, minSeason, maxSeason)
	}
	if r.Week < minWeek || r.Week > maxWeek {
		return fmt.Errorf("week %d outside %d-%d", r.Week, minWeek, maxWeek)
	}
	if len(r.PlayerIDs) == 0 {
		return fmt.Errorf("player_ids is empty")
	}
	if len(r.PlayerIDs) > maxPlayerIDs {
		return fmt.Errorf("%d player_ids exceeds the %d-player roster limit", len(r.PlayerIDs), maxPlayerIDs)
	}
	return nil
}
