package sleeper

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/eshifrin/bojjaes/internal/score"
)

// weekServer serves a weekly aggregate and a plays poll from one host, as
// Sleeper does.
func weekServer(t *testing.T, aggregate string, playsBody []byte) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/plays/") {
			w.Write(playsBody)
			return
		}
		w.Write([]byte(aggregate))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func firstForcer(t *testing.T, credits map[string]int) (string, int) {
	t.Helper()
	ids := []string{}
	for id := range credits {
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		t.Fatal("fixture credits no forcer")
	}
	sort.Strings(ids)
	return ids[0], credits[ids[0]]
}

func TestClientWeekStatsCreditsForcerFromPlays(t *testing.T) {
	playsBody := readFixture(t, "testdata/plays_2026_w3.json")
	forcer, want := firstForcer(t, forcedFumbleTurnovers(loadPlays(t, "testdata/plays_2026_w3.json"), noLog))
	srv := weekServer(t, fmt.Sprintf(`[{"player_id":%q,"stats":{"rush_yd":10}}]`, forcer), playsBody)
	c := Client{BaseURL: srv.URL, Plays: NewPlayStore(srv.URL, noLog)}

	week, err := c.WeekStats(context.Background(), 2026, 3)
	if err != nil {
		t.Fatalf("WeekStats: %v", err)
	}

	line, ok := week.Player(forcer)
	if !ok {
		t.Fatalf("Player(%s) absent", forcer)
	}
	if line.FFTurnover != want {
		t.Errorf("FFTurnover = %d, want %d", line.FFTurnover, want)
	}
	if line.RushYd != 10 {
		t.Errorf("RushYd = %d, want 10", line.RushYd)
	}
}

func TestClientWeekStatsAddsForcerAbsentFromAggregate(t *testing.T) {
	playsBody := readFixture(t, "testdata/plays_2026_w3.json")
	forcer, want := firstForcer(t, forcedFumbleTurnovers(loadPlays(t, "testdata/plays_2026_w3.json"), noLog))
	srv := weekServer(t, `[]`, playsBody)
	c := Client{BaseURL: srv.URL, Plays: NewPlayStore(srv.URL, noLog)}

	week, err := c.WeekStats(context.Background(), 2026, 3)
	if err != nil {
		t.Fatalf("WeekStats: %v", err)
	}

	line, ok := week.Player(forcer)
	if !ok {
		t.Fatalf("Player(%s) absent", forcer)
	}
	if want := (score.StatLine{PlayerID: forcer, Season: 2026, Week: 3, FFTurnover: want}); line != want {
		t.Errorf("line = %+v, want %+v", line, want)
	}
}

func TestClientWeekStatsSurvivesPlayFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/plays/") {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.Write([]byte(`[{"player_id":"9493","stats":{"rec_yd":167}}]`))
	}))
	t.Cleanup(srv.Close)
	c := Client{BaseURL: srv.URL, Plays: NewPlayStore(srv.URL, noLog)}

	week, err := c.WeekStats(context.Background(), 2026, 3)
	if err != nil {
		t.Fatalf("WeekStats: %v", err)
	}

	if line, _ := week.Player("9493"); line.RecYd != 167 {
		t.Errorf("RecYd = %d, want 167", line.RecYd)
	}
}

func TestClientWeekStatsAggregateFailureStillErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/plays/") {
			w.Write([]byte(`[]`))
			return
		}
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	c := Client{BaseURL: srv.URL, Plays: NewPlayStore(srv.URL, noLog)}

	if _, err := c.WeekStats(context.Background(), 2026, 3); err == nil {
		t.Fatal("want error")
	}
}
