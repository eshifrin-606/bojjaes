package main

import (
	"net/http"

	"github.com/eshifrin/bojjaes/internal/api"
	"github.com/eshifrin/bojjaes/internal/lineup"
	"github.com/eshifrin/bojjaes/internal/web"
)

// statsSource is what both handlers need of one cache: the API reads a week,
// the page also needs the instant it was fetched.
type statsSource interface {
	api.StatsSource
	web.StatsSource
}

// newMux builds the routing table as a value. On DefaultServeMux it would be
// process-wide state that cannot be built twice.
func newMux(stats statsSource, tree *lineup.Tree) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /{season}/{week}", web.Handler(tree, stats))
	// Method-qualified pattern, so anything but POST on this path gets a 405
	// from the mux rather than reaching a handler.
	mux.Handle("POST /scores", api.BatchHandler(stats))
	return mux
}
