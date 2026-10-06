package sleeper

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

const crosbyID = "5991"

// w3Aggregate serves the recorded week 3 aggregate after edit has changed its
// rows.
func w3Aggregate(t *testing.T, edit func(rows []map[string]any) []map[string]any) string {
	t.Helper()
	var rows []map[string]any
	if err := json.Unmarshal(readFixture(t, "testdata/stats_2026_w3.json"), &rows); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(edit(rows))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func unchanged(rows []map[string]any) []map[string]any { return rows }

func editCrosby(edit func(row map[string]any) map[string]any) func([]map[string]any) []map[string]any {
	return func(rows []map[string]any) []map[string]any {
		var out []map[string]any
		for _, r := range rows {
			if r["player_id"] == crosbyID {
				r = edit(r)
			}
			if r != nil {
				out = append(out, r)
			}
		}
		return out
	}
}

func TestClientWeekStatsCreditsForcerFromPlays(t *testing.T) {
	srv := weekServer(t, w3Aggregate(t, unchanged), readFixture(t, "testdata/plays_2026_w3.json"))
	c := Client{BaseURL: srv.URL, Plays: NewPlayStore(srv.URL, 0, noLog)}

	week, err := c.WeekStats(context.Background(), 2026, 3)
	if err != nil {
		t.Fatalf("WeekStats: %v", err)
	}

	line, ok := week.Player(crosbyID)
	if !ok {
		t.Fatalf("Player(%s) absent", crosbyID)
	}
	if line.FFTurnover != 1 {
		t.Errorf("FFTurnover = %d, want 1", line.FFTurnover)
	}
	if line.FumRec != 1 {
		t.Errorf("FumRec = %d, want 1 from the aggregate", line.FumRec)
	}
}

func TestClientWeekStatsCreditsForcerWithNullAggregateStats(t *testing.T) {
	nullStats := editCrosby(func(r map[string]any) map[string]any { r["stats"] = nil; return r })
	srv := weekServer(t, w3Aggregate(t, nullStats), readFixture(t, "testdata/plays_2026_w3.json"))
	c := Client{BaseURL: srv.URL, Plays: NewPlayStore(srv.URL, 0, noLog)}

	week, err := c.WeekStats(context.Background(), 2026, 3)
	if err != nil {
		t.Fatalf("WeekStats: %v", err)
	}

	line, ok := week.Player(crosbyID)
	if !ok {
		t.Fatalf("Player(%s) absent", crosbyID)
	}
	if want := (score.StatLine{PlayerID: crosbyID, Season: 2026, Week: 3, FFTurnover: 1}); line != want {
		t.Errorf("line = %+v, want %+v", line, want)
	}
}

func TestClientWeekStatsDoesNotCreditForcerMissingFromAggregate(t *testing.T) {
	dropped := editCrosby(func(map[string]any) map[string]any { return nil })
	srv := weekServer(t, w3Aggregate(t, dropped), readFixture(t, "testdata/plays_2026_w3.json"))
	logs := &logRecorder{}
	c := Client{BaseURL: srv.URL, Plays: NewPlayStore(srv.URL, 0, logs.logf)}

	week, err := c.WeekStats(context.Background(), 2026, 3)
	if err != nil {
		t.Fatalf("WeekStats: %v", err)
	}

	if line, ok := week.Player(crosbyID); ok {
		t.Errorf("Player(%s) = %+v, want absent", crosbyID, line)
	}
	if !strings.Contains(logs.joined(), "ff5e0d30") {
		t.Errorf("logs = %q, want Crosby's play", logs.joined())
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
	c := Client{BaseURL: srv.URL, Plays: NewPlayStore(srv.URL, 0, noLog)}

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
	c := Client{BaseURL: srv.URL, Plays: NewPlayStore(srv.URL, 0, noLog)}

	if _, err := c.WeekStats(context.Background(), 2026, 3); err == nil {
		t.Fatal("want error")
	}
}
