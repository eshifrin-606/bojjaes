package score

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// countingFixtureServer serves testdata/week14.json and records how many
// upstream requests the handler actually made. The response is labelled from
// the request rather than the fetch, so a handler asking upstream for the
// wrong week would still look right — the path assertion is what catches it.
func countingFixtureServer(t *testing.T, requests *int) *httptest.Server {
	t.Helper()

	const wantPath = "/v1/stats/nfl/regular/2025/14"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*requests++
		if r.URL.Path != wantPath {
			t.Errorf("requested %q, want %q", r.URL.Path, wantPath)
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "testdata/week14.json")
	}))
	t.Cleanup(srv.Close)
	return srv
}

func postScores(t *testing.T, h http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/scores", strings.NewReader(body))
	h.ServeHTTP(rec, req)
	return rec
}

func decodeBatch(t *testing.T, rec *httptest.ResponseRecorder) BatchResponse {
	t.Helper()

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body)
	}

	var got BatchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding response %s: %v", rec.Body, err)
	}
	return got
}

func TestBatchHandler(t *testing.T) {
	tests := []struct {
		name        string
		playerIDs   []string
		wantScored  map[string]float64
		wantNoStats []string
	}{
		{
			name:      "all players present",
			playerIDs: []string{NacuaPlayerID, "8138"},
			// Cook: 111 combined yards pays 3.5, one lost fumble costs 3.
			wantScored:  map[string]float64{NacuaPlayerID: 19, "8138": 0.5},
			wantNoStats: []string{},
		},
		{
			name:        "a mix of present and absent players",
			playerIDs:   []string{NacuaPlayerID, "does-not-exist"},
			wantScored:  map[string]float64{NacuaPlayerID: 19},
			wantNoStats: []string{"does-not-exist"},
		},
		{
			// A player who dressed and recorded nothing is a real 0.0, and
			// must not be reported as absent.
			name:        "a present but scoreless player is scored, not absent",
			playerIDs:   []string{"7591"},
			wantScored:  map[string]float64{"7591": 0},
			wantNoStats: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests int
			srv := countingFixtureServer(t, &requests)

			body, _ := json.Marshal(BatchRequest{Season: 2025, Week: 14, PlayerIDs: tt.playerIDs})
			got := decodeBatch(t, postScores(t, BatchHandler(srv.URL), string(body)))

			// One upstream fetch serves every requested player; scoring a
			// roster must not fan out into a request per player.
			if requests != 1 {
				t.Errorf("made %d upstream requests, want 1", requests)
			}

			if len(got.Scores) != len(tt.wantScored) {
				t.Fatalf("scored %d players, want %d (%+v)", len(got.Scores), len(tt.wantScored), got.Scores)
			}
			for _, s := range got.Scores {
				want, ok := tt.wantScored[s.Stats.PlayerID]
				if !ok {
					t.Errorf("unexpected scored player %q", s.Stats.PlayerID)
					continue
				}
				if s.Points != want {
					t.Errorf("player %s points = %v, want %v", s.Stats.PlayerID, s.Points, want)
				}
			}

			if fmt.Sprint(got.NoStats) != fmt.Sprint(tt.wantNoStats) {
				t.Errorf("no_stats = %v, want %v", got.NoStats, tt.wantNoStats)
			}
		})
	}
}

// An unplayed week returns an empty payload. That is a legitimate answer, so
// every player lands in no_stats and the request still succeeds.
func TestBatchHandlerEveryPlayerAbsent(t *testing.T) {
	srv := jsonServer(t, `{}`)

	got := decodeBatch(t, postScores(t, BatchHandler(srv.URL),
		`{"season":2026,"week":1,"player_ids":["9493","8138"]}`))

	if len(got.Scores) != 0 {
		t.Errorf("scored %d players for an unplayed week, want 0", len(got.Scores))
	}
	if len(got.NoStats) != 2 {
		t.Errorf("no_stats = %v, want both requested IDs", got.NoStats)
	}
}

// Splitting the response into two lists is what makes an ID able to vanish
// from both. The counts have to account for every requested player.
func TestBatchHandlerAccountsForEveryRequestedID(t *testing.T) {
	var requests int
	srv := countingFixtureServer(t, &requests)

	// A repeated present ID and a repeated absent one: each occurrence is
	// echoed, so the caller gets back what it asked for.
	playerIDs := []string{NacuaPlayerID, NacuaPlayerID, "8138", "nope", "nope"}
	body, _ := json.Marshal(BatchRequest{Season: 2025, Week: 14, PlayerIDs: playerIDs})

	got := decodeBatch(t, postScores(t, BatchHandler(srv.URL), string(body)))

	if len(got.Scores)+len(got.NoStats) != len(playerIDs) {
		t.Errorf("scores (%d) + no_stats (%d) != requested (%d)",
			len(got.Scores), len(got.NoStats), len(playerIDs))
	}
	if len(got.NoStats) != 2 {
		t.Errorf("no_stats = %v, want the repeated absent ID twice", got.NoStats)
	}
}

