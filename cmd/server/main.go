// Command server scores NFL players against the rushing/receiving slice of the
// HMFFL rules, from Sleeper's weekly stats.
//
// GET /{season}/{week} serves that week's matchup as an HTML page: the two
// teams of that week's lineup directory, their starters, and each lineup's
// total.
//
// POST /scores takes a season, a week, and up to a lineup's worth of player IDs,
// and answers with each player's stat line and point total.
//
// This is the composition root: it is the only package that names a provider.
package main

import (
	"log"
	"net/http"

	"github.com/eshifrin/bojjaes/internal/api"
	"github.com/eshifrin/bojjaes/internal/lineup"
	"github.com/eshifrin/bojjaes/internal/sleeper"
	"github.com/eshifrin/bojjaes/internal/statscache"
	"github.com/eshifrin/bojjaes/internal/web"
)

const addr = ":8080"

func main() {
	// Wrapped once, here, and handed to both handlers: the cache bounds
	// upstream volume only if everything that reads a week reads through the
	// same one. A cache per handler would be two budgets for one league.
	stats := statscache.New(sleeper.Client{BaseURL: sleeper.BaseURL}, statscache.TTL)

	// Method-qualified pattern, so anything but POST on this path gets a 405
	// from the mux rather than reaching a handler.
	http.Handle("POST /scores", api.BatchHandler(stats))
	http.Handle("GET /{season}/{week}", web.Handler(lineup.New(lineup.Embedded), stats))

	log.Printf("listening on %s; GET http://localhost%s/2025/15 and POST http://localhost%s/scores", addr, addr, addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
