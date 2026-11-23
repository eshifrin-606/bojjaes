package web

import (
	"context"
	"errors"
	"html"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/eshifrin/bojjaes/internal/games"
	"github.com/eshifrin/bojjaes/internal/lineup"
	"github.com/eshifrin/bojjaes/internal/nflweek"
	"github.com/eshifrin/bojjaes/internal/volume"
)

// stubCurrent stands in for the current-week source, counting how often the
// page asked.
type stubCurrent struct {
	current nflweek.Current
	err     error
	calls   int
}

func (s *stubCurrent) AsOf(context.Context) (nflweek.Current, time.Time, error) {
	s.calls++
	return s.current, time.Time{}, s.err
}

// woodRoster fills every slot once, with a spare RB, FB, WR, and CB so each
// select has more than one candidate and FLEX and D have something to tell
// apart.
const woodRoster = lineupHeader +
	"101,Josh Allen,QB,BUF\n" +
	"102,Bijan Robinson,RB,ATL\n" +
	"103,Derrick Henry,RB,BAL\n" +
	"104,Kyle Juszczyk,FB,SF\n" +
	"105,Amon-Ra St. Brown,WR,DET\n" +
	"106,Free Agent Wideout,WR,FA\n" +
	"107,Trey McBride,TE,ARI\n" +
	"108,Brandon Aubrey,K,DAL\n" +
	"109,Myles Garrett,DE,CLE\n" +
	"110,Sauce Gardner,CB,NYJ\n" +
	"111,Patrick Surtain,CB,DEN\n"

const bojjaesRoster = lineupHeader +
	"201,Lamar Jackson,QB,BAL\n"

func testRosters() *lineup.Rosters {
	return lineup.NewRosters(fstest.MapFS{
		"bojjaes.csv": {Data: []byte(bojjaesRoster)},
		"wood.csv":    {Data: []byte(woodRoster)},
	})
}

func testLineupTree() *lineup.Tree {
	return lineup.New(fstest.MapFS{})
}

// lineupFixture is one request's sources; tests change a field before get.
type lineupFixture struct {
	rosters *lineup.Rosters
	tree    *lineup.Tree
	current *stubCurrent
	games   *fakeGames
	saver   LineupSaver
}

func newLineupFixture() *lineupFixture {
	return &lineupFixture{
		rosters: testRosters(),
		tree:    testLineupTree(),
		current: &stubCurrent{current: nflweek.New(2026, 9, nflweek.Regular)},
		games:   &fakeGames{week: games.NewWeek(2026, 9, nil)},
	}
}

func (f *lineupFixture) get(url string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	LineupHandler(f.rosters, f.tree, f.current, f.games, f.saver).ServeHTTP(rec, httptest.NewRequest("GET", url, nil))
	return rec
}

func TestTheFormNamesTheCurrentWeek(t *testing.T) {
	body := newLineupFixture().get("/lineup").Body.String()

	if !strings.Contains(body, "2026 · Week 9") {
		t.Errorf("body does not name 2026 · Week 9:\n%s", body)
	}
}

func assertNoWeekToSet(t *testing.T, f *lineupFixture) {
	t.Helper()
	rec := f.get("/lineup")

	body := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(body, "No week to set") {
		t.Errorf("body does not say No week to set:\n%s", body)
	}
	if strings.Contains(body, `<select name="qb"`) {
		t.Error("body shows the slot selects")
	}
	if f.games.calls != 0 {
		t.Errorf("games source called %d times, want 0", f.games.calls)
	}
}

func TestTheOffseasonHasNoWeekToSet(t *testing.T) {
	f := newLineupFixture()
	f.current.current = nflweek.New(2026, 0, nflweek.Offseason)

	assertNoWeekToSet(t, f)
}

// Sleeper's week 18 is a regular-season week, but the league's season ends at
// lineup.LastWeek.
func TestNFLWeek18HasNoWeekToSet(t *testing.T) {
	f := newLineupFixture()
	f.current.current = nflweek.New(2026, 18, nflweek.Regular)

	assertNoWeekToSet(t, f)
}

func TestTheLeaguesLastWeekCanBeSet(t *testing.T) {
	f := newLineupFixture()
	f.current.current = nflweek.New(2026, 17, nflweek.Regular)

	body := f.get("/lineup").Body.String()

	if strings.Contains(body, "No week to set") || !strings.Contains(body, "2026 · Week 17") || len(slotSelectNames(body)) != lineup.StarterCount {
		t.Errorf("week 17 is not offered:\n%s", body)
	}
}

func TestAFailedCurrentWeekReadIsABadGateway(t *testing.T) {
	f := newLineupFixture()
	f.current.err = errors.New("sleeper is down")

	if rec := f.get("/lineup"); rec.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", rec.Code)
	}
}

func TestASeasonAndWeekInTheQueryAreIgnored(t *testing.T) {
	body := newLineupFixture().get("/lineup?season=2025&week=3").Body.String()

	if !strings.Contains(body, "2026 · Week 9") {
		t.Errorf("body does not name 2026 · Week 9:\n%s", body)
	}
}

func TestATeamWithNoRosterIsRefusedBeforeAnyFetch(t *testing.T) {
	for _, team := range []string{"nobody", "../wood"} {
		t.Run(team, func(t *testing.T) {
			f := newLineupFixture()

			rec := f.get("/lineup?team=" + url.QueryEscape(team))

			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400", rec.Code)
			}
			if f.current.calls != 0 || f.games.calls != 0 {
				t.Errorf("current called %d times, games %d; want 0 and 0", f.current.calls, f.games.calls)
			}
		})
	}
}

