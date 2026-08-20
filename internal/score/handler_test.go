package score

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// A player who dressed and recorded nothing scores a real 0.0. An `omitempty`
// on Points would erase it, leaving a scored player indistinguishable from an
// absent one in the JSON.
func TestBatchResponseKeepsGenuineZero(t *testing.T) {
	resp := newBatchResponse(2025, 14)
	resp.Scores = append(resp.Scores, ScoreResponse{
		Stats:  StatLine{PlayerID: "7591", Season: 2025, Week: 14},
		Points: 0,
	})

	body, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshalling: %v", err)
	}

	if !strings.Contains(string(body), `"points":0`) {
		t.Errorf("a zero score did not serialize an explicit points field: %s", body)
	}
}

// Absent buckets must read as "nothing here", not as "field missing". A `null`
// forces every caller to nil-check before ranging.
func TestBatchResponseEmptyBucketsAreArrays(t *testing.T) {
	body, err := json.Marshal(newBatchResponse(2026, 1))
	if err != nil {
		t.Fatalf("marshalling: %v", err)
	}

	for _, want := range []string{`"scores":[]`, `"no_stats":[]`} {
		if !strings.Contains(string(body), want) {
			t.Errorf("empty response does not contain %s: %s", want, body)
		}
	}
}

// The fetch must run under the request's context so a client disconnect
// cancels the upstream call instead of leaving it in flight.
func TestHandlerPassesRequestContext(t *testing.T) {
	srv := fixtureServer(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/score", nil).WithContext(ctx)
	Handler(srv.URL).ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Error("Handler served a score under a cancelled request context; the fetch is not running under it")
	}
}

func TestHandlerSuccess(t *testing.T) {
	// Nacua's real week 14 line. 167 receiving yards pays 3 + eight full
	// 10-yard increments = 7; two touchdowns, neither 40+, add 12.
	stats := StatLine{
		PlayerID: NacuaPlayerID,
		Season:   TargetSeason,
		Week:     TargetWeek,
		RecYd:    167,
		RecTD:    2,
	}
	const wantPoints = 19.0

	srv := fixtureServer(t)

	rec := httptest.NewRecorder()
	Handler(srv.URL).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/score", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body)
	}

	var got ScoreResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding response %s: %v", rec.Body, err)
	}

	if got.Stats != stats {
		t.Errorf("stats = %+v, want %+v", got.Stats, stats)
	}
	if got.Points != wantPoints {
		t.Errorf("points = %v, want %v", got.Points, wantPoints)
	}
}

// The target is a settled week in which the player is known present, so his
// absence means the fetch or the transform broke — not that he has no stats.
func TestHandlerTargetPlayerAbsent(t *testing.T) {
	var logged bytes.Buffer
	log.SetOutput(&logged)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	srv := jsonServer(t, `{}`)

	rec := httptest.NewRecorder()
	Handler(srv.URL).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/score", nil))

	if rec.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want %d (body: %s)", rec.Code, http.StatusBadGateway, rec.Body)
	}
	if strings.Contains(rec.Body.String(), `"points":0`) {
		t.Errorf("body served a zero score instead of an error: %s", rec.Body)
	}
	if !strings.Contains(rec.Body.String(), NacuaPlayerID) {
		t.Errorf("body does not name the missing player: %s", rec.Body)
	}
}

func TestHandlerFetchError(t *testing.T) {
	var logged bytes.Buffer
	log.SetOutput(&logged)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "sleeper is down", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	rec := httptest.NewRecorder()
	Handler(srv.URL).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/score", nil))

	// Without a server-side trace, an operator watching the process sees
	// nothing at all when the upstream is down.
	if !strings.Contains(logged.String(), "500") {
		t.Errorf("fetch error was not logged; log output: %q", logged.String())
	}

	if rec.Code < 500 || rec.Code > 599 {
		t.Errorf("status = %d, want 5xx", rec.Code)
	}

	// A zero score served with a 200 is the failure mode worth guarding: it
	// reads as "he scored nothing" rather than "we do not know".
	if strings.Contains(rec.Body.String(), `"points":0`) {
		t.Errorf("body served a zero score instead of an error: %s", rec.Body)
	}
	if !strings.Contains(rec.Body.String(), "500") {
		t.Errorf("body does not carry the underlying error: %s", rec.Body)
	}
}
