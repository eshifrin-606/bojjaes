package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/eshifrin/bojjaes/internal/games"
	"github.com/eshifrin/bojjaes/internal/lineup"
	"github.com/eshifrin/bojjaes/internal/score"
)

type stubStats struct{}

func (stubStats) AsOf(context.Context, int, int) (score.WeekStats, time.Time, error) {
	return score.WeekStats{}, time.Now(), nil
}

func (stubStats) FetchedAt(int, int) (time.Time, bool) {
	return time.Time{}, false
}

type stubGames struct{ games []games.Game }

func (s stubGames) AsOf(_ context.Context, season, week int) (games.Week, time.Time, error) {
	return games.NewWeek(season, week, s.games), time.Now(), nil
}

func testMux() *http.ServeMux {
	return newMux(stubStats{}, stubGames{}, lineup.New(lineup.Embedded))
}

func TestMuxServesTheMatchupPage(t *testing.T) {
	rec := httptest.NewRecorder()

	testMux().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/2025/15", nil))

	if rec.Code == http.StatusNotFound {
		t.Errorf("GET /2025/15 = %d, want the matchup handler to answer", rec.Code)
	}
}

func TestMuxRendersTheGamesSourcesGameState(t *testing.T) {
	detroit := stubGames{games: []games.Game{{Home: "DET", Away: "LAR", State: games.Complete}}}
	rec := httptest.NewRecorder()

	newMux(stubStats{}, detroit, lineup.New(lineup.Embedded)).
		ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/2025/15", nil))

	if body := rec.Body.String(); !strings.Contains(body, `<span class="game">Final</span>`) {
		t.Errorf("GET /2025/15 body has no Final game span; want the games source's DET game shown")
	}
}

// The route is method-qualified, so the 405 comes from the mux's routing table
// rather than from the handler.
func TestMuxRejectsTheWrongMethodOnTheMatchupRoute(t *testing.T) {
	rec := httptest.NewRecorder()

	testMux().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/2025/15", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /2025/15 = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

// /scores is not a JSON endpoint any more, but GET /{season} now matches any
// single path segment, so a POST there is a wrong-method request against that
// route rather than a path nothing is registered under.
func TestMuxNoLongerAnswersScores(t *testing.T) {
	rec := httptest.NewRecorder()

	testMux().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/scores", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /scores = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

// Against DefaultServeMux the second construction would panic on the duplicate
// pattern. This is the test that pins the mux as owned.
func TestMuxCanBeBuiltTwice(t *testing.T) {
	for _, mux := range []*http.ServeMux{testMux(), testMux()} {
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/2025/15", nil))

		if rec.Code == http.StatusNotFound {
			t.Errorf("GET /2025/15 = %d, want the matchup handler to answer", rec.Code)
		}
	}
}

func TestMuxServesTheVersionEndpoint(t *testing.T) {
	rec := httptest.NewRecorder()

	testMux().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/2025/15/version", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("GET /2025/15/version = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want the version handler's JSON", got)
	}
}

func TestMuxRejectsTheWrongMethodOnTheVersionRoute(t *testing.T) {
	rec := httptest.NewRecorder()

	testMux().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/2025/15/version", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /2025/15/version = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

// A third route beside these two must not capture either: each is told apart
// by something only its own handler does.
func TestMuxStillRoutesThePageAndTheSeason(t *testing.T) {
	page := httptest.NewRecorder()
	testMux().ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/2025/15", nil))
	if got := page.Header().Get("Content-Type"); page.Code != http.StatusOK || got != "text/html; charset=utf-8" {
		t.Errorf("GET /2025/15 = %d %q, want the matchup page's 200 HTML", page.Code, got)
	}

	season := httptest.NewRecorder()
	testMux().ServeHTTP(season, httptest.NewRequest(http.MethodGet, "/2025", nil))
	if got := season.Header().Get("Location"); season.Code != http.StatusFound || got != "/2025/16" {
		t.Errorf("GET /2025 = %d to %q, want the season redirect to /2025/16", season.Code, got)
	}
}
