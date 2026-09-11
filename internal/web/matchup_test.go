package web

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/eshifrin/bojjaes/internal/lineup"
	"github.com/eshifrin/bojjaes/internal/score"
	"github.com/eshifrin/bojjaes/internal/statscache"
)

// unusedSource fails the test if the page ever reaches the provider.
type unusedSource struct{ t *testing.T }

func (u unusedSource) WeekStatsAsOf(context.Context, int, int) (score.WeekStats, time.Time, error) {
	u.t.Helper()
	u.t.Error("provider was called for a request that should never have reached it")
	return score.WeekStats{}, time.Time{}, nil
}

// fakeSource stands in for the provider: a WeekStats built here rather than
// fetched, the fetch instant the page will stamp, and a count of how many times
// the page asked for one.
type fakeSource struct {
	weekStats score.WeekStats
	fetchedAt time.Time
	err       error

	calls int
}

func (f *fakeSource) WeekStatsAsOf(context.Context, int, int) (score.WeekStats, time.Time, error) {
	f.calls++
	if f.err != nil {
		return score.WeekStats{}, time.Time{}, f.err
	}
	return f.weekStats, f.fetchedAt, nil
}

func TestChicagoLocationLoadsAtInit(t *testing.T) {
	if chicagoLoc == nil {
		t.Fatal("chicagoLoc is nil; America/Chicago did not load")
	}
	if got := chicagoLoc.String(); got != "America/Chicago" {
		t.Errorf("chicagoLoc = %q, want %q", got, "America/Chicago")
	}
}

// weekFS is one week of a lineup tree, held in memory. Each entry is a team
// name mapped to that lineup file's contents.
func weekFS(season, week int, lineups map[string]string) fstest.MapFS {
	dir := fmt.Sprintf("%d/%d", season, week)
	fsys := fstest.MapFS{dir: &fstest.MapFile{Mode: fs.ModeDir}}
	for team, body := range lineups {
		fsys[dir+"/"+team+".csv"] = &fstest.MapFile{Data: []byte(body)}
	}
	return fsys
}

// A column of nine starters scoring 12, 0, 6, 3, 9, 0, 15, 4 and 7 — the spec's
// worked example, totalling 56 — against an opponent's seven touchdowns and two
// scoreless lines, totalling 42.
var (
	ourLine = []struct {
		id, name string
		points   float64
		stats    score.StatLine
	}{
		{id: "1", name: "Puka Nacua", points: 12, stats: score.StatLine{RushTD: 2}},
		{id: "2", name: "Bijan Robinson", points: 0},
		{id: "3", name: "Rachaad White", points: 6, stats: score.StatLine{RushTD: 1}},
		{id: "4", name: "Cam Little", points: 3, stats: score.StatLine{FGMade: 1}},
		{id: "5", name: "Jaxon Smith-Njigba", points: 9, stats: score.StatLine{RushTD: 1, FGMade: 1}},
		{id: "6", name: "Tyreek Hill", points: 0},
		{id: "7", name: "Brandon Aubrey", points: 15, stats: score.StatLine{FGMade: 5}},
		{id: "8", name: "Chase McLaughlin", points: 4, stats: score.StatLine{XPMade: 4}},
		{id: "9", name: "Kyren Williams", points: 7, stats: score.StatLine{RushTD: 1, XPMade: 1}},
	}

	theirLine = []struct {
		id, name string
		points   float64
		stats    score.StatLine
	}{
		{id: "11", name: "Josh Allen", points: 6, stats: score.StatLine{RushTD: 1}},
		{id: "12", name: "Saquon Barkley", points: 6, stats: score.StatLine{RushTD: 1}},
		{id: "13", name: "CeeDee Lamb", points: 6, stats: score.StatLine{RecTD: 1}},
		{id: "14", name: "Amon-Ra St. Brown", points: 6, stats: score.StatLine{RecTD: 1}},
		{id: "15", name: "Derrick Henry", points: 6, stats: score.StatLine{RushTD: 1}},
		{id: "16", name: "Malik Nabers", points: 6, stats: score.StatLine{RecTD: 1}},
		{id: "17", name: "Trey McBride", points: 6, stats: score.StatLine{RecTD: 1}},
		{id: "18", name: "Jayden Daniels", points: 0},
		{id: "19", name: "Ladd McConkey", points: 0},
	}
)

