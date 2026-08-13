package score

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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

	h := Handler(func() (StatLine, error) { return stats, nil })

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
	h := Handler(func() (StatLine, error) {
		return StatLine{}, errors.New("sleeper is down")
	})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/score", nil))

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
