package web

import (
	"net/http"
	"net/http/httptest"
	"slices"
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

func TestSeasonShowsItsHighestWeek(t *testing.T) {
	tree := lineup.New(treeFS(matchupWeek(2026, 1), matchupWeek(2026, 2)))
	source := &fakeSource{weekStats: fixtureStats()}

	rec := serveSeason(SeasonHandler(tree, source, noGames{}), Handler(tree, unusedSource{t}, noGames{}), http.MethodGet, "/2026")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Location"); got != "" {
		t.Errorf("Location = %q, want none", got)
	}
	var heading string
	if m := headingPattern.FindStringSubmatch(rec.Body.String()); m != nil {
		heading = m[1]
	}
	if heading != "2026 · Week 2" {
		t.Errorf("h1 = %q, want %q", heading, "2026 · Week 2")
	}
}

func TestSeasonShowsTheLatestWeeksBar(t *testing.T) {
	tree := lineup.New(treeFS(matchupWeek(2026, 1), matchupWeek(2026, 2)))
	h := Handler(tree, &fakeSource{weekStats: fixtureStats()}, noGames{})

	seasonBar := weekBarPattern.FindString(serveSeason(SeasonHandler(tree, &fakeSource{weekStats: fixtureStats()}, noGames{}), h, http.MethodGet, "/2026").Body.String())
	weekBar := weekBarPattern.FindString(serveSeason(SeasonHandler(tree, unusedSource{t}, noGames{}), h, http.MethodGet, "/2026/2").Body.String())

	if seasonBar == "" {
		t.Fatal("no week bar rendered on /2026")
	}
	if seasonBar != weekBar {
		t.Errorf("week bars differ:\n/2026:\n%s\n/2026/2:\n%s", seasonBar, weekBar)
	}
	if prev, next := renderedWeekLinks(seasonBar); prev != "/2026/1" || next != "" {
		t.Errorf("week links = prev %q, next %q, want prev %q, next %q", prev, next, "/2026/1", "")
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
			rec := serveSeason(SeasonHandler(tree, unusedSource{t}, noGames{}), Handler(tree, unusedSource{t}, noGames{}), http.MethodGet, tt.path)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("GET %s = %d, want %d", tt.path, rec.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestSeasonWithNoWeeksIsNotFound(t *testing.T) {
	tree := lineup.New(fstest.MapFS{})

	rec := serveSeason(SeasonHandler(tree, unusedSource{t}, noGames{}), Handler(tree, unusedSource{t}, noGames{}), http.MethodGet, "/2027")

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestSeasonGapStillShowsItsHighestWeek(t *testing.T) {
	tree := lineup.New(treeFS(matchupWeek(2026, 1), matchupWeek(2026, 3)))

	rec := serveSeason(SeasonHandler(tree, &fakeSource{weekStats: fixtureStats()}, noGames{}), Handler(tree, unusedSource{t}, noGames{}), http.MethodGet, "/2026")

	var heading string
	if m := headingPattern.FindStringSubmatch(rec.Body.String()); m != nil {
		heading = m[1]
	}
	if heading != "2026 · Week 3" {
		t.Errorf("h1 = %q, want %q", heading, "2026 · Week 3")
	}
}

func TestSeasonRefusesALatestWeekThatIsNotAMatchup(t *testing.T) {
	tree := lineup.New(weekFS(2026, 3, map[string]string{
		"aroma": lineupHeader, "bojjaes": lineupHeader, "wood": lineupHeader,
	}))

	rec := serveSeason(SeasonHandler(tree, unusedSource{t}, noGames{}), Handler(tree, unusedSource{t}, noGames{}), http.MethodGet, "/2026")

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestSeasonFetchesOnlyItsLatestWeek(t *testing.T) {
	tree := lineup.New(treeFS(matchupWeek(2026, 1), matchupWeek(2026, 2)))
	source := &fakeSource{weekStats: fixtureStats()}

	serveSeason(SeasonHandler(tree, source, noGames{}), Handler(tree, unusedSource{t}, noGames{}), http.MethodGet, "/2026")

	if want := [][2]int{{2026, 2}}; !slices.Equal(source.asked, want) {
		t.Errorf("provider asked for %v, want %v", source.asked, want)
	}
}

// The mux prefers the more specific pattern, so a season/week request must
// still reach the existing handler unchanged.
func TestSeasonWeekPathStillReachesTheExistingHandler(t *testing.T) {
	weekTree := weekFS(2026, 2, map[string]string{
		"bojjaes": lineupHeader + "1,Puka Nacua,WR,LAR\n",
		"wood":    lineupHeader + "2,Josh Allen,QB,BUF\n",
	})
	tree := lineup.New(weekTree)

	rec := serveSeason(SeasonHandler(tree, unusedSource{t}, noGames{}), Handler(tree, &fakeSource{}, noGames{}), http.MethodGet, "/2026/2")

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
