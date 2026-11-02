package web

import (
	"context"
	"errors"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/eshifrin/bojjaes/internal/games"
	"github.com/eshifrin/bojjaes/internal/lineup"
	"github.com/eshifrin/bojjaes/internal/nflweek"
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
	LineupHandler(f.rosters, f.tree, f.current, f.games).ServeHTTP(rec, httptest.NewRequest("GET", url, nil))
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
	if strings.Contains(body, "<script>") {
		t.Errorf("body holds an unescaped <script>:\n%s", body)
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
	if !strings.Contains(form, `<button>Check</button>`) {
		t.Errorf("the slots form has no Check button:\n%s", form)
	}
}

var textareaPattern = regexp.MustCompile(`(?s)<textarea[^>]*>(.*?)</textarea>`)

// summary is the page's textarea text, unescaped, and whether there is one.
func summary(body string) (string, bool) {
	m := textareaPattern.FindStringSubmatch(body)
	if m == nil {
		return "", false
	}
	return html.UnescapeString(m[1]), true
}

func TestAValidProposalIsSummarisedAsPasteableText(t *testing.T) {
	body := newLineupFixture().get(proposalURL("/lineup", woodWeek9...)).Body.String()

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
	if got, ok := summary(body); got != want {
		t.Errorf("summary (shown %v) = %q, want %q", ok, got, want)
	}
}

func TestAnIncompleteLineupHasNoSummaryOrDownload(t *testing.T) {
	tests := map[string]string{
		"a duplicate proposal":         proposalURL("/lineup", woodWeek9With(4, "102")...),
		"a starting lineup with holes": "/lineup?team=wood",
	}
	for name, u := range tests {
		t.Run(name, func(t *testing.T) {
			body := newLineupFixture().get(u).Body.String()

			if _, ok := summary(body); ok {
				t.Errorf("a summary is shown:\n%s", body)
			}
			if strings.Contains(body, "/lineup.csv") {
				t.Errorf("a download is linked:\n%s", body)
			}
		})
	}
}

var csvLinkPattern = regexp.MustCompile(`href="(/lineup\.csv[^"]*)"`)

func TestAValidLineupLinksItsDownloadAndNamesWhereItIsCommitted(t *testing.T) {
	body := newLineupFixture().get(proposalURL("/lineup", woodWeek9...)).Body.String()

	m := csvLinkPattern.FindStringSubmatch(body)
	want := "/lineup.csv?team=wood&qb=101&rb=102&wr=105&te=107&flex=103&flex=104&k=108&d=109&d=110"
	if m == nil || html.UnescapeString(m[1]) != want {
		t.Errorf("download link = %v, want %q", m, want)
	}
	if !strings.Contains(body, "internal/data/lineups/2026/9/wood.csv") {
		t.Errorf("body does not name the committed path:\n%s", body)
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

func TestADownloadThatCannotBeALineupFileIsRefused(t *testing.T) {
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
				t.Errorf("status = %d, want %d", rec.Code, tt.want)
			}
		})
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
		t.Errorf("escaped name appears %d times, want 2 (QB option and summary):\n%s", got, body)
	}
}
