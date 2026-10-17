package web

import (
	"context"
	"errors"
	"html"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/eshifrin/bojjaes/internal/games"
	"github.com/eshifrin/bojjaes/internal/lineup"
)

// fakeGames stands in for the games source: a week built here, or an error,
// and a count of how often the page asked.
type fakeGames struct {
	week  games.Week
	err   error
	calls int
}

func (f *fakeGames) AsOf(context.Context, int, int) (games.Week, time.Time, error) {
	f.calls++
	if f.err != nil {
		return games.Week{}, time.Time{}, f.err
	}
	return f.week, time.Time{}, nil
}

// gamesWeek is a 2025 week 15 holding the given games.
func gamesWeek(gs ...games.Game) games.Week {
	return games.NewWeek(2025, 15, gs)
}

// chaseTree is the fixture matchup with Ja'Marr Chase (CIN) in place of our
// first starter, so a CIN game has a line to show on. Replacements are
// old/new pairs applied to the opponent's lineup file.
func chaseTree(replacements ...string) *lineup.Tree {
	ours := strings.Replace(lineupCSV(ourLine), "1,Puka Nacua,WR,LAR", "1,Ja'Marr Chase,WR,CIN", 1)
	theirs := lineupCSV(theirLine)
	for i := 0; i+1 < len(replacements); i += 2 {
		theirs = strings.Replace(theirs, replacements[i], replacements[i+1], 1)
	}
	return lineup.New(weekFS(2025, 15, map[string]string{"bojjaes": ours, "wood": theirs}))
}

var (
	fullMetaPattern = regexp.MustCompile(`(?s)class="meta">(.*)</span>$`)
	gamePattern     = regexp.MustCompile(`(?s)<span class="game">(.*?)</span>`)
)

// starterMeta returns the raw markup inside the named starter's meta span.
// The meta span is the row's last element, so it runs to the row's last
// </span>, game span included.
func starterMeta(t *testing.T, body, name string) string {
	t.Helper()
	for _, row := range rowPattern.FindAllStringSubmatch(body, -1) {
		if !strings.Contains(html.UnescapeString(row[1]), ">"+name+"<") {
			continue
		}
		m := fullMetaPattern.FindStringSubmatch(row[1])
		if m == nil {
			t.Fatalf("%s's row has no meta span: %s", name, row[1])
		}
		return m[1]
	}
	t.Fatalf("no row for %s in:\n%s", name, body)
	return ""
}

// metaText is a meta span's markup as a reader sees it.
func metaText(meta string) string {
	return html.UnescapeString(tagPattern.ReplaceAllString(meta, ""))
}

func serveGames(tree *lineup.Tree, gs *fakeGames) string {
	return serve(Handler(tree, &fakeSource{weekStats: fixtureStats()}, gs), http.MethodGet, "/2025/15").Body.String()
}

func TestACompleteGameReadsFinalAfterTheTeam(t *testing.T) {
	gs := &fakeGames{week: gamesWeek(games.Game{Home: "CIN", Away: "PIT", State: games.Complete, Period: "F", Status: "complete"})}

	meta := starterMeta(t, serveGames(chaseTree(), gs), "Ja'Marr Chase")

	if got, want := metaText(meta), "WR · CIN · Final"; got != want {
		t.Errorf("meta line = %q, want %q", got, want)
	}
	if m := gamePattern.FindStringSubmatch(meta); m == nil || m[1] != "Final" {
		t.Errorf("meta markup = %q, want Final inside span.game", meta)
	}
}

func TestAFailedGamesFetchServesThePageWithoutGameStates(t *testing.T) {
	gs := &fakeGames{err: errors.New("games endpoint is down")}

	rec := serve(Handler(chaseTree(), &fakeSource{weekStats: fixtureStats()}, gs), http.MethodGet, "/2025/15")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	without := serveGames(chaseTree(), &fakeGames{week: gamesWeek()})
	if got, want := renderedStarters(body), renderedStarters(without); !slices.Equal(got, want) {
		t.Errorf("points = %q, want those of a page with no games, %q", got, want)
	}
	if got, want := renderedTotals(body), renderedTotals(without); !slices.Equal(got, want) {
		t.Errorf("totals = %q, want %q", got, want)
	}
	meta := starterMeta(t, body, "Ja'Marr Chase")
	if got, want := metaText(meta), "WR · CIN"; got != want {
		t.Errorf("meta line = %q, want %q", got, want)
	}
	if strings.Contains(body, `class="game"`) {
		t.Errorf("a failed games fetch still rendered a game state:\n%s", meta)
	}
}

func TestAWeekWithNoGamesShowsNoByes(t *testing.T) {
	body := serveGames(chaseTree(), &fakeGames{week: gamesWeek()})

	if got, want := metaText(starterMeta(t, body, "Ja'Marr Chase")), "WR · CIN"; got != want {
		t.Errorf("meta line = %q, want %q", got, want)
	}
	if strings.Contains(body, `class="game"`) || strings.Contains(body, "Bye") {
		t.Error("a week with no games rendered a game state")
	}
}

// upcomingCIN is a week whose CIN game has not kicked off.
func upcomingCIN(kickoff time.Time) *fakeGames {
	return &fakeGames{week: gamesWeek(games.Game{Home: "CIN", Away: "PIT", State: games.Upcoming, Kickoff: kickoff, Status: "pre_game"})}
}