// lineupCSV writes one of the fixtures above as a lineup file.
func lineupCSV(records []struct {
	id, name string
	points   float64
	stats    score.StatLine
}) string {
	var b strings.Builder
	for _, r := range records {
		fmt.Fprintf(&b, "%s,%s\n", r.id, r.name)
	}
	return b.String()
}

// fixtureStats is the payload both fixture lineups are scored from. A record
// the caller leaves out is a player the provider has no entry for.
func fixtureStats(omit ...string) score.WeekStats {
	skip := make(map[string]bool, len(omit))
	for _, id := range omit {
		skip[id] = true
	}

	players := make(map[string]score.StatLine)
	for _, r := range append(append([]struct {
		id, name string
		points   float64
		stats    score.StatLine
	}{}, ourLine...), theirLine...) {
		if skip[r.id] {
			continue
		}
		line := r.stats
		line.PlayerID = r.id
		players[r.id] = line
	}
	return score.NewWeekStats(2025, 15, players)
}

// fixtureWeek holds the two fixture lineups as a week of a lineup tree.
func fixtureWeek() fstest.MapFS {
	return weekFS(2025, 15, map[string]string{
		"bojjaes": lineupCSV(ourLine),
		"wood":    lineupCSV(theirLine),
	})
}

// renderedStarters pulls each rendered starter line out of the page as
// "name=points". Reading the markup lives here alone, so a restyle touches one
// helper rather than every assertion.
var starterPattern = regexp.MustCompile(`(?s)<li>.*?class="player">(.*?)</span>.*?class="points">(.*?)</span>.*?</li>`)

func renderedStarters(body string) []string {
	var lines []string
	for _, m := range starterPattern.FindAllStringSubmatch(body, -1) {
		lines = append(lines, m[1]+"="+m[2])
	}
	return lines
}

var classPattern = regexp.MustCompile(`class="([^"]*)"`)

// timePattern pulls each <time> element's datetime attribute and visible text.
var timePattern = regexp.MustCompile(`(?s)<time datetime="([^"]*)">(.*?)</time>`)

func renderedTimes(body string) [][2]string {
	var out [][2]string
	for _, m := range timePattern.FindAllStringSubmatch(body, -1) {
		out = append(out, [2]string{m[1], m[2]})
	}
	return out
}

// asOfInstant is a fixed, non-zero fetch instant tests stamp on the fake so the
// rendered timestamp is a known value. 18:24 UTC on 2026-09-07 is 1:24 PM in
// Chicago, in CDT — a summer instant, so the zone abbreviation is unambiguous.
var asOfInstant = time.Date(2026, time.September, 7, 18, 24, 0, 0, time.UTC)

// asOfChicagoText is asOfInstant as the Chicago wall-clock string the page
// shows a reader.
const asOfChicagoText = "Mon, Sep 7 2026 1:24 PM CDT"

var tagPattern = regexp.MustCompile(`<[^>]*>`)

func visibleText(body string) string {
	return tagPattern.ReplaceAllString(body, " ")
}

// renderedTotals pulls the two column totals out of the page.
var totalPattern = regexp.MustCompile(`class="total">(.*?)</`)

func renderedTotals(body string) []string {
	var totals []string
	for _, m := range totalPattern.FindAllStringSubmatch(body, -1) {
		totals = append(totals, m[1])
	}
	return totals
}

// serve routes through the same mux pattern main registers, so the tests cover
// the segment wildcards and the method qualifier rather than the handler alone.
func serve(h http.Handler, method, path string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	mux.Handle("GET /{season}/{week}", h)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func TestBadRequestPaths(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "non-numeric week", path: "/2025/fifteen"},
		{name: "week past the season", path: "/2025/23"},
		{name: "season before the provider's records", path: "/1998/3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// An empty tree and a provider that fails the test if called: a
			// refusal that touched either would show up as something other
			// than a 400.
			h := Handler(lineup.New(fstest.MapFS{}), unusedSource{t})

			if rec := serve(h, http.MethodGet, tt.path); rec.Code != http.StatusBadRequest {
				t.Errorf("GET %s = %d, want %d", tt.path, rec.Code, http.StatusBadRequest)
			}
		})
	}
}

