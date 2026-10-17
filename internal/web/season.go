package web

import (
	"log"
	"net/http"
	"strconv"

	"github.com/eshifrin/bojjaes/internal/lineup"
	"github.com/eshifrin/bojjaes/internal/score"
)

// SeasonHandler shows a bare season's latest existing week in place, so
// /{season} stays the current week's address as new weeks arrive. It renders
// rather than redirects, so the latest week's refusals match its numbered URL.
func SeasonHandler(tree *lineup.Tree, source StatsSource, gameSource GamesSource) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		season, ok := pathSeason(w, r)
		if !ok {
			return
		}

		week, ok := tree.LatestWeek(season)
		if !ok {
			log.Printf("resolving latest week for season %d: no week directories", season)
			http.Error(w, "no such season", http.StatusNotFound)
			return
		}

		renderWeek(w, r, tree, source, gameSource, season, week)
	})
}

func pathSeason(w http.ResponseWriter, r *http.Request) (season int, ok bool) {
	season, err := strconv.Atoi(r.PathValue("season"))
	if err != nil {
		http.Error(w, "season must be a number", http.StatusBadRequest)
		return 0, false
	}
	// week 1 is always in range, so this checks only the season bound
	// without a separate season-only validator.
	if err := score.ValidateSeasonWeek(season, 1); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return 0, false
	}
	return season, true
}
