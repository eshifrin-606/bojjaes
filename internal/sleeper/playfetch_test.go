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
	"sync/atomic"
	"testing"
	"time"
)

func readFixture(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestFetchRecentPlaysRequestsAndDecodes(t *testing.T) {
	body := readFixture(t, "testdata/plays_2026_w3.json")
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.Query().Encode()
		w.Write(body)
	}))
	defer srv.Close()

	got, err := fetchRecentPlays(context.Background(), srv.URL, 2026, 3, 300)
	if err != nil {
		t.Fatal(err)
	}

	if gotPath != "/plays/nfl/recent" {
		t.Errorf("path = %q", gotPath)
	}
	if want := "limit=300&season=2026&season_type=regular&week=3"; gotQuery != want {
		t.Errorf("query = %q, want %q", gotQuery, want)
	}
	if want := len(loadPlays(t, "testdata/plays_2026_w3.json")); len(got) != want {
		t.Errorf("decoded %d plays, want %d", len(got), want)
	}
}

func TestFetchRecentPlaysErrors(t *testing.T) {
	tests := map[string]http.HandlerFunc{
		"non-200":  func(w http.ResponseWriter, r *http.Request) { http.Error(w, `[]`, http.StatusBadGateway) },
		"bad body": func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"not":"an array"}`)) },
	}
	for name, h := range tests {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(h)
			defer srv.Close()

			if _, err := fetchRecentPlays(context.Background(), srv.URL, 2026, 3, 300); err == nil {
				t.Fatal("want error")
			}
		})
	}
}

func TestForcedFumblesPollsMergesAndAttributes(t *testing.T) {
	ps := newPlayServer(t)
	ps.pollBody = readFixture(t, "testdata/plays_2026_w3.json")
	store := NewPlayStore(ps.URL, noLog)

	got := store.ForcedFumbles(context.Background(), 2026, 3)

	assertCredits(t, got, forcedFumbleTurnovers(loadPlays(t, "testdata/plays_2026_w3.json"), noLog))
}

// playServer answers polls (limit 300) immediately and whole-week fetches
// (limit 5000) only once release is closed, so tests decide when a background
// fetch finishes.
type playServer struct {
	*httptest.Server
	pollBody, wholeBody []byte
	pollStatus          int
	wholeStarted        chan struct{}
	release             chan struct{}
	releaseOnce         sync.Once
	wholeCount          atomic.Int32
	wholeStatus         int
	pollBlock           chan struct{} // when set, polls hang until it closes
}

func newPlayServer(t *testing.T) *playServer {
	t.Helper()
	ps := &playServer{
		pollBody:     []byte(`[]`),
		wholeBody:    []byte(`[]`),
		pollStatus:   http.StatusOK,
		wholeStatus:  http.StatusOK,
		wholeStarted: make(chan struct{}, 10),
		release:      make(chan struct{}),
	}
	ps.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") == "5000" {
			ps.wholeCount.Add(1)
			ps.wholeStarted <- struct{}{}
			select {
			case <-ps.release:
			case <-r.Context().Done():
			}
			w.WriteHeader(ps.wholeStatus)
			w.Write(ps.wholeBody)
			return
		}
		if ps.pollBlock != nil {
			select {
			case <-ps.pollBlock:
			case <-r.Context().Done():
			}
		}
		w.WriteHeader(ps.pollStatus)
		w.Write(ps.pollBody)
	}))
	t.Cleanup(func() {
		ps.releaseWholeFetches()
		ps.Close()
	})
	return ps
}

func (ps *playServer) releaseWholeFetches() { ps.releaseOnce.Do(func() { close(ps.release) }) }

func awaitSignal(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
	}
}

func TestForcedFumblesSeedsEmptyWeekInBackground(t *testing.T) {
	ps := newPlayServer(t)
	ps.pollBody = readFixture(t, "testdata/plays_2026_w3.json")
	store := NewPlayStore(ps.URL, noLog)

	returned := make(chan struct{})
	go func() {
		store.ForcedFumbles(context.Background(), 2026, 3)
		close(returned)
	}()

	awaitSignal(t, ps.wholeStarted, "whole-week fetch to start")
	awaitSignal(t, returned, "ForcedFumbles to return while the whole-week fetch is blocked")
}

func jsonPlays(t *testing.T, plays ...play) []byte {
	t.Helper()
	b, err := json.Marshal(plays)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func (ps *playServer) wholeFetches() int { return int(ps.wholeCount.Load()) }

func TestForcedFumblesGapStartsWholeWeekFetch(t *testing.T) {
	ps := newPlayServer(t)
	ps.releaseWholeFetches()
	ps.pollBody = jsonPlays(t, play{ID: "new", UpdatedAt: 200})
	store := NewPlayStore(ps.URL, noLog)
	store.now = func() time.Time { return time.UnixMilli(200) } // live week, so only the gap can trigger
	store.merge(2026, 3, []play{{ID: "old", UpdatedAt: 100}})

	store.ForcedFumbles(context.Background(), 2026, 3)
	store.wait()

	if got := ps.wholeFetches(); got != 1 {
		t.Errorf("whole-week fetches = %d, want 1", got)
	}
}

func TestForcedFumblesOverlappingPollSkipsWholeWeekFetch(t *testing.T) {
	ps := newPlayServer(t)
	ps.releaseWholeFetches()
	ps.pollBody = jsonPlays(t, play{ID: "old", UpdatedAt: 100}, play{ID: "new", UpdatedAt: 200})
	store := NewPlayStore(ps.URL, noLog)
	store.now = func() time.Time { return time.UnixMilli(200) }
	store.merge(2026, 3, []play{{ID: "old", UpdatedAt: 100}})

	store.ForcedFumbles(context.Background(), 2026, 3)
	store.wait()

	if got := ps.wholeFetches(); got != 0 {
		t.Errorf("whole-week fetches = %d, want 0", got)
	}
}

func TestForcedFumblesConcurrentTriggersShareOneWholeWeekFetch(t *testing.T) {
	ps := newPlayServer(t)
	store := NewPlayStore(ps.URL, noLog)

	store.ForcedFumbles(context.Background(), 2026, 3)
	awaitSignal(t, ps.wholeStarted, "first whole-week fetch to start")
	store.ForcedFumbles(context.Background(), 2026, 3)
	ps.releaseWholeFetches()
	store.wait()

	if got := ps.wholeFetches(); got != 1 {
		t.Errorf("whole-week fetches = %d, want 1", got)
	}
}

func TestForcedFumblesAttributesWholeWeekPlaysOnNextCall(t *testing.T) {
	ps := newPlayServer(t)
	ps.wholeBody = readFixture(t, "testdata/plays_2026_w3.json")
	store := NewPlayStore(ps.URL, noLog)

	first := store.ForcedFumbles(context.Background(), 2026, 3)
	awaitSignal(t, ps.wholeStarted, "whole-week fetch to start")
	ps.releaseWholeFetches()
	store.wait()
	second := store.ForcedFumbles(context.Background(), 2026, 3)

	assertCredits(t, first, map[string]int{})
	assertCredits(t, second, forcedFumbleTurnovers(loadPlays(t, "testdata/plays_2026_w3.json"), noLog))
}

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

func TestForcedFumblesPollFailureWithNothingHeld(t *testing.T) {
	ps := newPlayServer(t)
	ps.releaseWholeFetches()
	ps.pollStatus = http.StatusBadGateway
	logs := &logRecorder{}
	store := NewPlayStore(ps.URL, logs.logf)

	got := store.ForcedFumbles(context.Background(), 2026, 3)
	store.wait()

	assertCredits(t, got, map[string]int{})
	if !strings.Contains(logs.joined(), "poll") {
		t.Errorf("logs = %q, want the poll failure", logs.joined())
	}
}

func TestForcedFumblesPollFailureScoresHeldPlays(t *testing.T) {
	ps := newPlayServer(t)
	ps.releaseWholeFetches()
	ps.pollStatus = http.StatusBadGateway
	logs := &logRecorder{}
	store := NewPlayStore(ps.URL, logs.logf)
	store.merge(2026, 3, loadPlays(t, "testdata/plays_2026_w3.json"))

	got := store.ForcedFumbles(context.Background(), 2026, 3)
	store.wait()

	assertCredits(t, got, forcedFumbleTurnovers(loadPlays(t, "testdata/plays_2026_w3.json"), noLog))
	if !strings.Contains(logs.joined(), "poll") {
		t.Errorf("logs = %q, want the poll failure", logs.joined())
	}
}

func TestForcedFumblesWholeWeekFailureKeepsHeldPlaysAndAllowsRetry(t *testing.T) {
	ps := newPlayServer(t)
	ps.releaseWholeFetches()
	ps.wholeStatus = http.StatusBadGateway
	ps.pollBody = jsonPlays(t, play{ID: "new", UpdatedAt: 200})
	logs := &logRecorder{}
	store := NewPlayStore(ps.URL, logs.logf)
	store.merge(2026, 3, []play{{ID: "old", UpdatedAt: 100}})

	store.ForcedFumbles(context.Background(), 2026, 3)
	store.wait()
	store.merge(2026, 3, []play{{ID: "newer", UpdatedAt: 300}})
	ps.pollBody = jsonPlays(t, play{ID: "newest", UpdatedAt: 400})
	store.ForcedFumbles(context.Background(), 2026, 3)
	store.wait()

	assertIDs(t, snapshotIDs(store, 2026, 3), "new", "newer", "newest", "old")
	if !strings.Contains(logs.joined(), "whole-week") {
		t.Errorf("logs = %q, want the whole-week failure", logs.joined())
	}
	if got := ps.wholeFetches(); got != 2 {
		t.Errorf("whole-week fetches = %d, want 2 (retry after failure)", got)
	}
}

func TestForcedFumblesBoundsHungPoll(t *testing.T) {
	ps := newPlayServer(t)
	ps.pollBlock = make(chan struct{})
	t.Cleanup(func() { close(ps.pollBlock) })
	logs := &logRecorder{}
	store := NewPlayStore(ps.URL, logs.logf)
	store.pollTimeout = 10 * time.Millisecond

	returned := make(chan struct{})
	go func() {
		store.ForcedFumbles(context.Background(), 2026, 3)
		close(returned)
	}()

	awaitSignal(t, returned, "ForcedFumbles to give up on the hung poll")
	if !strings.Contains(logs.joined(), "poll") {
		t.Errorf("logs = %q, want the poll failure", logs.joined())
	}
}

func TestForcedFumblesRefreshesQuietWeekOnce(t *testing.T) {
	changed := time.Date(2026, 9, 28, 20, 0, 0, 0, time.UTC)
	ps := newPlayServer(t)
	ps.releaseWholeFetches()
	ps.pollBody = jsonPlays(t, play{ID: "a", UpdatedAt: changed.UnixMilli()})
	store := NewPlayStore(ps.URL, noLog)
	store.now = func() time.Time { return changed.Add(2 * time.Hour) }
	store.merge(2026, 3, []play{{ID: "a", UpdatedAt: changed.UnixMilli()}})

	store.ForcedFumbles(context.Background(), 2026, 3)
	store.wait()

	if got := ps.wholeFetches(); got != 1 {
		t.Errorf("whole-week fetches = %d, want 1 post-game refresh", got)
	}

	store.ForcedFumbles(context.Background(), 2026, 3)
	store.wait()

	if got := ps.wholeFetches(); got != 1 {
		t.Errorf("whole-week fetches after a second read = %d, want still 1", got)
	}
}