// The 405 comes from the mux's method-qualified pattern, not from the handler.
// The test pins the registration: dropping the GET would make the page answer
// a POST.
func TestPostIsNotAllowed(t *testing.T) {
	h := Handler(lineup.New(fstest.MapFS{}), unusedSource{t})

	if rec := serve(h, http.MethodPost, "/2025/15"); rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /2025/15 = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestBothTeamsAppearWithUsFirst(t *testing.T) {
	weekTree := weekFS(2025, 15, map[string]string{
		"bojjaes": "9493,Puka Nacua\n",
		"wood":    "8138,Bijan Robinson\n",
	})

	rec := serve(Handler(lineup.New(weekTree), &fakeSource{}), http.MethodGet, "/2025/15")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body)
	}

	body := rec.Body.String()
	ours, theirs := strings.Index(body, "bojjaes"), strings.Index(body, "wood")
	switch {
	case ours < 0 || theirs < 0:
		t.Fatalf("body names bojjaes at %d and wood at %d; want both present:\n%s", ours, theirs, body)
	case ours > theirs:
		t.Errorf("wood appears before bojjaes; the Bojjaes hold the left column")
	}
}

// The column order comes from Matchup, which knows which lineup is ours, and
// never from the directory listing.
func TestAlphabeticallyEarlierOpponentStaysOnTheRight(t *testing.T) {
	weekTree := weekFS(2025, 15, map[string]string{
		"bojjaes":   "9493,Puka Nacua\n",
		"aardvarks": "8138,Bijan Robinson\n",
	})

	rec := serve(Handler(lineup.New(weekTree), &fakeSource{}), http.MethodGet, "/2025/15")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body)
	}

	body := rec.Body.String()
	if ours, theirs := strings.Index(body, "bojjaes"), strings.Index(body, "aardvarks"); ours > theirs {
		t.Errorf("aardvarks appears before bojjaes; the Bojjaes hold the left column")
	}
}

func TestWeekRefusals(t *testing.T) {
	tests := []struct {
		name    string
		lineups map[string]string
		week    int
		want    int
	}{
		{
			name: "no such week directory",
			week: 17,
			want: http.StatusNotFound,
		},
		{
			name: "three lineups",
			lineups: map[string]string{
				"bojjaes": "9493,Puka Nacua\n",
				"wood":    "8138,Bijan Robinson\n",
				"aroma":   "7591,Rachaad White\n",
			},
			week: 15,
			want: http.StatusInternalServerError,
		},
		{
			name:    "one lineup",
			lineups: map[string]string{"bojjaes": "9493,Puka Nacua\n"},
			week:    15,
			want:    http.StatusInternalServerError,
		},
		{
			name: "a matchup we are not in",
			lineups: map[string]string{
				"wood":  "8138,Bijan Robinson\n",
				"aroma": "7591,Rachaad White\n",
			},
			week: 15,
			want: http.StatusInternalServerError,
		},
		{
			name: "a lineup line with no id",
			lineups: map[string]string{
				"bojjaes": "9493,Puka Nacua\n",
				"wood":    "8138,Bijan Robinson\n,Rachaad White\n",
			},
			week: 15,
			want: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			weekTree := weekFS(2025, 15, tt.lineups)

			// No refusal reaches the provider: there is no reason to fetch a
			// week we already know we cannot render.
			rec := serve(Handler(lineup.New(weekTree), unusedSource{t}), http.MethodGet, "/2025/"+strconv.Itoa(tt.week))
			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d", rec.Code, tt.want)
			}
			body := rec.Body.String()
			if strings.Contains(body, "<ol>") {
				t.Errorf("a refusal rendered a scoreboard:\n%s", body)
			}
			// The name of the file that failed goes to the log, which reaches
			// whoever runs the server; the body reaches whoever asked for the
			// page, and tells them nothing about where lineups are kept.
			if strings.Contains(body, ".csv") {
				t.Errorf("the response body names a lineup file:\n%s", body)
			}
		})
	}
}

func TestStartersRenderWithTheirPoints(t *testing.T) {
	rec := serve(Handler(lineup.New(fixtureWeek()), &fakeSource{weekStats: fixtureStats()}), http.MethodGet, "/2025/15")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body)
	}

	want := []string{
		"Puka Nacua=12", "Bijan Robinson=0", "Rachaad White=6",
		"Cam Little=3", "Jaxon Smith-Njigba=9", "Tyreek Hill=0",
		"Brandon Aubrey=15", "Chase McLaughlin=4", "Kyren Williams=7",
		"Josh Allen=6", "Saquon Barkley=6", "CeeDee Lamb=6",
		"Amon-Ra St. Brown=6", "Derrick Henry=6", "Malik Nabers=6",
		"Trey McBride=6", "Jayden Daniels=0", "Ladd McConkey=0",
	}
	if got := renderedStarters(rec.Body.String()); !slices.Equal(got, want) {
		t.Errorf("rendered starters:\n got %q\nwant %q", got, want)
	}
}

