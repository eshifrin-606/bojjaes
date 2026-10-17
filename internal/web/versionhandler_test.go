package web

import (
	"encoding/json"
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

// End to end over a real statscache.Cache shared by the page and the endpoint,
// as main wires them. The clock starts off a whole second so a formatter that
// truncated on one side would show up as a mismatch.
func TestTheEndpointFollowsTheCacheThePageIsServedFrom(t *testing.T) {
	upstream := &countingStatsSource{weekStats: fixtureStats()}
	clock := &webFakeClock{t: asOfInstant.Add(123456789 * time.Nanosecond)}
	cache := weekcache.New(upstream, 5*time.Minute, weekcache.WithClock(clock.now))
	page := Handler(lineup.New(fixtureWeek()), cache)
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
