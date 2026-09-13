package main

import (
	"net/http"

	"github.com/eshifrin/bojjaes/internal/lineup"
	"github.com/eshifrin/bojjaes/internal/web"
)

// newMux builds the routing table as a value. On DefaultServeMux it would be
// process-wide state that cannot be built twice.
func newMux(stats web.StatsSource, tree *lineup.Tree) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /{season}/{week}", web.Handler(tree, stats))
	return mux
}