// The season and week in the response are echoed from the request, so only
// the upstream path shows which week was actually fetched.
func TestBatchHandlerFetchesTheRequestedWeek(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)

	postScores(t, BatchHandler(srv.URL), `{"season":2023,"week":7,"player_ids":["9493"]}`)

	const wantPath = "/v1/stats/nfl/regular/2023/7"
	if gotPath != wantPath {
		t.Errorf("fetched %q, want %q", gotPath, wantPath)
	}
}

func TestBatchHandlerRejectsMalformedRequests(t *testing.T) {
	// wantMsg pins which problem the error names. An undecodable body leaves
	// every field zero, so season validation would reject it too — but telling
	// the caller their season is out of range when they sent broken JSON sends
	// them after the wrong bug.
	tests := []struct {
		name    string
		body    string
		wantMsg string
	}{
		{name: "body is not valid JSON", body: `{"season":`, wantMsg: "request body"},
		{
			// A truncated upload: the fields that did arrive are complete and
			// in range, so nothing but the decode itself flags it.
			name:    "body is truncated but would otherwise be valid",
			body:    `{"season":2025,"week":14,"player_ids":["9493"]`,
			wantMsg: "request body",
		},
		{name: "season missing", body: `{"week":14,"player_ids":["9493"]}`, wantMsg: "season"},
		{name: "season out of range", body: `{"season":1850,"week":14,"player_ids":["9493"]}`, wantMsg: "season"},
		{name: "week missing", body: `{"season":2025,"player_ids":["9493"]}`, wantMsg: "week"},
		{name: "week out of range", body: `{"season":2025,"week":99,"player_ids":["9493"]}`, wantMsg: "week"},
		{name: "empty player list", body: `{"season":2025,"week":14,"player_ids":[]}`, wantMsg: "player_ids"},
		{name: "player list omitted", body: `{"season":2025,"week":14}`, wantMsg: "player_ids"},
		{name: "too many player IDs", body: playerIDsBody(t, 27), wantMsg: "roster limit"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests int
			srv := countingFixtureServer(t, &requests)

			rec := postScores(t, BatchHandler(srv.URL), tt.body)

			if rec.Code < 400 || rec.Code > 499 {
				t.Errorf("status = %d, want 4xx (body: %s)", rec.Code, rec.Body)
			}
			// A request we already know is malformed must not cost an
			// upstream call.
			if requests != 0 {
				t.Errorf("made %d upstream requests for a malformed request, want 0", requests)
			}
			if !strings.Contains(rec.Body.String(), tt.wantMsg) {
				t.Errorf("error %q does not name the problem (%q)", strings.TrimSpace(rec.Body.String()), tt.wantMsg)
			}
		})
	}
}

// The cap is the league's maximum roster size, so a full roster is a valid
// request rather than an off-by-one rejection.
func TestBatchHandlerAcceptsMaxPlayerIDs(t *testing.T) {
	var requests int
	srv := countingFixtureServer(t, &requests)

	got := decodeBatch(t, postScores(t, BatchHandler(srv.URL), playerIDsBody(t, maxPlayerIDs)))

	if len(got.Scores)+len(got.NoStats) != maxPlayerIDs {
		t.Errorf("accounted for %d players, want %d", len(got.Scores)+len(got.NoStats), maxPlayerIDs)
	}
}

func TestBatchHandlerUpstreamFailure(t *testing.T) {
	var logged bytes.Buffer
	log.SetOutput(&logged)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "sleeper is down", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	rec := postScores(t, BatchHandler(srv.URL), `{"season":2025,"week":14,"player_ids":["9493"]}`)

	if rec.Code < 500 || rec.Code > 599 {
		t.Errorf("status = %d, want 5xx", rec.Code)
	}
	// A partial body would read as "these players scored nothing" rather than
	// "we do not know".
	if strings.Contains(rec.Body.String(), `"scores"`) {
		t.Errorf("served a partial result instead of an error: %s", rec.Body)
	}
	if rec.Body.Len() == 0 {
		t.Error("failed the request without an error message")
	}
}

// playerIDsBody builds a request body with n distinct player IDs.
func playerIDsBody(t *testing.T, n int) string {
	t.Helper()

	ids := make([]string, n)
	for i := range ids {
		ids[i] = fmt.Sprintf("id-%d", i)
	}

	body, err := json.Marshal(BatchRequest{Season: 2025, Week: 14, PlayerIDs: ids})
	if err != nil {
		t.Fatalf("building request body: %v", err)
	}
	return string(body)
}
