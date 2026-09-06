package sleeper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eshifrin/bojjaes/internal/score"
)

func TestFetchWeekStatsReturnsAPlayerFromTheFixture(t *testing.T) {
	srv := fixtureServer(t)

	week, err := FetchWeekStats(context.Background(), srv.URL, 2025, 14)
	if err != nil {
		t.Fatalf("FetchWeekStats: %v", err)
	}

	line, ok := week.Player("9493")
	if !ok {
		t.Fatalf("Player(9493) reported absent; want found")
	}
	if line.RecYd != 167 {
		t.Errorf("RecYd = %d, want 167", line.RecYd)
	}
}

// A null entry decodes to a nil map, which reads every stat as zero. Carrying
// it into the snapshot would turn "we have nothing for this player" into a
// scoreless week.
func TestFetchWeekStatsSkipsNullEntries(t *testing.T) {
	srv := jsonServer(t, `{"9493": null}`)

	week, err := FetchWeekStats(context.Background(), srv.URL, 2025, 14)
	if err != nil {
		t.Fatalf("FetchWeekStats: %v", err)
	}

	if line, ok := week.Player("9493"); ok {
		t.Errorf("Player(9493) reported found with %+v; want absent", line)
	}
}

// An unplayed week returns 200 with `{}`. That is an answer, not a failure.
func TestFetchWeekStatsEmptyPayloadIsNotAnError(t *testing.T) {
	srv := jsonServer(t, `{}`)

	week, err := FetchWeekStats(context.Background(), srv.URL, 2025, 18)
	if err != nil {
		t.Fatalf("FetchWeekStats on an empty payload: %v", err)
	}

	if line, ok := week.Player("9493"); ok {
		t.Errorf("Player(9493) reported found with %+v; want absent", line)
	}
}

func TestFetchWeekStatsUpstreamFailureIsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "sleeper is down", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	week, err := FetchWeekStats(context.Background(), srv.URL, 2025, 14)
	if err == nil {
		t.Fatal("FetchWeekStats returned no error on a 500")
	}
	for _, want := range []string{"2025", "14"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name %s", err, want)
		}
	}

	// A snapshot alongside an error is a snapshot someone will read.
	if _, ok := week.Player(nacuaPlayerID); ok {
		t.Error("a populated WeekStats was returned alongside the error")
	}
}

// The settled week that GET /score existed to prove: Nacua's real 2025 week 14
// line, fetched and transformed and scored. It cannot change, so a difference
// here is the fetch or the transform breaking.
func TestFetchWeekStatsSettledWeek(t *testing.T) {
	srv := fixtureServer(t)

	week, err := FetchWeekStats(context.Background(), srv.URL, 2025, 14)
	if err != nil {
		t.Fatalf("FetchWeekStats: %v", err)
	}

	got, ok := week.Player(nacuaPlayerID)
	if !ok {
		t.Fatalf("Player(%s) reported absent in a week he played", nacuaPlayerID)
	}

	// 167 receiving yards pays 3 + eight full 10-yard increments = 7; two
	// touchdowns, neither 40+, add 12.
	want := score.StatLine{PlayerID: nacuaPlayerID, Season: 2025, Week: 14, RecYd: 167, RecTD: 2}
	if got != want {
		t.Errorf("stats = %+v, want %+v", got, want)
	}
	if pts := score.Points(got); pts != 19 {
		t.Errorf("Points = %v, want 19", pts)
	}
}