type option struct {
	value, text string
	selected    bool
}

var (
	selectPattern = regexp.MustCompile(`(?s)<select name="([^"]*)"[^>]*>(.*?)</select>`)
	optionPattern = regexp.MustCompile(`<option value="([^"]*)"( selected)?>([^<]*)</option>`)
)

// selects returns each select of the given name, in page order, as its
// options with their text unescaped.
func selects(body, name string) [][]option {
	var found [][]option
	for _, m := range selectPattern.FindAllStringSubmatch(body, -1) {
		if m[1] != name {
			continue
		}
		var opts []option
		for _, o := range optionPattern.FindAllStringSubmatch(m[2], -1) {
			opts = append(opts, option{value: html.UnescapeString(o[1]), text: html.UnescapeString(o[3]), selected: o[2] != ""})
		}
		found = append(found, opts)
	}
	return found
}

func onlySelect(t *testing.T, body, name string) []option {
	t.Helper()
	found := selects(body, name)
	if len(found) != 1 {
		t.Fatalf("found %d selects named %q, want 1:\n%s", len(found), name, body)
	}
	return found[0]
}

func TestThePickerDefaultsToOurTeam(t *testing.T) {
	body := newLineupFixture().get("/lineup").Body.String()

	want := []option{{value: "bojjaes", text: "bojjaes", selected: true}, {value: "wood", text: "wood"}}
	if got := onlySelect(t, body, "team"); !slices.Equal(got, want) {
		t.Errorf("team picker = %+v, want %+v", got, want)
	}
}

func TestThePickerSelectsTheRequestedTeam(t *testing.T) {
	body := newLineupFixture().get("/lineup?team=wood").Body.String()

	want := []option{{value: "bojjaes", text: "bojjaes"}, {value: "wood", text: "wood", selected: true}}
	if got := onlySelect(t, body, "team"); !slices.Equal(got, want) {
		t.Errorf("team picker = %+v, want %+v", got, want)
	}
	if !regexp.MustCompile(`<form method="get" action="/lineup">\s*<label for="team">`).MatchString(body) {
		t.Errorf("the team picker is not a GET form to /lineup:\n%s", body)
	}
}

// The picker submits itself on change, so a button would do nothing.
func TestTheTeamPickerHasNoButton(t *testing.T) {
	body := newLineupFixture().get("/lineup").Body.String()

	for _, m := range getFormPattern.FindAllStringSubmatch(body, -1) {
		if strings.Contains(m[1], `name="team" id="team"`) && strings.Contains(m[1], "<button") {
			t.Errorf("the team picker has a button:\n%s", m[1])
		}
	}
}

func slotSelectNames(body string) []string {
	var names []string
	for _, m := range selectPattern.FindAllStringSubmatch(body, -1) {
		if m[1] != "team" {
			names = append(names, m[1])
		}
	}
	return names
}

func TestTheSlotsAreNineSelectsInSlotOrder(t *testing.T) {
	body := newLineupFixture().get("/lineup?team=wood").Body.String()

	want := []string{"qb", "rb", "wr", "te", "flex", "flex", "k", "d", "d"}
	if got := slotSelectNames(body); !slices.Equal(got, want) {
		t.Errorf("slot selects = %q, want %q", got, want)
	}
}

func optionValues(opts []option) []string {
	var values []string
	for _, o := range opts {
		values = append(values, o.value)
	}
	return values
}

// slotValues is each select of the named slot as its option values.
func slotValues(body, name string) [][]string {
	var found [][]string
	for _, opts := range selects(body, name) {
		found = append(found, optionValues(opts))
	}
	return found
}

func TestTheQBSelectOffersOnlyTheQB(t *testing.T) {
	body := newLineupFixture().get("/lineup?team=wood").Body.String()

	want := [][]string{{"", "101"}}
	if got := slotValues(body, "qb"); !slices.EqualFunc(got, want, slices.Equal) {
		t.Errorf("qb options = %q, want %q", got, want)
	}
}

func TestFLEXAndDOfferEveryPositionTheyAccept(t *testing.T) {
	body := newLineupFixture().get("/lineup?team=wood").Body.String()

	flex := []string{"", "102", "103", "104", "105", "106", "107"}
	if got, want := slotValues(body, "flex"), [][]string{flex, flex}; !slices.EqualFunc(got, want, slices.Equal) {
		t.Errorf("flex options = %q, want %q", got, want)
	}
	d := []string{"", "109", "110", "111"}
	if got, want := slotValues(body, "d"), [][]string{d, d}; !slices.EqualFunc(got, want, slices.Equal) {
		t.Errorf("d options = %q, want %q", got, want)
	}
}

// optionText is the text of the named slot's first select's option for id.
func optionText(t *testing.T, body, slot, id string) string {
	t.Helper()
	found := selects(body, slot)
	if len(found) == 0 {
		t.Fatalf("no %s select:\n%s", slot, body)
	}
	for _, o := range found[0] {
		if o.value == id {
			return o.text
		}
	}
	t.Fatalf("the %s select does not offer %s: %+v", slot, id, found[0])
	return ""
}

func upcomingDET() games.Week {
	kickoff := time.Date(2026, 11, 8, 12, 0, 0, 0, chicagoLoc)
	return games.NewWeek(2026, 9, []games.Game{{Home: "DET", Away: "GB", State: games.Upcoming, Kickoff: kickoff, Status: "pre_game"}})
}

