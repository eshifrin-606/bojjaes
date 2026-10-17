package web

import (
	"encoding/json"
	"net/http"
	"time"
)

type versionResponse struct {
	Version          string `json:"version"`
	NextCheckSeconds int    `json:"nextCheckSeconds"`
}

// VersionSource reports the fetch instant of the stats held for one season and
// week, and false when none is held. Like StatsSource it is declared by the
// consumer; main supplies the cache for both.
type VersionSource interface {
	FetchedAt(season, week int) (time.Time, bool)
}

// VersionHandler reports the stats version the server holds for the season and
// week in the request path, empty when it holds none. It takes no lineup tree
// and no StatsSource: a visible tab asks every few seconds, so answering must
// be unable to read a lineup or start a fetch.
func VersionHandler(source VersionSource) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		season, week, ok := seasonWeek(w, r)
		if !ok {
			return
		}

		writeVersion(w, source, season, week)
	})
}

func writeVersion(w http.ResponseWriter, source VersionSource, season, week int) {
	resp := versionResponse{NextCheckSeconds: checkDelaySeconds}
	if fetchedAt, held := source.FetchedAt(season, week); held {
		resp.Version = versionToken(fetchedAt)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(resp)
}

// LatestWeekSource resolves a season to its latest week. It is the season
// form's only view of the lineup tree, so that endpoint cannot open a roster.
type LatestWeekSource interface {
	LatestWeek(season int) (week int, ok bool)
}

// SeasonVersionHandler reports the stats version held for the season's latest
// week, resolved on every request so an open /{season} tab moves to a new week.
func SeasonVersionHandler(weeks LatestWeekSource, source VersionSource) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		season, ok := pathSeason(w, r)
		if !ok {
			return
		}
		week, ok := weeks.LatestWeek(season)
		if !ok {
			http.Error(w, "no such season", http.StatusNotFound)
			return
		}
		writeVersion(w, source, season, week)
	})
}
