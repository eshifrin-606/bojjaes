package web

import (
	"net/http"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/eshifrin/bojjaes/internal/lineup"
	"github.com/eshifrin/bojjaes/internal/statscache"
)

var (
	asOfTagPattern     = regexp.MustCompile(`<p class="as-of"([^>]*)>`)
	dataVersionPattern = regexp.MustCompile(`data-version="([^"]*)"`)
)

// asOfAttributes returns the attributes the as-of paragraph carries beyond
// its class.
func asOfAttributes(t *testing.T, body string) string {
	t.Helper()

	m := asOfTagPattern.FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("no <p class=\"as-of\"> element:\n%s", body)
	}
	return m[1]
}

// renderedVersion is the stats version on the as-of paragraph. It fails the
// test unless the whole page states exactly one.
func renderedVersion(t *testing.T, body string) string {
	t.Helper()

	if n := len(dataVersionPattern.FindAllString(body, -1)); n != 1 {
		t.Fatalf("page has %d data-version attributes, want exactly 1", n)
	}
	m := dataVersionPattern.FindStringSubmatch(asOfAttributes(t, body))
	if m == nil {
		t.Fatalf("the as-of paragraph carries no data-version:\n%s", body)
	}
	return m[1]
}

func TestThePageCarriesItsStatsVersion(t *testing.T) {
	source := &fakeSource{weekStats: fixtureStats(), fetchedAt: asOfInstant}
	rec := serve(Handler(lineup.New(fixtureWeek()), source), http.MethodGet, "/2025/15")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body)
	}

	want := strconv.FormatInt(asOfInstant.UnixNano(), 10)
	if got := renderedVersion(t, rec.Body.String()); got != want {
		t.Errorf("data-version = %q, want the fetch instant in Unix nanoseconds, %q", got, want)
	}
}

// The <time> element's RFC 3339 value would call these two the same fetch, and
// a page that did would miss the second one.
func TestTwoFetchesInOneSecondHaveDifferentVersions(t *testing.T) {
	versions := make([]string, 2)
	for i, fetchedAt := range []time.Time{asOfInstant.Add(100 * time.Millisecond), asOfInstant.Add(100*time.Millisecond + time.Nanosecond)} {
		source := &fakeSource{weekStats: fixtureStats(), fetchedAt: fetchedAt}
		rec := serve(Handler(lineup.New(fixtureWeek()), source), http.MethodGet, "/2025/15")
		versions[i] = renderedVersion(t, rec.Body.String())
	}

	if versions[0] == versions[1] {
		t.Errorf("two fetches a nanosecond apart both carry version %q, want them to differ", versions[0])
	}
}

var dataNextCheckPattern = regexp.MustCompile(`data-next-check="([^"]*)"`)

func TestThePageStatesItsFirstCheckDelay(t *testing.T) {
	source := &fakeSource{weekStats: fixtureStats(), fetchedAt: asOfInstant}
	rec := serve(Handler(lineup.New(fixtureWeek()), source), http.MethodGet, "/2025/15")

	m := dataNextCheckPattern.FindStringSubmatch(asOfAttributes(t, rec.Body.String()))
	if m == nil {
		t.Fatalf("the as-of paragraph carries no data-next-check:\n%s", asOfTagPattern.FindString(rec.Body.String()))
	}
	if want := strconv.Itoa(checkDelaySeconds); m[1] != want {
		t.Errorf("data-next-check = %q, want the default delay %q", m[1], want)
	}
}

func TestACacheHitCarriesTheOriginalFetchsVersion(t *testing.T) {
	upstream := &countingStatsSource{weekStats: fixtureStats()}
	clock := &webFakeClock{t: asOfInstant}
	cache := statscache.New(upstream, 5*time.Minute, statscache.WithClock(clock.now))
	h := Handler(lineup.New(fixtureWeek()), cache)

	first := renderedVersion(t, serve(h, http.MethodGet, "/2025/15").Body.String())
	clock.advance(3 * time.Minute)
	second := renderedVersion(t, serve(h, http.MethodGet, "/2025/15").Body.String())

	if first != second {
		t.Errorf("two pages from one cached fetch carry versions %q and %q, want the same", first, second)
	}
	if got := upstream.callCount(); got != 1 {
		t.Fatalf("upstream called %d times, want 1: the second page was not a cache hit", got)
	}
}