func TestEachColumnTotalsItsStarters(t *testing.T) {
	rec := serve(Handler(lineup.New(fixtureWeek()), &fakeSource{weekStats: fixtureStats()}), http.MethodGet, "/2025/15")

	want := []string{"56", "42"}
	if got := renderedTotals(rec.Body.String()); !slices.Equal(got, want) {
		t.Errorf("totals = %q, want %q", got, want)
	}
}

func TestBenchPlayersAreNotRendered(t *testing.T) {
	bench := "20,First Bench\n21,Second Bench\n22,Third Bench\n"
	weekTree := weekFS(2025, 15, map[string]string{
		"bojjaes": lineupCSV(ourLine) + bench,
		"wood":    lineupCSV(theirLine),
	})

	rec := serve(Handler(lineup.New(weekTree), &fakeSource{weekStats: fixtureStats()}), http.MethodGet, "/2025/15")

	body := rec.Body.String()
	for _, name := range []string{"First Bench", "Second Bench", "Third Bench"} {
		if strings.Contains(body, name) {
			t.Errorf("bench player %q appears on the page", name)
		}
	}
	if got := renderedTotals(body); !slices.Equal(got, []string{"56", "42"}) {
		t.Errorf("totals = %q, want the starters-only totals [56 42]", got)
	}
}

func TestOneFetchServesBothColumns(t *testing.T) {
	source := &fakeSource{weekStats: fixtureStats()}

	serve(Handler(lineup.New(fixtureWeek()), source), http.MethodGet, "/2025/15")

	if source.calls != 1 {
		t.Errorf("provider called %d times, want 1: the two columns must come from one snapshot", source.calls)
	}
}

func TestAFailedFetchServesNoPage(t *testing.T) {
	source := &fakeSource{err: errors.New("upstream is down")}

	rec := serve(Handler(lineup.New(fixtureWeek()), source), http.MethodGet, "/2025/15")

	if rec.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadGateway)
	}
	if body := rec.Body.String(); strings.Contains(body, "<ol>") {
		t.Errorf("a failed fetch rendered a scoreboard:\n%s", body)
	}
}

// The provider's payload cannot say whether a missing player has not kicked
// off, is inactive, or does not exist, so absence is shown as absence.
func TestAMissingStarterIsNotZero(t *testing.T) {
	// Brandon Aubrey, who would have scored 15, has no entry this week.
	rec := serve(Handler(lineup.New(fixtureWeek()), &fakeSource{weekStats: fixtureStats("7")}), http.MethodGet, "/2025/15")

	body := rec.Body.String()
	if !slices.Contains(renderedStarters(body), "Brandon Aubrey=--") {
		t.Errorf("absent starter did not render the placeholder; got %q", renderedStarters(body))
	}
	if slices.Contains(renderedStarters(body), "Brandon Aubrey=0") {
		t.Error("absent starter rendered as 0, which claims he played and scored nothing")
	}
	// The terminal report still says "no stats"; the page must not.
	if strings.Contains(body, "no stats") {
		t.Errorf("the page still renders the old placeholder wording:\n%s", body)
	}
}

func TestAnAbsentStarterContributesNothingToTheTotal(t *testing.T) {
	rec := serve(Handler(lineup.New(fixtureWeek()), &fakeSource{weekStats: fixtureStats("7")}), http.MethodGet, "/2025/15")

	// 56 less Aubrey's 15: the other eight starters, and nothing for him.
	if got := renderedTotals(rec.Body.String()); !slices.Equal(got, []string{"41", "42"}) {
		t.Errorf("totals = %q, want [41 42]", got)
	}
}

