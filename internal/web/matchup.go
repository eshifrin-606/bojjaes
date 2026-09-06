// Package web serves the HTML matchup page. It joins the lineup tree in
// internal/roster to the scoring in internal/score, and owns the template that
// renders the result; neither of those packages learns about HTML.
package web

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/eshifrin/bojjaes/internal/roster"
	"github.com/eshifrin/bojjaes/internal/score"
)

// noStats stands in for a point value the provider has nothing to say about.
const noStats = "no stats"

//go:embed matchup.html
var templateFS embed.FS

// Parsed once, at package initialisation, so a broken template stops the
// process at startup rather than the first request.
var page = template.Must(template.ParseFS(templateFS, "matchup.html"))

// matchup is what the template renders: two columns and nothing that compares
// them.
type matchup struct {
	Season, Week int
	Columns      [2]column
}

type column struct {
	Team     string
	Starters []starter
	Total    string
}

type starter struct {
	Name string
	// Points is formatted here rather than in the template, so the choice
	// between a number and a placeholder is made in Go, where it is testable.
	Points string
}

// StatsSource supplies one season and week's stats. It is declared here, by the
// consumer, so no provider package is named on this side of the boundary; main
// supplies the implementation.
type StatsSource interface {
	WeekStats(ctx context.Context, season, week int) (score.WeekStats, error)
}

// Handler renders the matchup page for the season and week in the request path.
func Handler(tree *roster.Tree, source StatsSource) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Both segments are validated before the lineup tree is opened and
		// before the provider is called: a typo in a URL must not reach
		// Sleeper, and a stranger guessing paths must not make us fetch.
		season, err := strconv.Atoi(r.PathValue("season"))
		if err != nil {
			http.Error(w, "season and week must be numbers", http.StatusBadRequest)
			return
		}
		week, err := strconv.Atoi(r.PathValue("week"))
		if err != nil {
			http.Error(w, "season and week must be numbers", http.StatusBadRequest)
			return
		}
		if err := score.ValidateSeasonWeek(season, week); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ours, theirs, err := tree.Matchup(season, week)
		if err != nil {
			// The log carries the failing directory; the body never does,
			// since it reaches whoever asked.
			log.Printf("resolving %d week %d: %v", season, week, err)

			// A week we never played is the reader asking for something that
			// does not exist. A week directory that exists but is not a
			// matchup is our lineup tree being wrong about a well-formed
			// request, which is ours to fix, not theirs.
			if errors.Is(err, roster.ErrNoWeek) {
				http.Error(w, "no such week", http.StatusNotFound)
				return
			}
			http.Error(w, "that week is not a matchup", http.StatusInternalServerError)
			return
		}

		// Both rosters are read before the provider is called: a week our
		// lineup tree is wrong about is our mistake to fix, and there is no
		// reason to fetch a week we cannot render.
		teams := [2]string{ours, theirs}
		var lineups [2]roster.Roster
		for i, team := range teams {
			lineup, err := tree.Read(season, week, team)
			if err != nil {
				log.Printf("reading %s roster for %d week %d: %v", team, season, week, err)
				http.Error(w, "that week's rosters could not be read", http.StatusInternalServerError)
				return
			}
			lineups[i] = lineup
		}

		// Fetched once, and both columns scored from it: the two lineups must
		// not be read from different snapshots of the week.
		weekStats, err := source.WeekStats(r.Context(), season, week)
		if err != nil {
			log.Printf("fetching stats for %d week %d: %v", season, week, err)

			// Never a zeroed page: it would read as "these players scored
			// nothing" rather than "we do not know".
			http.Error(w, "the week's stats could not be fetched", http.StatusBadGateway)
			return
		}

		view := matchup{Season: season, Week: week}
		for i, team := range teams {
			view.Columns[i] = scoreColumn(team, lineups[i], weekStats)
		}

		// Rendered into a buffer first: executing straight into the
		// ResponseWriter would commit a 200 and half a page before it could
		// fail.
		var buf bytes.Buffer
		if err := page.Execute(&buf, view); err != nil {
			log.Printf("rendering %d week %d: %v", season, week, err)
			http.Error(w, "the page could not be rendered", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(buf.Bytes())
	})
}

// formatPoints renders a point total without a trailing zero: whole scores are
// the common case and read better as "12" than "12.0", while a half-sack still
// shows its half.
func formatPoints(pts float64) string {
	return strconv.FormatFloat(pts, 'f', -1, 64)
}

// scoreColumn scores one lineup's starters out of the week's stats.
func scoreColumn(team string, lineup roster.Roster, weekStats score.WeekStats) column {
	col := column{Team: team}

	var total float64
	for _, rec := range lineup.Starters() {
		stats, played := weekStats.Player(rec.ID)
		if !played {
			// Absence and a scoreless week are different facts, and Player's
			// second return is the only thing that tells them apart. The
			// wording matches scripts/scores.sh while both UIs exist.
			col.Starters = append(col.Starters, starter{Name: rec.Name, Points: noStats})
			continue
		}

		pts := score.Points(stats)
		total += pts
		col.Starters = append(col.Starters, starter{Name: rec.Name, Points: formatPoints(pts)})
	}
	col.Total = formatPoints(total)

	return col
}
