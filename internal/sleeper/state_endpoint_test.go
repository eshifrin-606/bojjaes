package sleeper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eshifrin/bojjaes/internal/nflweek"
)

// stateServer serves body at /state/nfl, as Sleeper's v1 host does.
func stateServer(t *testing.T, body []byte) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/state/nfl" {
			http.NotFound(w, r)
			return
		}
		w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestCurrentWeekReadsTheFixture(t *testing.T) {
	srv := stateServer(t, readFixture(t, "testdata/state_nfl.json"))

	current, err := Client{V1BaseURL: srv.URL}.CurrentWeek(context.Background())
	if err != nil {
		t.Fatalf("CurrentWeek: %v", err)
	}

	if season, week, ok := current.Settable(); season != 2026 || week != 9 || !ok {
		t.Errorf("Settable() = %d, %d, %t; want 2026, 9, true", season, week, ok)
	}
}

func TestStatePhase(t *testing.T) {
	for _, tt := range []struct {
		seasonType string
		want       nflweek.Phase
	}{
		{"pre", nflweek.Preseason},
		{"regular", nflweek.Regular},
		{"post", nflweek.Postseason},
		{"off", nflweek.Offseason},
		{"exhibition", nflweek.Other},
	} {
		t.Run(tt.seasonType, func(t *testing.T) {
			if got := statePhase(tt.seasonType); got != tt.want {
				t.Errorf("statePhase(%q) = %v, want %v", tt.seasonType, got, tt.want)
			}
		})
	}
}

// A season that is not a number fails the read, so it is never mistaken for
// "no settable week".
func TestCurrentWeekFailsOnANonNumericSeason(t *testing.T) {
	srv := stateServer(t, []byte(`{"season":"","season_type":"regular","week":9}`))

	if current, err := (Client{V1BaseURL: srv.URL}).CurrentWeek(context.Background()); err == nil {
		t.Errorf("CurrentWeek = %+v, nil; want an error", current)
	}
}

func TestCurrentWeekFailsOnABadResponse(t *testing.T) {
	for _, tt := range []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"error status", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "down", http.StatusServiceUnavailable)
		}},
		{"not the state object", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`[]`))
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			t.Cleanup(srv.Close)

			if current, err := (Client{V1BaseURL: srv.URL}).CurrentWeek(context.Background()); err == nil {
				t.Errorf("CurrentWeek = %+v, nil; want an error", current)
			}
		})
	}
}

func TestCurrentWeekLogsTheStateFetch(t *testing.T) {
	srv := stateServer(t, readFixture(t, "testdata/state_nfl.json"))
	logs := &logRecorder{}

	if _, err := (Client{V1BaseURL: srv.URL, Logf: logs.logf}).CurrentWeek(context.Background()); err != nil {
		t.Fatalf("CurrentWeek: %v", err)
	}

	if !strings.Contains(logs.joined(), "sleeper state: 2026 regular w9 in ") {
		t.Errorf("logs = %q, want the state line", logs.joined())
	}
}

func TestCurrentWeekFailureLogsNoStateLine(t *testing.T) {
	srv := stateServer(t, []byte(`{"season":"","season_type":"regular","week":9}`))
	logs := &logRecorder{}

	if _, err := (Client{V1BaseURL: srv.URL, Logf: logs.logf}).CurrentWeek(context.Background()); err == nil {
		t.Fatal("CurrentWeek succeeded; want an error")
	}

	if strings.Contains(logs.joined(), "sleeper state:") {
		t.Errorf("logs = %q, want no state line", logs.joined())
	}
}
