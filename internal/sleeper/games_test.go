package sleeper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/eshifrin/bojjaes/internal/games"
)

func TestWeekGamesRequestsTheWeeksScores(t *testing.T) {
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Write([]byte(`[]`))
	}))
	t.Cleanup(srv.Close)

	if _, err := (Client{BaseURL: srv.URL}).WeekGames(context.Background(), 2026, 5); err != nil {
		t.Fatalf("WeekGames: %v", err)
	}

	if want := "/scores/nfl/regular/2026/5"; path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
}

func weekGamesFrom(t *testing.T, body string) games.Week {
	t.Helper()
	srv := jsonServer(t, body)
	week, err := (Client{BaseURL: srv.URL}).WeekGames(context.Background(), 2026, 5)
	if err != nil {
		t.Fatalf("WeekGames: %v", err)
	}
	return week
}

func gameFor(t *testing.T, week games.Week, team string) games.Game {
	t.Helper()
	g, ok := week.ForTeam(team)
	if !ok {
		t.Fatalf("ForTeam(%q) found no game", team)
	}
	return g
}

func TestWeekGamesReadsTheTeams(t *testing.T) {
	week := weekGamesFrom(t, `[{"status":"complete","metadata":{"home_team":"ARI","away_team":"DET"}}]`)

	g := gameFor(t, week, "ARI")

	if g.Home != "ARI" || g.Away != "DET" {
		t.Errorf("game = %s at %s, want DET at ARI", g.Away, g.Home)
	}
}

func TestWeekGamesMapsStatusToState(t *testing.T) {
	tests := []struct {
		status string
		want   games.State
	}{
		{"pre_game", games.Upcoming},
		{"in_game", games.InProgress},
		{"complete", games.Complete},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			week := weekGamesFrom(t, `[{"status":"`+tt.status+`","metadata":{"home_team":"ARI","away_team":"DET"}}]`)

			if got := gameFor(t, week, "ARI").State; got != tt.want {
				t.Errorf("State = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWeekGamesKeepsAnUnknownStatusAsSent(t *testing.T) {
	week := weekGamesFrom(t, `[{"status":"postponed","metadata":{"home_team":"ARI","away_team":"DET"}}]`)

	g := gameFor(t, week, "ARI")

	if g.State != games.Other || g.Status != "postponed" {
		t.Errorf("State, Status = %v, %q; want Other, %q", g.State, g.Status, "postponed")
	}
}

func TestWeekGamesReadsKickoffFromEpochMilliseconds(t *testing.T) {
	week := weekGamesFrom(t, `[{"status":"pre_game","start_time":1791750300000,"metadata":{"home_team":"ARI","away_team":"DET"}}]`)

	want := time.Date(2026, 10, 11, 20, 25, 0, 0, time.UTC)
	if got := gameFor(t, week, "ARI").Kickoff; !got.Equal(want) {
		t.Errorf("Kickoff = %v, want %v", got, want)
	}
}

func TestWeekGamesCarriesQuarterTextAsPeriod(t *testing.T) {
	week := weekGamesFrom(t, `[{"status":"in_game","metadata":{"home_team":"ARI","away_team":"DET","quarter":"HALF"}}]`)

	if got := gameFor(t, week, "ARI").Period; got != "HALF" {
		t.Errorf("Period = %q, want HALF", got)
	}
}

func TestWeekGamesReadsANullQuarterAsEmpty(t *testing.T) {
	week := weekGamesFrom(t, `[{"status":"pre_game","metadata":{"home_team":"ARI","away_team":"DET","quarter":null}}]`)

	if got := gameFor(t, week, "ARI").Period; got != "" {
		t.Errorf("Period = %q, want empty", got)
	}
}

func TestWeekGamesSkipsAGameWithoutBothTeams(t *testing.T) {
	week := weekGamesFrom(t, `[
		{"status":"pre_game","metadata":{"home_team":"ARI","away_team":"DET"}},
		{"status":"pre_game","metadata":{"away_team":"KC"}},
		{"status":"pre_game","metadata":{"home_team":"NYJ"}},
		{"status":"pre_game","metadata":{"home_team":"SEA","away_team":"TB"}}
	]`)

	for _, team := range []string{"KC", "NYJ"} {
		if _, ok := week.ForTeam(team); ok {
			t.Errorf("ForTeam(%s) found a game, want it skipped", team)
		}
	}
	gameFor(t, week, "ARI")
	gameFor(t, week, "SEA")
}

func TestWeekGamesFailsNamingTheWeek(t *testing.T) {
	unavailable := func(t *testing.T) string {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		t.Cleanup(srv.Close)
		return srv.URL
	}
	tests := []struct {
		name    string
		baseURL func(t *testing.T) string
	}{
		{"503", unavailable},
		{"not a list", func(t *testing.T) string { return jsonServer(t, `{"not":"a list"}`).URL }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := (Client{BaseURL: tt.baseURL(t)}).WeekGames(context.Background(), 2026, 5)

			if err == nil || !strings.Contains(err.Error(), "season 2026 week 5") {
				t.Errorf("err = %v, want one naming season 2026 week 5", err)
			}
		})
	}
}

func TestWeekGamesEmptyListIsAnEmptyWeek(t *testing.T) {
	if week := weekGamesFrom(t, `[]`); !week.Empty() {
		t.Error("Empty() = false, want true")
	}
}

func TestWeekGamesLogsTheFetch(t *testing.T) {
	srv := jsonServer(t, `[
		{"status":"pre_game","metadata":{"home_team":"ARI","away_team":"DET"}},
		{"status":"pre_game","metadata":{"home_team":"SEA","away_team":"TB"}}
	]`)
	logs := &logRecorder{}
	c := Client{BaseURL: srv.URL, Logf: logs.logf}

	if _, err := c.WeekGames(context.Background(), 2026, 5); err != nil {
		t.Fatalf("WeekGames: %v", err)
	}

	if !strings.Contains(logs.joined(), "sleeper games 2026 w5: 2 games in ") {
		t.Errorf("logs = %q, want the games line", logs.joined())
	}
}

// The fixture is a real response for a finished week: fifteen games, all
// complete, ARI's ending in regulation and three in overtime.
func TestWeekGamesDecodesARecordedWeek(t *testing.T) {
	srv := jsonServer(t, string(readFixture(t, "testdata/games_2026_w5.json")))
	logs := &logRecorder{}
	c := Client{BaseURL: srv.URL, Logf: logs.logf}

	week, err := c.WeekGames(context.Background(), 2026, 5)
	if err != nil {
		t.Fatalf("WeekGames: %v", err)
	}

	if !strings.Contains(logs.joined(), ": 15 games in ") {
		t.Errorf("logs = %q, want 15 games", logs.joined())
	}
	if g := gameFor(t, week, "ARI"); g.State != games.Complete || g.Period != "F" {
		t.Errorf("ARI State, Period = %v, %q; want Complete, F", g.State, g.Period)
	}
}