func TestAnOptionStatesItsPlayersUpcomingGame(t *testing.T) {
	f := newLineupFixture()
	f.games.week = upcomingDET()

	body := f.get("/lineup?team=wood").Body.String()

	if got, want := optionText(t, body, "wr", "105"), "Amon-Ra St. Brown · WR · DET · Sun 12p"; got != want {
		t.Errorf("option text = %q, want %q", got, want)
	}
}

func TestAnOptionWithNoGameEndsAtWhatIsKnown(t *testing.T) {
	tests := []struct {
		name     string
		week     games.Week
		err      error
		slot, id string
		want     string
	}{
		{name: "a team with no game in a week with games is on bye", week: upcomingDET(), slot: "qb", id: "101", want: "Josh Allen · QB · BUF · Bye"},
		{name: "a free agent has no game", week: upcomingDET(), slot: "wr", id: "106", want: "Free Agent Wideout · WR · FA"},
		{name: "a failed games read ends at the team", err: errors.New("sleeper is down"), slot: "wr", id: "105", want: "Amon-Ra St. Brown · WR · DET"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newLineupFixture()
			f.games.week, f.games.err = tt.week, tt.err

			rec := f.get("/lineup?team=wood")

			if rec.Code != http.StatusOK {
				t.Errorf("status = %d, want 200", rec.Code)
			}
			if got := optionText(t, rec.Body.String(), tt.slot, tt.id); got != tt.want {
				t.Errorf("option text = %q, want %q", got, tt.want)
			}
		})
	}
}

var (
	cardNamePattern = regexp.MustCompile(`(?s)<span class="chosen"[^>]*><span class="name">([^<]*)</span>`)
	cardMetaPattern = regexp.MustCompile(`(?s)<span class="chosen"[^>]*>.*?<span class="meta">(.*?)<select`)
)

// cardNames is each slot card's name, in page order.
func cardNames(body string) []string {
	var names []string
	for _, m := range cardNamePattern.FindAllStringSubmatch(body, -1) {
		names = append(names, html.UnescapeString(m[1]))
	}
	return names
}

// cardMetas is each slot card's meta line as text, in page order.
func cardMetas(body string) []string {
	var metas []string
	for _, m := range cardMetaPattern.FindAllStringSubmatch(body, -1) {
		metas = append(metas, strings.TrimSpace(html.UnescapeString(tagPattern.ReplaceAllString(m[1], ""))))
	}
	return metas
}

func TestEachSlotsCardNamesItsChosenPlayer(t *testing.T) {
	f := newLineupFixture()
	f.tree = lineup.New(weekFS(2026, 9, map[string]string{"wood": woodLine(woodWeek9...)}))

	body := f.get("/lineup?team=wood").Body.String()

	want := []string{"Josh Allen", "Bijan Robinson", "Amon-Ra St. Brown", "Trey McBride", "Derrick Henry", "Kyle Juszczyk", "Brandon Aubrey", "Myles Garrett", "Sauce Gardner"}
	if got := cardNames(body); !slices.Equal(got, want) {
		t.Errorf("card names = %q, want %q", got, want)
	}
}

func TestACardsMetaLineStatesPositionTeamAndGame(t *testing.T) {
	f := newLineupFixture()
	f.tree = lineup.New(weekFS(2026, 9, map[string]string{"wood": woodLine(woodWeek9...)}))
	f.games.week = upcomingDET()

	body := f.get("/lineup?team=wood").Body.String()

	metas := cardMetas(body)
	if len(metas) != lineup.StarterCount {
		t.Fatalf("found %d card meta lines, want %d:\n%s", len(metas), lineup.StarterCount, body)
	}
	if got, want := metas[2], "WR · DET · Sun 12p"; got != want {
		t.Errorf("WR card meta = %q, want %q", got, want)
	}
	if got, want := metas[0], "QB · BUF · Bye"; got != want {
		t.Errorf("QB card meta = %q, want %q", got, want)
	}
}

func TestAnEmptySlotsCardIsEmpty(t *testing.T) {
	f := newLineupFixture()
	f.games.week = upcomingDET()

	body := f.get("/lineup?team=wood").Body.String()

	empty := make([]string, lineup.StarterCount)
	if got := cardNames(body); !slices.Equal(got, empty) {
		t.Errorf("card names = %q, want all empty", got)
	}
	if got := cardMetas(body); !slices.Equal(got, empty) {
		t.Errorf("card metas = %q, want all empty", got)
	}
}

// woodLine writes a lineup file of wood's roster players, in the order given.
func woodLine(ids ...string) string {
	records := map[string]string{}
	for _, line := range strings.Split(strings.TrimSuffix(woodRoster, "\n"), "\n")[1:] {
		records[strings.Split(line, ",")[0]] = line
	}
	body := lineupHeader
	for _, id := range ids {
		body += records[id] + "\n"
	}
	return body
}

var (
	woodWeek9 = []string{"101", "102", "105", "107", "103", "104", "108", "109", "110"}
	woodWeek7 = []string{"101", "103", "106", "107", "102", "105", "108", "110", "111"}
)

// selectedSlots is each slot select's selected value, in page order; "none"
// when no option is marked selected.
func selectedSlots(body string) []string {
	var values []string
	for _, m := range selectPattern.FindAllStringSubmatch(body, -1) {
		if m[1] == "team" {
			continue
		}
		value := "none"
		for _, o := range optionPattern.FindAllStringSubmatch(m[2], -1) {
			if o[2] != "" {
				value = html.UnescapeString(o[1])
			}
		}
		values = append(values, value)
	}
	return values
}

