// Package web serves the HTML matchup page. It joins the lineup tree in
// internal/lineup to the scoring in internal/score, and owns the template that
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
	"time"
	// The binary carries its own zone database, so America/Chicago loads the
	// same whether or not the deploy image ships /usr/share/zoneinfo.
	_ "time/tzdata"

	"github.com/eshifrin/bojjaes/internal/lineup"
	"github.com/eshifrin/bojjaes/internal/score"
)

// noStats stands in for a point value the provider has nothing to say about.
const noStats = "--"

//go:embed matchup.html
var templateFS embed.FS

// Parsed once, at package initialisation, so a broken template stops the
// process at startup rather than the first request.
var page = template.Must(template.ParseFS(templateFS, "matchup.html"))

// chicagoLoc is the league's timezone; the as-of line's wall-clock rendering is
// in it. Loaded once here, like the template parse, so a missing zone database
// is a boot failure with a clear message rather than a per-request 500.
var chicagoLoc = mustLoadLocation("America/Chicago")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic("web: loading timezone " + name + ": " + err.Error())
	}
	return loc
}

// matchup is what the template renders: two columns and nothing that compares
// them, plus the instant the stats were fetched — stated once, outside both
// columns.
type matchup struct {
	Season, Week int
	Columns      [2]column

	// FetchedAtRFC3339 is the fetch instant as a machine-readable RFC 3339
	// string; it is the original instant, not shifted into Chicago, so the
	// offset it carries is unambiguous. FetchedAtText is the same instant as a
	// Chicago wall-clock time for a reader.
	FetchedAtRFC3339 string
	FetchedAtText    string
}

// fetchedAtLayout renders the fetch instant with weekday, date, 12-hour time,
// and zone abbreviation, so a reader can tell which clock it is and check it
// against their own.
const fetchedAtLayout = "Mon, Jan 2 2006 3:04 PM MST"

type column struct {
	Team     string
	Starters []starter
	Total    string
}

type starter struct {
	Name      string
	ShortName string
	// Points is formatted here rather than in the template, so the choice
	// between a number and a placeholder is made in Go, where it is testable.
	Points string
}

// StatsSource supplies one season and week's stats along with the instant they
// were fetched from the provider. It is declared here, by the consumer, so no
// provider package is named on this side of the boundary; main supplies the
// implementation. The page states when its stats were fetched, so an un-cached
// source — which has no honest fetch instant — would not satisfy this by
// design.
type StatsSource interface {
	WeekStatsAsOf(ctx context.Context, season, week int) (score.WeekStats, time.Time, error)
}

// Handler renders the matchup page for the season and week in the request path.
func Handler(tree *lineup.Tree, source StatsSource) http.Handler {
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
			if errors.Is(err, lineup.ErrNoWeek) {
				http.Error(w, "no such week", http.StatusNotFound)
				return
			}
			http.Error(w, "that week is not a matchup", http.StatusInternalServerError)
			return
		}

		// Both lineups are read before the provider is called: a week our
		// lineup tree is wrong about is our mistake to fix, and there is no
		// reason to fetch a week we cannot render.
		teams := [2]string{ours, theirs}
		var lineups [2]lineup.Lineup
		for i, team := range teams {
			l, err := tree.Read(season, week, team)
			if err != nil {
				log.Printf("reading %s lineup for %d week %d: %v", team, season, week, err)
				http.Error(w, "that week's lineups could not be read", http.StatusInternalServerError)
				return
			}
			lineups[i] = l
		}

		// Fetched once, and both columns scored from it: the two lineups must
		// not be read from different snapshots of the week.
		weekStats, fetchedAt, err := source.WeekStatsAsOf(r.Context(), season, week)
		if err != nil {
			log.Printf("fetching stats for %d week %d: %v", season, week, err)

			// Never a zeroed page: it would read as "these players scored
			// nothing" rather than "we do not know".
			http.Error(w, "the week's stats could not be fetched", http.StatusBadGateway)
			return
		}

		view := matchup{
			Season:           season,
			Week:             week,
			FetchedAtRFC3339: fetchedAt.Format(time.RFC3339),
			FetchedAtText:    fetchedAt.In(chicagoLoc).Format(fetchedAtLayout),
		}
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
func scoreColumn(team string, l lineup.Lineup, weekStats score.WeekStats) column {
	col := column{Team: team}

	var total float64
	for _, rec := range l.Starters() {
		s := starter{Name: rec.Name, ShortName: rec.ShortName, Points: noStats}

		// Absence and a scoreless week are different facts, and Player's
		// second return is the only thing that tells them apart.
		if stats, played := weekStats.Player(rec.ID); played {
			pts := score.Points(stats)
			total += pts
			s.Points = formatPoints(pts)
		}
		col.Starters = append(col.Starters, s)
	}
	col.Total = formatPoints(total)

	return col
}
