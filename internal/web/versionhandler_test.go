package web

import (
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/eshifrin/bojjaes/internal/lineup"
	"github.com/eshifrin/bojjaes/internal/weekcache"
)

// fakeVersions stands in for the cache: a fetch instant it holds or does not,
// and a count of how often it was asked.
type fakeVersions struct {
	fetchedAt time.Time
	held      bool

	asked [][2]int
}

func (f *fakeVersions) FetchedAt(season, week int) (time.Time, bool) {
	f.asked = append(f.asked, [2]int{season, week})
	return f.fetchedAt, f.held
}

// serveVersion routes through the pattern main registers, as serve does for
// the page.
func serveVersion(h http.Handler, method, path string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	mux.Handle("GET /{season}/{week}/version", h)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

// versionBody decodes the endpoint's JSON into its raw fields, so a test can
// tell a field that is empty from one that is missing.
func versionBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body %q is not JSON: %v", rec.Body, err)
	}
	return body
}

func TestTheEndpointReportsTheHeldFetchsVersion(t *testing.T) {
	source := &fakeVersions{fetchedAt: asOfInstant, held: true}

	body := versionBody(t, serveVersion(VersionHandler(source), http.MethodGet, "/2025/15/version"))

	if got, want := body["version"], versionToken(asOfInstant); got != want {
		t.Errorf("version = %#v, want the fetch instant's token %q", got, want)
	}
	if want := [][2]int{{2025, 15}}; !slices.Equal(source.asked, want) {
		t.Errorf("the handler asked for %v, want %v", source.asked, want)
	}
}

// Empty is the answer for every way of holding nothing — never fetched,
// invalidated, in flight, restarted — and the page reads it as a mismatch.
func TestTheEndpointReportsAnEmptyVersionWhenNothingIsHeld(t *testing.T) {
	body := versionBody(t, serveVersion(VersionHandler(&fakeVersions{}), http.MethodGet, "/2025/15/version"))

	if got, present := body["version"]; !present || got != "" {
		t.Errorf("version = %#v (present: %t), want an empty string", got, present)
	}
}

func TestTheEndpointStatesTheNextCheckDelay(t *testing.T) {
	for _, source := range []*fakeVersions{{}, {fetchedAt: asOfInstant, held: true}} {
		body := versionBody(t, serveVersion(VersionHandler(source), http.MethodGet, "/2025/15/version"))

		// encoding/json decodes every number into a float64.
		if got, want := body["nextCheckSeconds"], float64(checkDelaySeconds); got != want {
			t.Errorf("held=%t: nextCheckSeconds = %#v, want the default delay %v", source.held, got, want)
		}
	}
}

func TestTheEndpointsResponseIsJSONAndNotStored(t *testing.T) {
	rec := serveVersion(VersionHandler(&fakeVersions{fetchedAt: asOfInstant, held: true}), http.MethodGet, "/2025/15/version")

	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json")
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want %q", got, "no-store")
	}
}

func TestTheEndpointRefusesAMalformedPath(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "non-numeric week", path: "/2026/five/version"},
		{name: "non-numeric season", path: "/twenty26/5/version"},
		{name: "week past the season", path: "/2026/23/version"},
		{name: "season before the provider's records", path: "/1998/3/version"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := &fakeVersions{fetchedAt: asOfInstant, held: true}

			rec := serveVersion(VersionHandler(source), http.MethodGet, tt.path)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("GET %s = %d, want %d", tt.path, rec.Code, http.StatusBadRequest)
			}
			if strings.Contains(rec.Body.String(), versionToken(asOfInstant)) {
				t.Errorf("GET %s reported a version in a refusal: %s", tt.path, rec.Body)
			}
			if len(source.asked) != 0 {
				t.Errorf("GET %s asked the source for %v; a refusal must not reach it", tt.path, source.asked)
			}
		})
	}
}

// End to end over a real weekcache.Cache shared by the page and the endpoint,
// as main wires them. The clock starts off a whole second so a formatter that
// truncated on one side would show up as a mismatch.
func TestTheEndpointFollowsTheCacheThePageIsServedFrom(t *testing.T) {
	upstream := &countingStatsSource{weekStats: fixtureStats()}
	clock := &webFakeClock{t: asOfInstant.Add(123456789 * time.Nanosecond)}
	cache := weekcache.New("statscache", upstream.WeekStats, 5*time.Minute, weekcache.WithClock(clock.now))
	page := Handler(lineup.New(fixtureWeek()), cache, noGames{})
	endpoint := VersionHandler(cache)

	reported := func() any {
		t.Helper()
		return versionBody(t, serveVersion(endpoint, http.MethodGet, "/2025/15/version"))["version"]
	}

	if got := reported(); got != "" {
		t.Errorf("before any fetch the endpoint reports %#v, want an empty version", got)
	}
	if got := upstream.callCount(); got != 0 {
		t.Fatalf("upstream called %d times by the endpoint alone, want 0", got)
	}

	first := renderedVersion(t, serve(page, http.MethodGet, "/2025/15").Body.String())
	if got := reported(); got != first {
		t.Errorf("the endpoint reports %#v, want the fresh page's version %q", got, first)
	}

	cache.Invalidate(2025, 15)
	if got := reported(); got != "" {
		t.Errorf("after invalidation the endpoint reports %#v, want an empty version", got)
	}
	if got := upstream.callCount(); got != 1 {
		t.Fatalf("upstream called %d times, want 1: only the page may fetch", got)
	}

	clock.advance(3 * time.Second)
	second := renderedVersion(t, serve(page, http.MethodGet, "/2025/15").Body.String())
	if second == first {
		t.Errorf("the refetched page carries the first fetch's version %q", first)
	}
	if got := reported(); got != second {
		t.Errorf("after a refetch the endpoint reports %#v, want the new page's version %q", got, second)
	}
	if got := upstream.callCount(); got != 2 {
		t.Errorf("upstream called %d times, want 2", got)
	}
}