func TestTheSlotsStartOnThisWeeksLineup(t *testing.T) {
	f := newLineupFixture()
	f.tree = lineup.New(weekFS(2026, 9, map[string]string{"wood": woodLine(woodWeek9...)}))

	body := f.get("/lineup?team=wood").Body.String()

	if got := selectedSlots(body); !slices.Equal(got, woodWeek9) {
		t.Errorf("selected = %q, want %q", got, woodWeek9)
	}
}

func TestTheSlotsStartOnTheLatestLineupNotAfterThisWeek(t *testing.T) {
	f := newLineupFixture()
	f.tree = lineup.New(treeFS(
		weekFS(2026, 7, map[string]string{"wood": woodLine(woodWeek7...)}),
		weekFS(2026, 10, map[string]string{"wood": woodLine(woodWeek9...)}),
	))

	body := f.get("/lineup?team=wood").Body.String()

	if got := selectedSlots(body); !slices.Equal(got, woodWeek7) {
		t.Errorf("selected = %q, want %q", got, woodWeek7)
	}
}

func TestTheFormSaysThisWeeksLineupIsSaved(t *testing.T) {
	f := newLineupFixture()
	f.tree = lineup.New(weekFS(2026, 9, map[string]string{"wood": woodLine(woodWeek9...)}))

	body := f.get("/lineup?team=wood").Body.String()

	if !strings.Contains(body, "Saved for week 9") {
		t.Errorf("body does not say Saved for week 9:\n%s", body)
	}
}

func TestTheFormSaysThisWeekIsNotSavedAndWhichWeekTheSlotsStartFrom(t *testing.T) {
	f := newLineupFixture()
	f.tree = lineup.New(weekFS(2026, 7, map[string]string{"wood": woodLine(woodWeek7...)}))

	body := f.get("/lineup?team=wood").Body.String()

	if !strings.Contains(body, "Not saved for week 9 · starting from week 7") {
		t.Errorf("body does not say Not saved for week 9 · starting from week 7:\n%s", body)
	}
}

func TestWithNoLineupTheFormSaysOnlyNotSaved(t *testing.T) {
	body := newLineupFixture().get("/lineup?team=wood").Body.String()

	if !strings.Contains(body, "Not saved for week 9</p>") {
		t.Errorf("body does not say just Not saved for week 9:\n%s", body)
	}
}

var emptySlots = []string{"", "", "", "", "", "", "", "", ""}

func TestWithNoLineupEverySlotStartsEmpty(t *testing.T) {
	body := newLineupFixture().get("/lineup?team=wood").Body.String()

	if got := selectedSlots(body); !slices.Equal(got, emptySlots) {
		t.Errorf("selected = %q, want %q", got, emptySlots)
	}
}

func TestAStarterWhoLeftTheRosterIsOfferedAndSelected(t *testing.T) {
	f := newLineupFixture()
	starting := strings.Replace(woodLine(woodWeek9...), "107,Trey McBride,TE,ARI", "999,Travis Kelce,TE,KC", 1)
	f.tree = lineup.New(weekFS(2026, 9, map[string]string{"wood": starting}))

	body := f.get("/lineup?team=wood").Body.String()

	if got := selectedSlots(body)[3]; got != "999" {
		t.Errorf("TE selected = %q, want 999", got)
	}
	if got, want := optionText(t, body, "te", "999"), "Travis Kelce · TE · KC"; got != want {
		t.Errorf("option text = %q, want %q", got, want)
	}
}

func TestAMalformedStartingLineupLeavesTheSlotsEmptyAndSaysSo(t *testing.T) {
	f := newLineupFixture()
	f.tree = lineup.New(weekFS(2026, 9, map[string]string{"wood": woodLine("101", "102")}))

	rec := f.get("/lineup?team=wood")

	body := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if got := selectedSlots(body); !slices.Equal(got, emptySlots) {
		t.Errorf("selected = %q, want %q", got, emptySlots)
	}
	if !strings.Contains(body, "The starting lineup could not be read") {
		t.Errorf("body does not say the starting lineup could not be read:\n%s", body)
	}
}

// The notes sit under the slots so that fixing a slot keeps it in place.
func TestTheNotesFollowTheSlots(t *testing.T) {
	malformed := newLineupFixture()
	malformed.tree = lineup.New(weekFS(2026, 9, map[string]string{"wood": woodLine("101", "102")}))
	tests := map[string]struct {
		body string
		note string
	}{
		"the problems": {
			newLineupFixture().get(proposalURL("/lineup", woodWeek9With(4, "102")...)).Body.String(),
			`<ul class="problems">`,
		},
		"an unreadable starting lineup": {
			malformed.get("/lineup?team=wood").Body.String(),
			"The starting lineup could not be read",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			note, slots := strings.Index(tt.body, tt.note), strings.Index(tt.body, "</section>")
			if note < 0 || slots < 0 || note < slots {
				t.Errorf("%q is not after the slots:\n%s", tt.note, tt.body)
			}
		})
	}
}

// proposalURL proposes the IDs in slot order; an empty ID leaves its slot's
// parameter out.
func proposalURL(path string, ids ...string) string {
	q := "team=wood"
	for i, slot := range lineup.Slots() {
		if i < len(ids) && ids[i] != "" {
			q += "&" + strings.ToLower(string(slot)) + "=" + url.QueryEscape(ids[i])
		}
	}
	return path + "?" + q
}