// A total is always a number: a column with nothing to add up totals 0, not the
// placeholder its starter lines carry.
func TestAColumnWithNoStatsAtAllTotalsZero(t *testing.T) {
	rec := serve(Handler(lineup.New(fixtureWeek()), &fakeSource{weekStats: fixtureStats("11", "12", "13", "14", "15", "16", "17", "18", "19")}), http.MethodGet, "/2025/15")
	body := rec.Body.String()

	starters := renderedStarters(body)
	if len(starters) != 18 {
		t.Fatalf("rendered %d starters, want 18: %q", len(starters), starters)
	}
	for _, line := range starters[9:] {
		if !strings.HasSuffix(line, "=--") {
			t.Errorf("starter with no stats rendered %q, want the placeholder", line)
		}
	}
	if got := renderedTotals(body); !slices.Equal(got, []string{"56", "0"}) {
		t.Errorf("totals = %q, want [56 0]", got)
	}
}

// A scoreless week is a real line, not an absence: Jayden Daniels is in the
// payload having done nothing.
func TestAScorelessStarterIsZero(t *testing.T) {
	rec := serve(Handler(lineup.New(fixtureWeek()), &fakeSource{weekStats: fixtureStats()}), http.MethodGet, "/2025/15")

	if !slices.Contains(renderedStarters(rec.Body.String()), "Jayden Daniels=0") {
		t.Errorf("scoreless starter did not render 0; got %q", renderedStarters(rec.Body.String()))
	}
}

// A starter whose game has not kicked off looks exactly like one who was
// inactive, so anything implying a winner would present an unsettled reading
// as a result.
func TestThePageShowsNoMargin(t *testing.T) {
	rec := serve(Handler(lineup.New(fixtureWeek()), &fakeSource{weekStats: fixtureStats()}), http.MethodGet, "/2025/15")
	body := rec.Body.String()

	if got := renderedTotals(body); !slices.Equal(got, []string{"56", "42"}) {
		t.Fatalf("totals = %q, want [56 42]", got)
	}

	// 56 - 42. It appears nowhere as a rendered value, and no element carries
	// a difference under any name.
	for _, line := range append(renderedStarters(body), renderedTotals(body)...) {
		if strings.HasSuffix(line, "=14") || line == "14" {
			t.Errorf("the margin is rendered as a value: %q", line)
		}
	}
	// Every value the page renders is a starter's points or a column total.
	// The difference is not among them, and there is no third kind of value
	// for it to hide in.
	for _, line := range append(renderedStarters(body), renderedTotals(body)...) {
		if strings.HasSuffix(line, "=14") || line == "14" {
			t.Errorf("the margin is rendered as a value: %q", line)
		}
	}
}

// The two columns differ only in their content: one class, used twice, with
// nothing keyed on which total is larger.
func TestTheTwoColumnsCarryTheSameMarkup(t *testing.T) {
	rec := serve(Handler(lineup.New(fixtureWeek()), &fakeSource{weekStats: fixtureStats()}), http.MethodGet, "/2025/15")
	body := rec.Body.String()

	if got := strings.Count(body, `class="column"`); got != 2 {
		t.Errorf("class=\"column\" appears %d times, want exactly 2", got)
	}
	// Class names carry a leader vocabulary only if someone adds one; the CSS
	// has no rule for such a class to reach.
	for _, m := range classPattern.FindAllStringSubmatch(body, -1) {
		for _, word := range []string{"win", "lead", "los", "ahead", "behind", "trail"} {
			if strings.Contains(strings.ToLower(m[1]), word) {
				t.Errorf("class %q carries a leader vocabulary", m[1])
			}
		}
	}
}

// Names reach the page from file names and hand-edited CSV, so contextual
// escaping is the only thing between a typo and injected markup.
func TestLineupTextIsEscaped(t *testing.T) {
	weekTree := weekFS(2025, 15, map[string]string{
		"bojjaes": "1,<b>Puka</b> Nacua\n",
		"wood":    "11,Josh Allen\n",
	})

	rec := serve(Handler(lineup.New(weekTree), &fakeSource{weekStats: fixtureStats()}), http.MethodGet, "/2025/15")

	body := rec.Body.String()
	if strings.Contains(body, "<b>") {
		t.Errorf("a lineup name was rendered as markup:\n%s", body)
	}
	if !strings.Contains(body, "&lt;b&gt;Puka&lt;/b&gt; Nacua") {
		t.Errorf("the name was not rendered as escaped text:\n%s", body)
	}
}

func TestThePageIsServedAsHTML(t *testing.T) {
	rec := serve(Handler(lineup.New(fixtureWeek()), &fakeSource{weekStats: fixtureStats()}), http.MethodGet, "/2025/15")

	if got := rec.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want an HTML type", got)
	}
}

