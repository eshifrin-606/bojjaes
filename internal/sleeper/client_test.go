package sleeper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/eshifrin/bojjaes/internal/score"
	"github.com/eshifrin/bojjaes/internal/sleeper/plays"
)

func readFixture(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func noLog(string, ...any) {}

type logRecorder struct {
	mu    sync.Mutex
	lines []string
}

func (l *logRecorder) logf(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lines = append(l.lines, fmt.Sprintf(format, args...))
}

func (l *logRecorder) joined() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return strings.Join(l.lines, "\n")
}

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
	srv := weekServer(t, w3Aggregate(t, unchanged), readFixture(t, "plays/testdata/plays_2026_w3.json"))
	c := Client{BaseURL: srv.URL, Plays: plays.NewStore(srv.URL, 0, noLog)}

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
	srv := weekServer(t, w3Aggregate(t, nullStats), readFixture(t, "plays/testdata/plays_2026_w3.json"))
	c := Client{BaseURL: srv.URL, Plays: plays.NewStore(srv.URL, 0, noLog)}

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
	srv := weekServer(t, w3Aggregate(t, dropped), readFixture(t, "plays/testdata/plays_2026_w3.json"))
	logs := &logRecorder{}
	c := Client{BaseURL: srv.URL, Plays: plays.NewStore(srv.URL, 0, logs.logf)}

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
	c := Client{BaseURL: srv.URL, Plays: plays.NewStore(srv.URL, 0, noLog)}

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
	c := Client{BaseURL: srv.URL, Plays: plays.NewStore(srv.URL, 0, noLog)}

	if _, err := c.WeekStats(context.Background(), 2026, 3); err == nil {
		t.Fatal("want error")
	}
}

func TestClientWeekStatsLogsAggregateFetch(t *testing.T) {
	srv := weekServer(t, `[{"player_id":"1","stats":{}},{"player_id":"2","stats":{}}]`, []byte(`[]`))
	logs := &logRecorder{}
	c := Client{BaseURL: srv.URL, Logf: logs.logf}

	if _, err := c.WeekStats(context.Background(), 2026, 3); err != nil {
		t.Fatalf("WeekStats: %v", err)
	}

	if !strings.Contains(logs.joined(), "sleeper stats 2026 w3: 2 rows in ") {
		t.Errorf("logs = %q, want the aggregate line", logs.joined())
	}
}

func TestClientWeekStatsReturnsAPlayerFromTheFixture(t *testing.T) {
	srv := fixtureServer(t)

	week, err := Client{BaseURL: srv.URL}.WeekStats(context.Background(), 2025, 14)
	if err != nil {
		t.Fatalf("WeekStats: %v", err)
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
func TestClientWeekStatsSkipsNullEntries(t *testing.T) {
	srv := jsonServer(t, `[{"player_id":"9493","stats":null}]`)

	week, err := Client{BaseURL: srv.URL}.WeekStats(context.Background(), 2025, 14)
	if err != nil {
		t.Fatalf("WeekStats: %v", err)
	}

	if line, ok := week.Player("9493"); ok {
		t.Errorf("Player(9493) reported found with %+v; want absent", line)
	}
}

// An unplayed week returns 200 with `[]`. That is an answer, not a failure.
func TestClientWeekStatsEmptyPayloadIsNotAnError(t *testing.T) {
	srv := jsonServer(t, `[]`)

	week, err := Client{BaseURL: srv.URL}.WeekStats(context.Background(), 2025, 18)
	if err != nil {
		t.Fatalf("WeekStats on an empty payload: %v", err)
	}

	if line, ok := week.Player("9493"); ok {
		t.Errorf("Player(9493) reported found with %+v; want absent", line)
	}
}

func TestClientWeekStatsUpstreamFailureIsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "sleeper is down", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	week, err := Client{BaseURL: srv.URL}.WeekStats(context.Background(), 2025, 14)
	if err == nil {
		t.Fatal("WeekStats returned no error on a 500")
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
func TestClientWeekStatsSettledWeek(t *testing.T) {
	srv := fixtureServer(t)

	week, err := Client{BaseURL: srv.URL}.WeekStats(context.Background(), 2025, 14)
	if err != nil {
		t.Fatalf("WeekStats: %v", err)
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