func TestAProposalStartsEachSlotOnItsProposedPlayer(t *testing.T) {
	f := newLineupFixture()
	f.tree = lineup.New(weekFS(2026, 9, map[string]string{"wood": woodLine(woodWeek9...)}))

	body := f.get(proposalURL("/lineup", woodWeek7...)).Body.String()

	if got := selectedSlots(body); !slices.Equal(got, woodWeek7) {
		t.Errorf("selected = %q, want %q", got, woodWeek7)
	}
	if strings.Contains(body, `class="problems"`) {
		t.Errorf("a valid proposal shows problems:\n%s", body)
	}
}

func TestRepeatedSlotsFillInURLOrder(t *testing.T) {
	for _, flex := range [][2]string{{"104", "103"}, {"103", "104"}} {
		body := newLineupFixture().get("/lineup?team=wood&flex=" + flex[0] + "&flex=" + flex[1]).Body.String()

		if got := selectedSlots(body)[4:6]; !slices.Equal(got, flex[:]) {
			t.Errorf("flex selected = %q, want %q", got, flex)
		}
	}
}

var (
	problemsPattern = regexp.MustCompile(`(?s)<ul class="problems">(.*?)</ul>`)
	itemPattern     = regexp.MustCompile(`(?s)<li>(.*?)</li>`)
)

// problems is the page's listed problems, unescaped.
func problems(body string) []string {
	m := problemsPattern.FindStringSubmatch(body)
	if m == nil {
		return nil
	}
	var found []string
	for _, item := range itemPattern.FindAllStringSubmatch(m[1], -1) {
		found = append(found, html.UnescapeString(item[1]))
	}
	return found
}

// woodWeek9With is the week 9 lineup with one slot's ID replaced.
func woodWeek9With(slot int, id string) []string {
	ids := slices.Clone(woodWeek9)
	ids[slot] = id
	return ids
}

func TestAnUnknownProposedIDIsNamedAndItsSlotIsEmpty(t *testing.T) {
	rec := newLineupFixture().get(proposalURL("/lineup", woodWeek9With(0, "99999")...))

	body := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if got, want := problems(body), []string{"QB: no player 99999 on the roster or starting lineup"}; !slices.Equal(got, want) {
		t.Errorf("problems = %q, want %q", got, want)
	}
	if got := selectedSlots(body)[0]; got != "" {
		t.Errorf("QB selected = %q, want empty", got)
	}
}

func TestAnIneligibleProposedPlayerIsNamedAndStaysSelected(t *testing.T) {
	// 106 rather than 105, which week 9 already starts at WR.
	body := newLineupFixture().get(proposalURL("/lineup", woodWeek9With(0, "106")...)).Body.String()

	if got, want := problems(body), []string{"Free Agent Wideout (WR) cannot play QB"}; !slices.Equal(got, want) {
		t.Errorf("problems = %q, want %q", got, want)
	}
	if got := selectedSlots(body)[0]; got != "106" {
		t.Errorf("QB selected = %q, want 106", got)
	}
}

func TestAPlayerProposedInTwoSlotsIsNamed(t *testing.T) {
	body := newLineupFixture().get(proposalURL("/lineup", woodWeek9With(4, "102")...)).Body.String()

	if got, want := problems(body), []string{"Bijan Robinson is in two slots"}; !slices.Equal(got, want) {
		t.Errorf("problems = %q, want %q", got, want)
	}
	if got := selectedSlots(body); got[1] != "102" || got[4] != "102" {
		t.Errorf("RB and FLEX selected = %q and %q, want 102 in both", got[1], got[4])
	}
}

func TestAProposalWithNoKNamesTheEmptySlot(t *testing.T) {
	body := newLineupFixture().get(proposalURL("/lineup", woodWeek9With(6, "")...)).Body.String()

	if got, want := problems(body), []string{"K slot is empty"}; !slices.Equal(got, want) {
		t.Errorf("problems = %q, want %q", got, want)
	}
}

func TestAProposalWithThreeFLEXValuesSaysSo(t *testing.T) {
	body := newLineupFixture().get(proposalURL("/lineup", woodWeek9...) + "&flex=106").Body.String()

	if got, want := problems(body), []string{"too many FLEX players: 3 for 2 slots"}; !slices.Equal(got, want) {
		t.Errorf("problems = %q, want %q", got, want)
	}
}

func TestAProposedIDWithMarkupIsEscapedAndUnknown(t *testing.T) {
	rec := newLineupFixture().get(proposalURL("/lineup", woodWeek9With(0, "<script>")...))

	body := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	clean := newLineupFixture().get(proposalURL("/lineup", woodWeek9...)).Body.String()
	if strings.Count(body, "<script>") != strings.Count(clean, "<script>") {
		t.Errorf("body holds an unescaped <script> beyond the page's own:\n%s", body)
	}
	if got, want := problems(body), []string{"QB: no player <script> on the roster or starting lineup"}; !slices.Equal(got, want) {
		t.Errorf("problems = %q, want %q", got, want)
	}
}

var getFormPattern = regexp.MustCompile(`(?s)<form method="get" action="/lineup">(.*?)</form>`)

func TestTheSlotsAreAGETFormCarryingTheTeam(t *testing.T) {
	body := newLineupFixture().get("/lineup?team=wood").Body.String()

	var form string
	for _, m := range getFormPattern.FindAllStringSubmatch(body, -1) {
		if strings.Contains(m[1], `<select name="qb"`) {
			form = m[1]
		}
	}
	if form == "" {
		t.Fatalf("no GET form to /lineup holds the slots:\n%s", body)
	}
	if !strings.Contains(form, `<input type="hidden" name="team" value="wood">`) {
		t.Errorf("the slots form does not carry team=wood:\n%s", form)
	}
	if !strings.Contains(form, `<button>Copy</button>`) {
		t.Errorf("the slots form has no Copy button:\n%s", form)
	}
}

