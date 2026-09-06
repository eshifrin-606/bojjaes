// Command server scores NFL players against the rushing/receiving slice of the
// HMFFL rules, from Sleeper's weekly stats.
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
	"github.com/eshifrin/bojjaes/internal/sleeper"
)

const addr = ":8080"

func main() {
	weeks := sleeper.Client{BaseURL: sleeper.BaseURL}

	// Method-qualified pattern, so anything but POST on this path gets a 405
	// from the mux rather than reaching a handler.
	http.Handle("POST /scores", api.BatchHandler(weeks))

	log.Printf("listening on %s; POST http://localhost%s/scores", addr, addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