func TestThePageStampsTheFetchInstantAsRFC3339(t *testing.T) {
	source := &fakeSource{weekStats: fixtureStats(), fetchedAt: asOfInstant}
	rec := serve(Handler(lineup.New(fixtureWeek()), source), http.MethodGet, "/2025/15")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body)
	}

	times := renderedTimes(rec.Body.String())
	if len(times) != 1 {
		t.Fatalf("page has %d <time> elements, want exactly 1", len(times))
	}
	if got, want := times[0][0], asOfInstant.Format(time.RFC3339); got != want {
		t.Errorf("datetime attribute = %q, want %q", got, want)
	}
}

// The datetime attribute is a machine instant: it must parse back to the same
// moment, which pins that the handler keeps the offset rather than rendering a
// bare Chicago wall-clock string.
func TestTheDatetimeAttributeIsAnUnambiguousInstant(t *testing.T) {
	source := &fakeSource{weekStats: fixtureStats(), fetchedAt: asOfInstant}
	rec := serve(Handler(lineup.New(fixtureWeek()), source), http.MethodGet, "/2025/15")

	times := renderedTimes(rec.Body.String())
	if len(times) != 1 {
		t.Fatalf("page has %d <time> elements, want exactly 1", len(times))
	}

	parsed, err := time.Parse(time.RFC3339, times[0][0])
	if err != nil {
		t.Fatalf("datetime %q does not parse as RFC 3339: %v", times[0][0], err)
	}
	if !parsed.Equal(asOfInstant) {
		t.Errorf("datetime %q denotes %v, want the fetch instant %v", times[0][0], parsed.UTC(), asOfInstant)
	}
}

func TestThePageShowsTheFetchInstantInChicagoTime(t *testing.T) {
	source := &fakeSource{weekStats: fixtureStats(), fetchedAt: asOfInstant}
	rec := serve(Handler(lineup.New(fixtureWeek()), source), http.MethodGet, "/2025/15")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body)
	}

	if got := visibleText(rec.Body.String()); !strings.Contains(got, asOfChicagoText) {
		t.Errorf("visible text does not contain the Chicago wall-clock time %q:\n%s", asOfChicagoText, got)
	}
}

func TestTheAsOfLineIsLabelledAsFetchedAndDoesNotOverclaimCurrency(t *testing.T) {
	source := &fakeSource{weekStats: fixtureStats(), fetchedAt: asOfInstant}
	rec := serve(Handler(lineup.New(fixtureWeek()), source), http.MethodGet, "/2025/15")

	body := rec.Body.String()
	text := visibleText(body)

	label, _, found := strings.Cut(text, asOfChicagoText)
	if !found {
		t.Fatalf("Chicago wall-clock time not in visible text:\n%s", text)
	}
	if !strings.Contains(strings.ToLower(label), "fetched") {
		t.Errorf("the text before the timestamp does not say it was fetched: %q", label)
	}

	lower := strings.ToLower(body)
	for _, overclaim := range []string{"live", "current"} {
		if strings.Contains(lower, overclaim) {
			t.Errorf("the page contains %q, which overclaims how current it is", overclaim)
		}
	}
}

var asOfLinePattern = regexp.MustCompile(`(?s)<p class="as-of">(.*?)</p>`)

// The as-of line states a fetch time and nothing about the game: the no-winner
// rule that governs the two columns governs this element too.
func TestTheAsOfLineImpliesNoWinner(t *testing.T) {
	source := &fakeSource{weekStats: fixtureStats(), fetchedAt: asOfInstant}
	rec := serve(Handler(lineup.New(fixtureWeek()), source), http.MethodGet, "/2025/15")
	body := rec.Body.String()

	if got := renderedTotals(body); !slices.Equal(got, []string{"56", "42"}) {
		t.Fatalf("totals = %q, want [56 42]", got)
	}

	m := asOfLinePattern.FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("no <p class=\"as-of\"> element:\n%s", body)
	}
	lineText := visibleText(m[1])
	for _, forbidden := range []string{"56", "42", "14", "-"} {
		if strings.Contains(lineText, forbidden) {
			t.Errorf("the as-of line's text %q contains %q — it must not reference either total or a margin", lineText, forbidden)
		}
	}

	// The element is a sibling of <main>, not a descendant of a column.
	if strings.Index(body, `<p class="as-of">`) < strings.Index(body, "</main>") {
		t.Errorf("the as-of line is inside <main>; it must sit outside both columns")
	}
	afterMain := body[strings.Index(body, "</main>"):]
	if strings.Contains(asOfLinePattern.FindString(afterMain), `class="column"`) {
		t.Errorf("the as-of line is nested in a column section")
	}
}

