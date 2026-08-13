package score

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fixtureServer serves testdata/week14.json, failing the test if FetchStatLine
// asks for any path but the expected one.
func fixtureServer(t *testing.T) *httptest.Server {
	t.Helper()

	const wantPath = "/v1/stats/nfl/regular/2025/14"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

// jsonServer serves one canned body regardless of path.
func jsonServer(t *testing.T, body string) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestFetchStatLinePlayerAbsent(t *testing.T) {
	srv := fixtureServer(t)

	_, err := FetchStatLine(context.Background(), srv.URL, 2025, 14, "does-not-exist")
	if err == nil {
		t.Fatal("FetchStatLine returned no error for an absent player; a silent zero score is exactly the failure this guards")
	}

	// The message has to identify what was looked up, or a rotted hardcoded
	// ID is indistinguishable from a bad week.
	for _, want := range []string{"does-not-exist", "2025", "14"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

// Sleeper returns a null entry for some rostered-but-idle players. A nil map
// reads every stat as zero, so this has to fail the same way an absent player
// does rather than scoring a plausible 0.0.
func TestFetchStatLineNullEntry(t *testing.T) {
	srv := jsonServer(t, `{"9493": null}`)

	_, err := FetchStatLine(context.Background(), srv.URL, 2025, 14, NacuaPlayerID)
	if err == nil {
		t.Fatal("FetchStatLine returned no error for a null player entry; a silent zero score is exactly the failure this guards")
	}
	if !strings.Contains(err.Error(), NacuaPlayerID) {
		t.Errorf("error %q does not mention player %q", err, NacuaPlayerID)
	}
}

// A stalled read must be interruptible, or a hung upstream pins the handler
// goroutine for the life of the process.
func TestFetchStatLineContextCancelled(t *testing.T) {
	blocked := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-blocked
	}))
	t.Cleanup(func() {
		close(blocked)
		srv.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := FetchStatLine(ctx, srv.URL, 2025, 14, NacuaPlayerID)
		done <- err
	}()

	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("FetchStatLine returned no error after its context was cancelled")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("FetchStatLine ignored context cancellation and blocked on the stalled server")
	}
}

func TestFetchStatLine(t *testing.T) {
	srv := fixtureServer(t)

	tests := []struct {
		name     string
		playerID string
		want     StatLine
	}{
		{
			// Nacua's real week 14 line: 7 catches for 167 and 2 TDs, no
			// carries. rush_yd, fum_lost and the 40+ TD keys are absent from
			// his fixture entry, so this also covers missing keys reading as
			// zero.
			name:     "player present in the weekly payload",
			playerID: NacuaPlayerID,
			want: StatLine{
				PlayerID: "9493",
				Season:   2025,
				Week:     14,
				RecYd:    167,
				RecTD:    2,
			},
		},
		{
			// James Cook, the fixture's second entry, carries rush_yd and
			// fum_lost where Nacua's does not — so indexing the wrong player
			// cannot pass both cases.
			name:     "a different player in the same payload",
			playerID: "8138",
			want: StatLine{
				PlayerID: "8138",
				Season:   2025,
				Week:     14,
				RushYd:   80,
				RecYd:    31,
				FumLost:  1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FetchStatLine(context.Background(), srv.URL, 2025, 14, tt.playerID)
			if err != nil {
				t.Fatalf("FetchStatLine: %v", err)
			}
			if got != tt.want {
				t.Errorf("FetchStatLine = %+v, want %+v", got, tt.want)
			}
		})
	}
}
