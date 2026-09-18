package web

import (
	"log"
	"net/http"
	"strconv"

	"github.com/eshifrin/bojjaes/internal/lineup"
	"github.com/eshifrin/bojjaes/internal/score"
)

// SeasonHandler redirects a bare season to its latest existing week. The
// target is a fact about the tree's current contents, not a permanent
// identity for the season, so the redirect is temporary.
func SeasonHandler(tree *lineup.Tree) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		season, err := strconv.Atoi(r.PathValue("season"))
		if err != nil {
			http.Error(w, "season must be a number", http.StatusBadRequest)
			return
		}
		// week 1 is always in range, so this checks only the season bound
		// without a separate season-only validator.
		if err := score.ValidateSeasonWeek(season, 1); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		week, ok := tree.LatestWeek(season)
		if !ok {
			log.Printf("resolving latest week for season %d: no week directories", season)
			http.Error(w, "no such season", http.StatusNotFound)
			return
		}

		http.Redirect(w, r, "/"+strconv.Itoa(season)+"/"+strconv.Itoa(week), http.StatusFound)
	})
}