// countingStatsSource is the plain StatsSource the real cache wraps: a fixed
// payload and a count of upstream calls.
type countingStatsSource struct {
	mu        sync.Mutex
	calls     int
	weekStats score.WeekStats
}

func (c *countingStatsSource) WeekStats(context.Context, int, int) (score.WeekStats, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	return c.weekStats, nil
}

func (c *countingStatsSource) callCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls
}

type webFakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *webFakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *webFakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

// End to end over a real statscache.Cache: a page served from a cache hit dates
// its stats to the fetch, not to the request it answered, and does not spend a
// second upstream call to do it.
func TestAServedCacheHitDatesStatsToTheFetchNotTheRequest(t *testing.T) {
	upstream := &countingStatsSource{weekStats: fixtureStats()}
	clock := &webFakeClock{t: asOfInstant}
	cache := statscache.New(upstream, 5*time.Minute, statscache.WithClock(clock.now))
	h := Handler(lineup.New(fixtureWeek()), cache)

	first := serve(h, http.MethodGet, "/2025/15")
	if first.Code != http.StatusOK {
		t.Fatalf("first request status = %d, want 200 (body: %s)", first.Code, first.Body)
	}

	clock.advance(3 * time.Minute)

	second := serve(h, http.MethodGet, "/2025/15")
	if second.Code != http.StatusOK {
		t.Fatalf("second request status = %d, want 200 (body: %s)", second.Code, second.Body)
	}

	firstTimes, secondTimes := renderedTimes(first.Body.String()), renderedTimes(second.Body.String())
	if len(firstTimes) != 1 || len(secondTimes) != 1 {
		t.Fatalf("want one <time> element per page, got %d and %d", len(firstTimes), len(secondTimes))
	}
	if firstTimes[0][0] != secondTimes[0][0] {
		t.Errorf("second page dated to %q, first to %q — a cache hit must report the fetch instant", secondTimes[0][0], firstTimes[0][0])
	}
	if want := asOfInstant.Format(time.RFC3339); secondTimes[0][0] != want {
		t.Errorf("served instant = %q, want the fetch instant %q", secondTimes[0][0], want)
	}

	if got := upstream.callCount(); got != 1 {
		t.Errorf("upstream called %d times; the as-of path must not bypass the cache", got)
	}
}

// The script element's contents. Every assertion about the refresh goes
// through here, so a test looking for "visible" cannot be satisfied by the
// word appearing somewhere else in the document.
var scriptPattern = regexp.MustCompile(`(?s)<script[^>]*>(.*?)</script>`)

func renderedScript(t *testing.T, body string) string {
	t.Helper()

	found := scriptPattern.FindAllStringSubmatch(body, -1)
	if len(found) != 1 {
		t.Fatalf("page has %d script elements, want exactly 1", len(found))
	}
	return found[0][1]
}

func TestThePageCarriesARefreshScript(t *testing.T) {
	rec := serve(Handler(lineup.New(fixtureWeek()), &fakeSource{weekStats: fixtureStats()}), http.MethodGet, "/2025/15")

	if script := renderedScript(t, rec.Body.String()); strings.TrimSpace(script) == "" {
		t.Error("the page's script element is empty")
	}
}

func TestTheRefreshIntervalIsFiveMinutes(t *testing.T) {
	rec := serve(Handler(lineup.New(fixtureWeek()), &fakeSource{weekStats: fixtureStats()}), http.MethodGet, "/2025/15")

	script := renderedScript(t, rec.Body.String())
	if !strings.Contains(script, "5 * 60 * 1000") {
		t.Errorf("script does not refresh on a five-minute interval:\n%s", script)
	}
}

// A hidden tab must cost nothing upstream, so the reload is conditioned on the
// tab being visible at the moment the timer fires.
func TestTheRefreshIsGuardedByTabVisibility(t *testing.T) {
	rec := serve(Handler(lineup.New(fixtureWeek()), &fakeSource{weekStats: fixtureStats()}), http.MethodGet, "/2025/15")

	script := renderedScript(t, rec.Body.String())
	for _, want := range []string{"document.visibilityState", `"visible"`} {
		if !strings.Contains(script, want) {
			t.Errorf("script does not consult %s:\n%s", want, script)
		}
	}
}

