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

func (stubStats) WeekStats(context.Context, int, int) (score.WeekStats, error) {
	return score.WeekStats{}, nil
}

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

func TestMuxRejectsTheWrongMethodOnScores(t *testing.T) {
	rec := httptest.NewRecorder()

	testMux().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/scores", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /scores = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

// The Allow header is the mux's own, not a handler's: it is how a 405 is shown
// to come from the routing table rather than from BatchHandler rejecting a body.
func TestMuxAnswersTheWrongMethodItself(t *testing.T) {
	rec := httptest.NewRecorder()

	testMux().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/scores", nil))

	if got := rec.Header().Get("Allow"); got != http.MethodPost {
		t.Errorf("Allow = %q, want %q", got, http.MethodPost)
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