// slotsForm is the inside of the form holding the slot selects.
func slotsForm(t *testing.T, body string) string {
	t.Helper()
	for _, m := range slotsFormPattern.FindAllStringSubmatch(body, -1) {
		if strings.Contains(m[1], `<select name="qb"`) {
			return m[1]
		}
	}
	t.Fatalf("no form holds the slots:\n%s", body)
	return ""
}

var slotsFormPattern = regexp.MustCompile(`(?s)<form[^>]*>(.*?)</form>`)

func TestWithASaverTheSlotsFormPostsSaveForItsWeek(t *testing.T) {
	f := newLineupFixture()
	f.saver = &fakeSaver{}

	form := slotsForm(t, f.get("/lineup?team=wood").Body.String())

	for _, want := range []string{
		`<button class="primary" formmethod="post" formaction="/lineup" name="save">Save</button>`,
		`<input type="hidden" name="season" value="2026">`,
		`<input type="hidden" name="week" value="9">`,
	} {
		if !strings.Contains(form, want) {
			t.Errorf("the slots form lacks %s:\n%s", want, form)
		}
	}
}

var (
	actionsPattern = regexp.MustCompile(`(?s)<div class="actions">(.*?)</div>`)
	actionPattern  = regexp.MustCompile(`(?s)<(?:button|a)([^>]*)>(.*?)</(?:button|a)>`)
	ariaLabel      = regexp.MustCompile(`aria-label="([^"]*)"`)
)

// actionNames is the action row's controls by accessible name, left to right,
// each with its attributes.
func actionNames(t *testing.T, body string) (names, attrs []string) {
	t.Helper()
	row := actionsPattern.FindStringSubmatch(body)
	if row == nil {
		t.Fatalf("no action row:\n%s", body)
	}
	for _, m := range actionPattern.FindAllStringSubmatch(row[1], -1) {
		name := m[2]
		if l := ariaLabel.FindStringSubmatch(m[1]); l != nil {
			name = l[1]
		}
		names = append(names, name)
		attrs = append(attrs, m[1])
	}
	return names, attrs
}

func TestSaveLeadsTheActionRowAsThePrimaryButton(t *testing.T) {
	f := newLineupFixture()
	f.saver = &fakeSaver{}

	names, attrs := actionNames(t, f.get("/lineup?team=wood").Body.String())

	if want := []string{"Save", "Copy", "Download"}; !slices.Equal(names, want) {
		t.Fatalf("action row = %q, want %q", names, want)
	}
	if !strings.Contains(attrs[0], `class="primary"`) {
		t.Errorf("Save's attributes %q lack class=\"primary\"", attrs[0])
	}
}

func TestASavedWeeksSaveIsPlain(t *testing.T) {
	f := newLineupFixture()
	f.saver = &fakeSaver{}
	f.tree = lineup.New(weekFS(2026, 9, map[string]string{"wood": woodLine(woodWeek9...)}))

	names, attrs := actionNames(t, f.get("/lineup?team=wood").Body.String())

	if len(names) == 0 || names[0] != "Save" {
		t.Fatalf("action row = %q, want Save first", names)
	}
	if strings.Contains(attrs[0], `class="primary"`) {
		t.Errorf("a saved week's Save is highlighted: %q", attrs[0])
	}
}

func TestWithNoSaverThereIsNoSave(t *testing.T) {
	body := newLineupFixture().get("/lineup?team=wood").Body.String()

	if strings.Contains(body, ">Save</button>") {
		t.Errorf("the page offers Save with no saver:\n%s", body)
	}
}

// The form itself shows the lineup, so no separate summary repeats it.
func TestAValidLineupIsNotRepeatedAsText(t *testing.T) {
	body := newLineupFixture().get(proposalURL("/lineup", woodWeek9...)).Body.String()

	if strings.Contains(body, "<textarea") {
		t.Errorf("a valid lineup is repeated in a textarea:\n%s", body)
	}
}

var downloadButtonPattern = regexp.MustCompile(`(?s)<button formaction="/lineup\.csv" aria-label="Download">(.*?)</button>`)

// Download is offered whatever the slots hold; pressing it is the check.
func TestDownloadIsAnEnabledGlyphButtonForAnySlots(t *testing.T) {
	tests := map[string]string{
		"a duplicate proposal":         proposalURL("/lineup", woodWeek9With(4, "102")...),
		"a starting lineup with holes": "/lineup?team=wood",
		"a valid proposal":             proposalURL("/lineup", woodWeek9...),
	}
	for name, u := range tests {
		t.Run(name, func(t *testing.T) {
			form := slotsForm(t, newLineupFixture().get(u).Body.String())

			m := downloadButtonPattern.FindStringSubmatch(form)
			if m == nil {
				t.Fatalf("the slots form has no Download button to /lineup.csv:\n%s", form)
			}
			if !strings.HasPrefix(strings.TrimSpace(m[1]), "<svg") {
				t.Errorf("Download holds %q, want an inline svg", m[1])
			}
			if strings.Contains(form, "aria-disabled") {
				t.Errorf("Download is disabled:\n%s", form)
			}
		})
	}
}

func (f *lineupFixture) getCSV(url string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	LineupCSVHandler(f.rosters, f.tree, f.current).ServeHTTP(rec, httptest.NewRequest("GET", url, nil))
	return rec
}

