package web

import (
	"maps"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/eshifrin/bojjaes/internal/lineup"
)

// serveSeason routes through a mux carrying both patterns, the way main
// registers them, so a request that names a week still reaches the existing
// handler.
func serveSeason(seasonHandler, weekHandler http.Handler, method, path string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	mux.Handle("GET /{season}/{week}", weekHandler)
	mux.Handle("GET /{season}", seasonHandler)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func TestSeasonRedirectsToItsHighestWeek(t *testing.T) {
	fsys := weekFS(2026, 1, map[string]string{"bojjaes": lineupHeader, "wood": lineupHeader})
	maps.Copy(fsys, weekFS(2026, 2, map[string]string{"bojjaes": lineupHeader, "renegades": lineupHeader}))
	tree := lineup.New(fsys)

	rec := serveSeason(SeasonHandler(tree), Handler(tree, unusedSource{t}), http.MethodGet, "/2026")

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	if got := rec.Header().Get("Location"); got != "/2026/2" {
		t.Errorf("Location = %q, want %q", got, "/2026/2")
	}
}

func TestSeasonBadRequestPaths(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "non-numeric season", path: "/twentytwentysix"},
		{name: "season before the provider's records", path: "/1899"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := lineup.New(fstest.MapFS{})
			rec := serveSeason(SeasonHandler(tree), Handler(tree, unusedSource{t}), http.MethodGet, tt.path)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("GET %s = %d, want %d", tt.path, rec.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestSeasonWithNoWeeksIsNotFound(t *testing.T) {
	tree := lineup.New(fstest.MapFS{})

	rec := serveSeason(SeasonHandler(tree), Handler(tree, unusedSource{t}), http.MethodGet, "/2027")

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestSeasonRedirectSkipsAGap(t *testing.T) {
	fsys := weekFS(2026, 1, map[string]string{"bojjaes": lineupHeader, "wood": lineupHeader})
	maps.Copy(fsys, weekFS(2026, 3, map[string]string{"bojjaes": lineupHeader, "renegades": lineupHeader}))
	tree := lineup.New(fsys)

	rec := serveSeason(SeasonHandler(tree), Handler(tree, unusedSource{t}), http.MethodGet, "/2026")

	if got := rec.Header().Get("Location"); got != "/2026/3" {
		t.Errorf("Location = %q, want %q", got, "/2026/3")
	}
}

func TestSeasonRedirectsEvenWhenTheLatestWeekIsNotAMatchup(t *testing.T) {
	tree := lineup.New(weekFS(2026, 3, map[string]string{
		"aroma": lineupHeader, "bojjaes": lineupHeader, "wood": lineupHeader,
	}))

	rec := serveSeason(SeasonHandler(tree), Handler(tree, unusedSource{t}), http.MethodGet, "/2026")

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	if got := rec.Header().Get("Location"); got != "/2026/3" {
		t.Errorf("Location = %q, want %q", got, "/2026/3")
	}
}

func TestSeasonRedirectNeverCallsTheStatsProvider(t *testing.T) {
	tree := lineup.New(weekFS(2026, 1, map[string]string{"bojjaes": lineupHeader, "wood": lineupHeader}))

	serveSeason(SeasonHandler(tree), Handler(tree, unusedSource{t}), http.MethodGet, "/2026")
}

// The mux prefers the more specific pattern, so a season/week request must
// still reach the existing handler unchanged.
func TestSeasonWeekPathStillReachesTheExistingHandler(t *testing.T) {
	weekTree := weekFS(2026, 2, map[string]string{
		"bojjaes": lineupHeader + "1,Puka Nacua,WR,LAR\n",
		"wood":    lineupHeader + "2,Josh Allen,QB,BUF\n",
	})
	tree := lineup.New(weekTree)

	rec := serveSeason(SeasonHandler(tree), Handler(tree, &fakeSource{}), http.MethodGet, "/2026/2")

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