func TestTheScriptListensForTheTabComingBack(t *testing.T) {
	rec := serve(Handler(lineup.New(fixtureWeek()), &fakeSource{weekStats: fixtureStats()}), http.MethodGet, "/2025/15")

	script := renderedScript(t, rec.Body.String())
	if !strings.Contains(script, "visibilitychange") {
		t.Errorf("script registers no visibilitychange listener:\n%s", script)
	}
}

// Returning to the tab reloads only when the page is already older than the
// interval, so a glance away costs nothing.
func TestReturningToTheTabRefreshesOnlyWhenStale(t *testing.T) {
	rec := serve(Handler(lineup.New(fixtureWeek()), &fakeSource{weekStats: fixtureStats()}), http.MethodGet, "/2025/15")

	script := renderedScript(t, rec.Body.String())
	if !strings.Contains(script, "Date.now()") {
		t.Errorf("script captures no load time from the client clock:\n%s", script)
	}

	_, listener, found := strings.Cut(script, "visibilitychange")
	if !found {
		t.Fatalf("script registers no visibilitychange listener:\n%s", script)
	}
	if !strings.Contains(listener, "refreshInterval") {
		t.Errorf("the return path does not compare elapsed time against the refresh interval:\n%s", listener)
	}
}

// Team and player names reach the page from file names and hand-edited CSV.
// Keeping them out of the script keeps every one of them in the HTML text
// contexts the escaping requirement already covers.
func TestNoLineupTextReachesTheScript(t *testing.T) {
	rec := serve(Handler(lineup.New(fixtureWeek()), &fakeSource{weekStats: fixtureStats()}), http.MethodGet, "/2025/15")

	script := renderedScript(t, rec.Body.String())
	names := []string{"bojjaes", "wood"}
	for _, line := range renderedStarters(rec.Body.String()) {
		names = append(names, strings.Split(line, "=")[0])
	}
	for _, name := range names {
		if strings.Contains(script, name) {
			t.Errorf("%q appears inside the script:\n%s", name, script)
		}
	}
}

// The strongest form of the no-interpolation rule: nothing about the week can
// change the script, so an interpolation added later shows up here.
func TestTheScriptIsIdenticalAcrossWeeks(t *testing.T) {
	first := serve(Handler(lineup.New(fixtureWeek()), &fakeSource{weekStats: fixtureStats()}), http.MethodGet, "/2025/15")

	otherFS := weekFS(2024, 3, map[string]string{
		"bojjaes": "31,Ja'Marr Chase\n",
		"aroma":   "32,Travis Kelce\n",
	})
	otherStats := score.NewWeekStats(2024, 3, map[string]score.StatLine{
		"31": {PlayerID: "31", RecTD: 3},
		"32": {PlayerID: "32", RecTD: 1},
	})
	second := serve(Handler(lineup.New(otherFS), &fakeSource{weekStats: otherStats}), http.MethodGet, "/2024/3")

	if a, b := renderedScript(t, first.Body.String()), renderedScript(t, second.Body.String()); a != b {
		t.Errorf("the script differs between weeks:\n%s\n---\n%s", a, b)
	}
}

// The no-winner rule binds the script too: a refresh must not become the route
// by which the page starts marking which total moved.
func TestTheScriptImpliesNoWinner(t *testing.T) {
	rec := serve(Handler(lineup.New(fixtureWeek()), &fakeSource{weekStats: fixtureStats()}), http.MethodGet, "/2025/15")

	script := strings.ToLower(renderedScript(t, rec.Body.String()))
	for _, word := range []string{"win", "lead", "los", "ahead", "behind", "trail", "margin", "diff", "total", "score"} {
		if strings.Contains(script, word) {
			t.Errorf("the script carries a leader vocabulary (%q):\n%s", word, script)
		}
	}
	// Nothing survives a reload, so there is nothing to compare a new total
	// against — and no storage in which to keep one.
	for _, api := range []string{"localStorage", "sessionStorage", "document.cookie"} {
		if strings.Contains(script, strings.ToLower(api)) {
			t.Errorf("the script stashes state across loads via %s:\n%s", api, script)
		}
	}
}
