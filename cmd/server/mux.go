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
// process-wide state that cannot be built twice. A nil saver leaves POST
// /lineup unrouted, so it answers 405 and the form offers no Save.
func newMux(stats weekStats, games web.GamesSource, current web.CurrentWeekSource, tree *lineup.Tree, rosters *lineup.Rosters, saver web.LineupSaver) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /{season}/{week}", web.Handler(tree, stats, games))
	mux.Handle("GET /{season}/{week}/version", web.VersionHandler(stats))
	mux.Handle("GET /{season}/version", web.SeasonVersionHandler(tree, stats))
	mux.Handle("GET /{season}", web.SeasonHandler(tree, stats, games))
	mux.Handle("GET /lineup", web.LineupHandler(rosters, tree, current, games, saver))
	mux.Handle("GET /lineup.csv", web.LineupCSVHandler(rosters, tree, current))
	mux.Handle("GET /lineup.txt", web.LineupTextHandler(rosters, tree, current))
	if saver != nil {
		mux.Handle("POST /lineup", web.LineupSaveHandler(rosters, tree, current, games, saver))
	}
	return mux
}
