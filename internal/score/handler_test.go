package score

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

type ctxKey struct{}

// The fetch must run under the request's context so a client disconnect
// cancels the upstream call instead of leaving it in flight.
func TestHandlerPassesRequestContext(t *testing.T) {
	var got context.Context
	h := Handler(func(ctx context.Context) (StatLine, error) {
		got = ctx
		return StatLine{}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/score", nil)
	req = req.WithContext(context.WithValue(req.Context(), ctxKey{}, "from-request"))
	h.ServeHTTP(httptest.NewRecorder(), req)

	if got == nil {
		t.Fatal("Handler called fetch without a context")
	}
	if got.Value(ctxKey{}) != "from-request" {
		t.Error("Handler passed a context that is not the request's")
	}
}

func TestHandlerSuccess(t *testing.T) {
	// Nacua's real week 14 line. 167 receiving yards pays 3 + eight full
	// 10-yard increments = 7; two touchdowns, neither 40+, add 12.
	stats := StatLine{
		PlayerID: NacuaPlayerID,
		Season:   2025,
		Week:     14,
		RecYd:    167,
		RecTD:    2,
	}
	const wantPoints = 19.0

	h := Handler(func(context.Context) (StatLine, error) { return stats, nil })

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/score", nil))

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

func TestHandlerFetchError(t *testing.T) {
	var logged bytes.Buffer
	log.SetOutput(&logged)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	h := Handler(func(context.Context) (StatLine, error) {
		return StatLine{}, errors.New("sleeper is down")
	})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/score", nil))

	// Without a server-side trace, an operator watching the process sees
	// nothing at all when the upstream is down.
	if !strings.Contains(logged.String(), "sleeper is down") {
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
	if !strings.Contains(rec.Body.String(), "sleeper is down") {
		t.Errorf("body does not carry the underlying error: %s", rec.Body)
	}
}