// woodLineup is wood's roster players with the IDs given, built by slot.
func woodLineup(t *testing.T, ids []string) lineup.Lineup {
	t.Helper()
	roster, err := testRosters().Read("wood")
	if err != nil {
		t.Fatal(err)
	}
	var players [lineup.StarterCount]lineup.Player
	for i, id := range ids {
		players[i] = roster[slices.IndexFunc(roster, func(p lineup.Player) bool { return p.ID == id })]
	}
	l, err := lineup.FromSlots(players)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func TestAValidProposalDownloadsAsALineupFile(t *testing.T) {
	rec := newLineupFixture().getCSV(proposalURL("/lineup.csv", woodWeek9...))

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if got, want := rec.Header().Get("Content-Type"), "text/csv; charset=utf-8"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
	if got, want := rec.Header().Get("Content-Disposition"), `attachment; filename="wood.csv"`; got != want {
		t.Errorf("Content-Disposition = %q, want %q", got, want)
	}
	var want strings.Builder
	if err := woodLineup(t, woodWeek9).WriteCSV(&want); err != nil {
		t.Fatal(err)
	}
	if got := rec.Body.String(); got != want.String() {
		t.Errorf("body = %q, want %q", got, want.String())
	}
}

func TestADownloadOrTextThatCannotBeALineupIsRefused(t *testing.T) {
	tests := []struct {
		name  string
		url   string
		setup func(*lineupFixture)
		want  int
	}{
		{name: "a duplicate player", url: proposalURL("/lineup.csv", woodWeek9With(4, "102")...), want: http.StatusBadRequest},
		{name: "no slots at all", url: "/lineup.csv?team=wood", want: http.StatusBadRequest},
		{name: "no week to set", url: proposalURL("/lineup.csv", woodWeek9...), setup: func(f *lineupFixture) {
			f.current.current = nflweek.New(2026, 0, nflweek.Offseason)
		}, want: http.StatusNotFound},
		{name: "an unknown team", url: "/lineup.csv?team=nobody", want: http.StatusBadRequest},
		{name: "a failed current-week read", url: proposalURL("/lineup.csv", woodWeek9...), setup: func(f *lineupFixture) {
			f.current.err = errors.New("sleeper is down")
		}, want: http.StatusBadGateway},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newLineupFixture()
			if tt.setup != nil {
				tt.setup(f)
			}

			if rec := f.getCSV(tt.url); rec.Code != tt.want {
				t.Errorf("download status = %d, want %d", rec.Code, tt.want)
			}
			textURL := strings.Replace(tt.url, "/lineup.csv", "/lineup.txt", 1)
			if rec := f.getText(textURL); rec.Code != tt.want {
				t.Errorf("text status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}

// Without script, Download's plain GET is the check, so the 400 must say why.
func TestADuplicateDownloadIsRefusedNamingThePlayer(t *testing.T) {
	rec := newLineupFixture().getCSV(proposalURL("/lineup.csv", woodWeek9With(4, "102")...))

	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "Bijan Robinson is in two slots") {
		t.Errorf("download = %d %q, want 400 naming Bijan Robinson in two slots", rec.Code, rec.Body.String())
	}
}

func (f *lineupFixture) getText(url string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	LineupTextHandler(f.rosters, f.tree, f.current).ServeHTTP(rec, httptest.NewRequest("GET", url, nil))
	return rec
}

func TestAValidProposalIsAnsweredAsPasteableText(t *testing.T) {
	rec := newLineupFixture().getText(proposalURL("/lineup.txt", woodWeek9...))

	want := "wood · 2026 week 9\n" +
		"QB Josh Allen (BUF)\n" +
		"RB Bijan Robinson (ATL)\n" +
		"WR Amon-Ra St. Brown (DET)\n" +
		"TE Trey McBride (ARI)\n" +
		"FLEX Derrick Henry (BAL)\n" +
		"FLEX Kyle Juszczyk (SF)\n" +
		"K Brandon Aubrey (DAL)\n" +
		"D Myles Garrett (CLE)\n" +
		"D Sauce Gardner (NYJ)\n"
	if rec.Code != http.StatusOK || rec.Body.String() != want {
		t.Errorf("got %d %q, want 200 %q", rec.Code, rec.Body.String(), want)
	}
}

func TestARosterNameWithMarkupIsEscaped(t *testing.T) {
	f := newLineupFixture()
	f.rosters = lineup.NewRosters(fstest.MapFS{
		"wood.csv": {Data: []byte(strings.Replace(woodRoster, "Josh Allen", "<b>X</b>", 1))},
	})

	body := f.get(proposalURL("/lineup", woodWeek9...)).Body.String()

	if strings.Contains(body, "<b>") {
		t.Errorf("body holds a <b> element:\n%s", body)
	}
	if got := strings.Count(body, "&lt;b&gt;X&lt;/b&gt;"); got != 2 {
		t.Errorf("escaped name appears %d times, want 2 (the QB option and card):\n%s", got, body)
	}
}

type savedLineup struct {
	season, week int
	team         string
	lineup       lineup.Lineup
}

// fakeSaver records each save and answers err.
type fakeSaver struct {
	saves []savedLineup
	err   error
}

func (s *fakeSaver) Save(season, week int, team string, l lineup.Lineup) error {
	s.saves = append(s.saves, savedLineup{season, week, team, l})
	return s.err
}

// postSave posts the form's fields for team in the given week, proposing the
// IDs in slot order.
func (f *lineupFixture) postSave(saver LineupSaver, team string, season, week int, ids ...string) *httptest.ResponseRecorder {
	form := url.Values{"team": {team}, "season": {strconv.Itoa(season)}, "week": {strconv.Itoa(week)}}
	for i, slot := range lineup.Slots() {
		if i < len(ids) && ids[i] != "" {
			form.Add(strings.ToLower(string(slot)), ids[i])
		}
	}
	req := httptest.NewRequest("POST", "/lineup", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	LineupSaveHandler(f.rosters, f.tree, f.current, f.games, saver).ServeHTTP(rec, req)
	return rec
}

// newBojjaesSaveFixture gives the Bojjaes wood's roster, so they have a full
// lineup to save.
func newBojjaesSaveFixture() *lineupFixture {
	f := newLineupFixture()
	f.rosters = lineup.NewRosters(fstest.MapFS{
		"bojjaes.csv": {Data: []byte(woodRoster)},
		"wood.csv":    {Data: []byte(woodRoster)},
	})
	return f
}

func TestAValidLineupIsSavedAndRedirectsToTheForm(t *testing.T) {
	saver := &fakeSaver{}

	rec := newBojjaesSaveFixture().postSave(saver, "bojjaes", 2026, 9, woodWeek9...)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want 303", rec.Code)
	}
	if got, want := rec.Header().Get("Location"), "/lineup?team=bojjaes"; got != want {
		t.Errorf("Location = %q, want %q", got, want)
	}
	if len(saver.saves) != 1 {
		t.Fatalf("saves = %d, want 1", len(saver.saves))
	}
	got := saver.saves[0]
	if got.season != 2026 || got.week != 9 || got.team != "bojjaes" {
		t.Errorf("saved %s for %d week %d, want bojjaes for 2026 week 9", got.team, got.season, got.week)
	}
	if want := woodLineup(t, woodWeek9).Starters(); !slices.Equal(got.lineup.Starters(), want) {
		t.Errorf("saved starters = %v, want %v", got.lineup.Starters(), want)
	}
}

func TestAnInvalidLineupIsNotSavedAndTheFormKeepsItsPicks(t *testing.T) {
	saver := &fakeSaver{}
	ids := woodWeek9With(4, "102")

	rec := newBojjaesSaveFixture().postSave(saver, "bojjaes", 2026, 9, ids...)

	body := rec.Body.String()
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
	if got, want := problems(body), []string{"Bijan Robinson is in two slots"}; !slices.Equal(got, want) {
		t.Errorf("problems = %q, want %q", got, want)
	}
	if got := selectedSlots(body); !slices.Equal(got, ids) {
		t.Errorf("selected = %q, want %q", got, ids)
	}
	if len(saver.saves) != 0 {
		t.Errorf("saves = %d, want 0", len(saver.saves))
	}
}

func TestAnUnknownTeamIsRefusedBeforeTheCurrentWeekIsRead(t *testing.T) {
	f := newBojjaesSaveFixture()
	saver := &fakeSaver{}

	rec := f.postSave(saver, "../wood", 2026, 9, woodWeek9...)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
	if f.current.calls != 0 {
		t.Errorf("current week read %d times, want 0", f.current.calls)
	}
	if len(saver.saves) != 0 {
		t.Errorf("saves = %d, want 0", len(saver.saves))
	}
}

func TestALineupPostedForAnEarlierWeekIsRefused(t *testing.T) {
	f := newBojjaesSaveFixture()
	f.current.current = nflweek.New(2026, 13, nflweek.Regular)
	saver := &fakeSaver{}

	rec := f.postSave(saver, "bojjaes", 2026, 12, woodWeek9...)

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "week changed") {
		t.Errorf("body does not say the week changed:\n%s", rec.Body.String())
	}
	if len(saver.saves) != 0 {
		t.Errorf("saves = %d, want 0", len(saver.saves))
	}
}

func TestASaveWithNoWeekToSetOrNoCurrentWeekIsRefused(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*lineupFixture)
		want  int
	}{
		{name: "NFL week 18", setup: func(f *lineupFixture) {
			f.current.current = nflweek.New(2026, 18, nflweek.Regular)
		}, want: http.StatusNotFound},
		{name: "a failed current-week read", setup: func(f *lineupFixture) {
			f.current.err = errors.New("sleeper is down")
		}, want: http.StatusBadGateway},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newBojjaesSaveFixture()
			tt.setup(f)
			saver := &fakeSaver{}

			rec := f.postSave(saver, "bojjaes", 2026, 18, woodWeek9...)

			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d", rec.Code, tt.want)
			}
			if len(saver.saves) != 0 {
				t.Errorf("saves = %d, want 0", len(saver.saves))
			}
		})
	}
}

func TestASaveRefusedForAnotherOpponentIsAConflict(t *testing.T) {
	saver := &fakeSaver{err: &volume.OpponentSetError{Existing: "wood"}}

	rec := newBojjaesSaveFixture().postSave(saver, "bojjaes", 2026, 9, woodWeek9...)

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "bojjaes&#39; opponent is already wood") {
		t.Errorf("body does not name the opponent:\n%s", rec.Body.String())
	}
}

func TestAFailedSaveIsAServerErrorAndLogged(t *testing.T) {
	var logged strings.Builder
	log.SetOutput(&logged)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	saver := &fakeSaver{err: errors.New("disk full")}

	rec := newBojjaesSaveFixture().postSave(saver, "bojjaes", 2026, 9, woodWeek9...)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if !strings.Contains(logged.String(), "disk full") {
		t.Errorf("log = %q, want it to mention disk full", logged.String())
	}
}
