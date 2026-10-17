package main

import (
	"net/http"

	"github.com/eshifrin/bojjaes/internal/lineup"
	"github.com/eshifrin/bojjaes/internal/web"
)

// weekStats is one parameter rather than two so the page and the version
// endpoint cannot be wired to different caches: a version from any other
// source would never match the page's, and every check would reload it.
type weekStats interface {
	web.StatsSource
	web.VersionSource
}

// newMux builds the routing table as a value. On DefaultServeMux it would be
// process-wide state that cannot be built twice.
func newMux(stats weekStats, games web.GamesSource, tree *lineup.Tree) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /{season}/{week}", web.Handler(tree, stats, games))
	mux.Handle("GET /{season}/{week}/version", web.VersionHandler(stats))
	mux.Handle("GET /{season}", web.SeasonHandler(tree))
	return mux
}
