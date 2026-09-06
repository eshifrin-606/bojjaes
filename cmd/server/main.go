// Command server scores NFL players against the rushing/receiving slice of the
// HMFFL rules, from Sleeper's weekly stats.
//
// GET /{season}/{week} serves that week's matchup as an HTML page: the two
// teams of that week's lineup directory, their starters, and each lineup's
// total.
//
// POST /scores takes a season, a week, and up to a roster's worth of player IDs,
// and answers with each player's stat line and point total.
//
// This is the composition root: it is the only package that names a provider.
package main

import (
	"log"
	"net/http"

	"github.com/eshifrin/bojjaes/internal/api"
	"github.com/eshifrin/bojjaes/internal/roster"
	"github.com/eshifrin/bojjaes/internal/sleeper"
	"github.com/eshifrin/bojjaes/internal/web"
)

const addr = ":8080"

// lineupRoot is relative to the working directory, which is what the scripts
// already assume. Interim: it becomes an embedded filesystem when the lineup
// tree moves inside a package.
const lineupRoot = "scripts/lineups"

func main() {
	stats := sleeper.Client{BaseURL: sleeper.BaseURL}

	// Method-qualified pattern, so anything but POST on this path gets a 405
	// from the mux rather than reaching a handler.
	http.Handle("POST /scores", api.BatchHandler(stats))
	http.Handle("GET /{season}/{week}", web.Handler(roster.New(lineupRoot), stats))

	log.Printf("listening on %s; GET http://localhost%s/2025/15 and POST http://localhost%s/scores", addr, addr, addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
