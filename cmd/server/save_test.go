package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/eshifrin/bojjaes/internal/data"
	"github.com/eshifrin/bojjaes/internal/lineup"
	"github.com/eshifrin/bojjaes/internal/nflweek"
	"github.com/eshifrin/bojjaes/internal/volume"
)

// startOnVolume wires the server as main does with a volume mounted.
func startOnVolume(t *testing.T, mount string, current stubCurrent) *http.ServeMux {
	t.Helper()
	lineups, err := volume.Lineups(data.Lineups, mount, t.Logf)
	if err != nil {
		t.Fatalf("volume.Lineups: %v", err)
	}
	return newMux(stubStats{}, stubGames{}, current, lineup.New(lineups), lineup.NewRosters(data.Rosters), volume.NewStore(mount))
}

func TestASavedLineupIsServedAndOutlivesARestart(t *testing.T) {
	mount := t.TempDir()
	week12 := stubCurrent{nflweek.New(2026, 12, nflweek.Regular)}
	week11, err := lineup.New(data.Lineups).Read(2026, 11, lineup.OurTeam)
	if err != nil {
		t.Fatalf("reading git's week 11: %v", err)
	}

	form := url.Values{"team": {lineup.OurTeam}, "season": {"2026"}, "week": {"12"}}
	for i, p := range week11.Starters() {
		form.Add(strings.ToLower(string(lineup.Slots()[i])), p.ID)
	}
	req := httptest.NewRequest(http.MethodPost, "/lineup", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	mux := startOnVolume(t, mount, week12)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("POST /lineup = %d, want 303:\n%s", rec.Code, rec.Body)
	}

	form12 := httptest.NewRecorder()
	mux.ServeHTTP(form12, httptest.NewRequest(http.MethodGet, rec.Header().Get("Location"), nil))
	if !strings.Contains(form12.Body.String(), "Saved for week 12") {
		t.Errorf("the form after the save does not say Saved for week 12:\n%s", form12.Body)
	}
	assertSavedWeek12(t, mux, week11)
	assertSavedWeek12(t, startOnVolume(t, mount, week12), week11)
}

func assertSavedWeek12(t *testing.T, mux *http.ServeMux, saved lineup.Lineup) {
	t.Helper()
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/2026/12", nil))

	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(body, "<h2>TBD</h2>") {
		t.Errorf("GET /2026/12 = %d, want the saved week with its opponent TBD:\n%s", rec.Code, body)
	}
	for _, p := range saved.Starters() {
		if !strings.Contains(body, p.Name) {
			t.Errorf("GET /2026/12 does not show saved starter %s", p.Name)
		}
	}
}
