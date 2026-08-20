// Command server scores NFL players against the rushing/receiving slice of the
// HMFFL rules, from Sleeper's weekly stats.
//
// POST /scores takes a season, a week, and up to a roster's worth of player
// IDs. GET /score is a fixed smoke test over the same path: Puka Nacua, 2025
// week 14, a settled result that should never change.
package main

import (
	"log"
	"net/http"

	"github.com/eshifrin/bojjaes/internal/score"
)

const addr = ":8080"

func main() {
	// Method-qualified patterns, so anything but the listed method on these
	// paths gets a 405 from the mux rather than reaching a handler.
	http.Handle("GET /score", score.Handler(score.SleeperBaseURL))
	http.Handle("POST /scores", score.BatchHandler(score.SleeperBaseURL))

	log.Printf("listening on %s; GET http://localhost%s/score, POST http://localhost%s/scores", addr, addr, addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