func TestAnUpcomingGameStampsItsKickoffAsRFC3339(t *testing.T) {
	kickoff := time.Date(2026, 10, 11, 17, 0, 0, 0, time.UTC)

	meta := starterMeta(t, serveGames(chaseTree(), upcomingCIN(kickoff)), "Ja'Marr Chase")

	times := renderedTimes(meta)
	if len(times) != 1 || times[0][0] != "2026-10-11T17:00:00Z" {
		t.Errorf("meta markup = %q, want one <time> with datetime %q", meta, "2026-10-11T17:00:00Z")
	}
}

func TestAKickoffIsServedInChicagoTime(t *testing.T) {
	tests := []struct {
		name    string
		kickoff time.Time
		want    string
	}{
		{name: "a Sunday noon kickoff", kickoff: time.Date(2026, 10, 11, 17, 0, 0, 0, time.UTC), want: "Sun 12:00 PM"},
		{name: "the weekday is Chicago's, not UTC's", kickoff: time.Date(2026, 10, 13, 0, 15, 0, 0, time.UTC), want: "Mon 7:15 PM"},
		{name: "a December kickoff is on standard time", kickoff: time.Date(2026, 12, 13, 18, 0, 0, 0, time.UTC), want: "Sun 12:00 PM"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := starterMeta(t, serveGames(chaseTree(), upcomingCIN(tt.kickoff)), "Ja'Marr Chase")

			times := renderedTimes(meta)
			if len(times) != 1 || times[0][1] != tt.want {
				t.Errorf("meta markup = %q, want one <time> reading %q", meta, tt.want)
			}
		})
	}
}

// advancingGames answers each call with the CIN game one quarter further on,
// so two readings in one request would show different states.
type advancingGames struct{ calls int }

func (a *advancingGames) AsOf(context.Context, int, int) (games.Week, time.Time, error) {
	a.calls++
	quarter := strconv.Itoa(a.calls)
	return gamesWeek(games.Game{Home: "CIN", Away: "PIT", State: games.InProgress, Period: quarter, Status: "in_game"}), time.Time{}, nil
}

func TestBothColumnsStateTheirGamesFromOneReading(t *testing.T) {
	gs := &advancingGames{}
	tree := chaseTree("11,Josh Allen,QB,BUF", "11,Joe Burrow,QB,CIN")

	body := serve(Handler(tree, &fakeSource{weekStats: fixtureStats()}, gs), http.MethodGet, "/2025/15").Body.String()

	if gs.calls != 1 {
		t.Errorf("games source called %d times, want 1", gs.calls)
	}
	ours := metaText(starterMeta(t, body, "Ja'Marr Chase"))
	theirs := metaText(starterMeta(t, body, "Joe Burrow"))
	if ours != "WR · CIN · Q1" || theirs != "QB · CIN · Q1" {
		t.Errorf("CIN starters read %q and %q, want both on Q1", ours, theirs)
	}
}

func TestAStarterWithoutStatsStillShowsTheirGame(t *testing.T) {
	kickoff := time.Date(2026, 10, 11, 17, 0, 0, 0, time.UTC)

	body := serve(Handler(chaseTree(), &fakeSource{weekStats: fixtureStats("1")}, upcomingCIN(kickoff)), http.MethodGet, "/2025/15").Body.String()

	if !slices.Contains(renderedStarters(body), "Ja&#39;Marr Chase=--") {
		t.Errorf("starters = %q, want Ja'Marr Chase at --", renderedStarters(body))
	}
	if got, want := metaText(starterMeta(t, body, "Ja'Marr Chase")), "WR · CIN · Sun 12:00 PM"; got != want {
		t.Errorf("meta line = %q, want %q", got, want)
	}
}

func TestATeamOutsideAWeekWithGamesIsOnBye(t *testing.T) {
	tree := chaseTree("11,Josh Allen,QB,BUF", "11,Patrick Mahomes,QB,KC")

	body := serveGames(tree, upcomingCIN(time.Date(2026, 10, 11, 17, 0, 0, 0, time.UTC)))

	if got, want := metaText(starterMeta(t, body, "Patrick Mahomes")), "QB · KC · Bye"; got != want {
		t.Errorf("meta line = %q, want %q", got, want)
	}
}

func TestAFreeAgentShowsNothingAfterTheTeam(t *testing.T) {
	tree := chaseTree("19,Ladd McConkey,WR,LAC", "19,Ladd McConkey,WR,FA")

	body := serveGames(tree, upcomingCIN(time.Date(2026, 10, 11, 17, 0, 0, 0, time.UTC)))

	if got := starterMeta(t, body, "Ladd McConkey"); got != "WR · FA" {
		t.Errorf("meta markup = %q, want exactly %q", got, "WR · FA")
	}
}

func TestAFailedStatsFetchFailsThePageEvenWithGames(t *testing.T) {
	gs := upcomingCIN(time.Date(2026, 10, 11, 17, 0, 0, 0, time.UTC))

	rec := serve(Handler(chaseTree(), &fakeSource{err: errors.New("stats are down")}, gs), http.MethodGet, "/2025/15")

	if rec.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadGateway)
	}
	if strings.Contains(rec.Body.String(), "<ol>") {
		t.Error("a failed stats fetch rendered a scoreboard")
	}
}

// fixtureGames puts the fixture's LAR starters in one game, so the fixture
// page has a game state to show beside the as-of line.
func fixtureGames(state games.State) *fakeGames {
	return &fakeGames{week: gamesWeek(games.Game{Home: "LAR", Away: "SF", State: state, Kickoff: time.Date(2025, 12, 14, 21, 5, 0, 0, time.UTC)})}
}
