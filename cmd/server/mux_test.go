package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eshifrin/bojjaes/internal/lineup"
	"github.com/eshifrin/bojjaes/internal/score"
)

type stubStats struct{}

func (stubStats) WeekStatsAsOf(context.Context, int, int) (score.WeekStats, time.Time, error) {
	return score.WeekStats{}, time.Now(), nil
}

func testMux() *http.ServeMux {
	return newMux(stubStats{}, lineup.New(lineup.Embedded))
}

func TestMuxServesTheMatchupPage(t *testing.T) {
	rec := httptest.NewRecorder()

	testMux().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/2025/15", nil))

	if rec.Code == http.StatusNotFound {
		t.Errorf("GET /2025/15 = %d, want the matchup handler to answer", rec.Code)
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

// /scores is no longer a route: it is one path segment with nothing registered
// under it, so the mux answers 404 rather than reaching any handler.
func TestMuxNoLongerAnswersScores(t *testing.T) {
	rec := httptest.NewRecorder()

	testMux().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/scores", nil))

	if rec.Code != http.StatusNotFound {
		t.Errorf("POST /scores = %d, want %d", rec.Code, http.StatusNotFound)
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