func serveSeasonVersion(h http.Handler, path string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	mux.Handle("GET /{season}/version", h)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestTheSeasonFormReportsTheVersionTheSeasonPageCarries(t *testing.T) {
	tree := lineup.New(treeFS(matchupWeek(2026, 1), matchupWeek(2026, 2)))
	upstream := &countingStatsSource{weekStats: fixtureStats()}
	cache := weekcache.New("statscache", upstream.WeekStats, 5*time.Minute)

	page := renderedVersion(t, serveSeason(SeasonHandler(tree, cache, noGames{}), Handler(tree, cache, noGames{}), http.MethodGet, "/2026").Body.String())
	body := versionBody(t, serveSeasonVersion(SeasonVersionHandler(tree, cache), "/2026/version"))

	if got := body["version"]; got != page {
		t.Errorf("version = %#v, want the season page's %q", got, page)
	}
}

// The tree is mutable under the handler, as the volume is under the server:
// the season form must resolve the latest week per request, not once.
func TestTheSeasonFormFollowsANewWeek(t *testing.T) {
	fsys := treeFS(matchupWeek(2026, 1), matchupWeek(2026, 2))
	tree := lineup.New(fsys)
	upstream := &countingStatsSource{weekStats: fixtureStats()}
	clock := &webFakeClock{t: asOfInstant}
	cache := weekcache.New("statscache", upstream.WeekStats, 5*time.Minute, weekcache.WithClock(clock.now))
	endpoint := SeasonVersionHandler(tree, cache)
	reported := func() any {
		t.Helper()
		return versionBody(t, serveSeasonVersion(endpoint, "/2026/version"))["version"]
	}
	seasonPage := func() string {
		t.Helper()
		return renderedVersion(t, serveSeason(SeasonHandler(tree, cache, noGames{}), Handler(tree, cache, noGames{}), http.MethodGet, "/2026").Body.String())
	}

	weekTwo := seasonPage()
	maps.Copy(fsys, matchupWeek(2026, 3))

	if got := reported(); got != "" {
		t.Errorf("with week 3 unfetched the endpoint reports %#v, want an empty version", got)
	}

	clock.advance(3 * time.Second)
	weekThree := seasonPage()
	if weekThree == weekTwo {
		t.Fatalf("week 3's page carries week 2's version %q", weekTwo)
	}
	if got := reported(); got != weekThree {
		t.Errorf("the endpoint reports %#v, want week 3's version %q", got, weekThree)
	}
}

type fakeLatestWeek struct {
	week int
	ok   bool

	asked []int
}

func (f *fakeLatestWeek) LatestWeek(season int) (int, bool) {
	f.asked = append(f.asked, season)
	return f.week, f.ok
}

func TestTheSeasonFormRefusesASeasonWithNoWeeks(t *testing.T) {
	source := &fakeVersions{fetchedAt: asOfInstant, held: true}

	rec := serveSeasonVersion(SeasonVersionHandler(&fakeLatestWeek{}, source), "/2026/version")

	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /2026/version = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if len(source.asked) != 0 {
		t.Errorf("the handler asked the source for %v; a refusal must not reach it", source.asked)
	}
}

func TestTheSeasonFormRefusesAMalformedSeason(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "non-numeric season", path: "/twentytwentysix/version"},
		{name: "season before the provider's records", path: "/1998/version"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			weeks := &fakeLatestWeek{week: 2, ok: true}
			source := &fakeVersions{fetchedAt: asOfInstant, held: true}

			rec := serveSeasonVersion(SeasonVersionHandler(weeks, source), tt.path)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("GET %s = %d, want %d", tt.path, rec.Code, http.StatusBadRequest)
			}
			if len(weeks.asked) != 0 {
				t.Errorf("GET %s resolved the latest week of %v; a refusal must not reach the resolver", tt.path, weeks.asked)
			}
			if len(source.asked) != 0 {
				t.Errorf("GET %s asked the source for %v; a refusal must not reach it", tt.path, source.asked)
			}
		})
	}
}

func TestTheSeasonFormsResponseIsNotStoredAndCallsNoProvider(t *testing.T) {
	tree := lineup.New(treeFS(matchupWeek(2026, 1), matchupWeek(2026, 2)))
	upstream := &countingStatsSource{weekStats: fixtureStats()}
	cache := weekcache.New("statscache", upstream.WeekStats, 5*time.Minute)

	rec := serveSeasonVersion(SeasonVersionHandler(tree, cache), "/2026/version")

	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want %q", got, "no-store")
	}
	if got := upstream.callCount(); got != 0 {
		t.Errorf("upstream called %d times by the endpoint, want 0", got)
	}
}
