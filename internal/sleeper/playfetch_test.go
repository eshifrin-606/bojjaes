package sleeper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
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

func w3Identities(t *testing.T) identities {
	return fixtureIdentities(t, "testdata/stats_2026_w3.json")
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
	unreachable := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	unreachable.Close()

	tests := map[string]struct {
		baseURL    func(t *testing.T) string
		wantPrefix string
	}{
		"transport failure": {
			baseURL:    func(*testing.T) string { return unreachable.URL },
			wantPrefix: "fetching sleeper plays for season 2026 week 3: ",
		},
		"non-200": {
			baseURL: func(t *testing.T) string {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					http.Error(w, `[]`, http.StatusBadGateway)
				}))
				t.Cleanup(srv.Close)
				return srv.URL
			},
			wantPrefix: "fetching sleeper plays for season 2026 week 3: status 502 Bad Gateway",
		},
		"bad body": {
			baseURL:    func(t *testing.T) string { return jsonServer(t, `{"not":"an array"}`).URL },
			wantPrefix: "decoding sleeper plays for season 2026 week 3: ",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := fetchRecentPlays(context.Background(), tt.baseURL(t), 2026, 3, 300)
			if err == nil {
				t.Fatal("want error")
			}
			if !strings.HasPrefix(err.Error(), tt.wantPrefix) {
				t.Errorf("error %q, want prefix %q", err, tt.wantPrefix)
			}
		})
	}
}

func TestForcedFumblesPollsMergesAndAttributes(t *testing.T) {
	ps := newPlayServer(t)
	ps.pollBody = readFixture(t, "testdata/plays_2026_w3.json")
	store := NewPlayStore(ps.URL, 0, noLog)

	got := store.ForcedFumbles(context.Background(), 2026, 3, w3Identities(t))

	assertCredits(t, got, forcedFumbleTurnovers(loadPlays(t, "testdata/plays_2026_w3.json"), w3Identities(t), noLog))
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
	store := NewPlayStore(ps.URL, 0, noLog)

	returned := make(chan struct{})
	go func() {
		store.ForcedFumbles(context.Background(), 2026, 3, nil)
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
	store := NewPlayStore(ps.URL, 0, noLog)
	store.now = func() time.Time { return time.UnixMilli(200) } // live week, so only the gap can trigger
	store.merge(2026, 3, []play{{ID: "old", UpdatedAt: 100}})

	store.ForcedFumbles(context.Background(), 2026, 3, w3Identities(t))
	store.wait()

	if got := ps.wholeFetches(); got != 1 {
		t.Errorf("whole-week fetches = %d, want 1", got)
	}
}

func TestForcedFumblesGapDoesNotWaitWhateverTheColdWait(t *testing.T) {
	ps := newPlayServer(t)
	ps.pollBody = jsonPlays(t, play{ID: "new", UpdatedAt: 200})
	store := NewPlayStore(ps.URL, time.Minute, noLog)
	store.now = func() time.Time { return time.UnixMilli(200) }
	store.merge(2026, 3, []play{{ID: "old", UpdatedAt: 100}})

	returned := make(chan struct{})
	go func() {
		store.ForcedFumbles(context.Background(), 2026, 3, w3Identities(t))
		close(returned)
	}()

	awaitSignal(t, ps.wholeStarted, "gap's whole-week fetch to start")
	awaitSignal(t, returned, "ForcedFumbles to return while the gap's whole-week fetch is blocked")
}

func TestForcedFumblesOverlappingPollSkipsWholeWeekFetch(t *testing.T) {
	ps := newPlayServer(t)
	ps.releaseWholeFetches()
	ps.pollBody = jsonPlays(t, play{ID: "old", UpdatedAt: 100}, play{ID: "new", UpdatedAt: 200})
	store := NewPlayStore(ps.URL, 0, noLog)
	store.now = func() time.Time { return time.UnixMilli(200) }
	store.merge(2026, 3, []play{{ID: "old", UpdatedAt: 100}})

	store.ForcedFumbles(context.Background(), 2026, 3, w3Identities(t))
	store.wait()

	if got := ps.wholeFetches(); got != 0 {
		t.Errorf("whole-week fetches = %d, want 0", got)
	}
}

func TestForcedFumblesConcurrentTriggersShareOneWholeWeekFetch(t *testing.T) {
	ps := newPlayServer(t)
	store := NewPlayStore(ps.URL, 0, noLog)

	store.ForcedFumbles(context.Background(), 2026, 3, w3Identities(t))
	awaitSignal(t, ps.wholeStarted, "first whole-week fetch to start")
	store.ForcedFumbles(context.Background(), 2026, 3, w3Identities(t))
	ps.releaseWholeFetches()
	store.wait()

	if got := ps.wholeFetches(); got != 1 {
		t.Errorf("whole-week fetches = %d, want 1", got)
	}
}

func TestForcedFumblesAttributesWholeWeekPlaysOnNextCall(t *testing.T) {
	ps := newPlayServer(t)
	ps.wholeBody = readFixture(t, "testdata/plays_2026_w3.json")
	store := NewPlayStore(ps.URL, 0, noLog)

	first := store.ForcedFumbles(context.Background(), 2026, 3, w3Identities(t))
	awaitSignal(t, ps.wholeStarted, "whole-week fetch to start")
	ps.releaseWholeFetches()
	store.wait()
	second := store.ForcedFumbles(context.Background(), 2026, 3, w3Identities(t))

	assertCredits(t, first, map[string]int{})
	assertCredits(t, second, forcedFumbleTurnovers(loadPlays(t, "testdata/plays_2026_w3.json"), w3Identities(t), noLog))
}

func TestForcedFumblesColdWeekWaitsForWholeWeekPlays(t *testing.T) {
	ps := newPlayServer(t)
	ps.releaseWholeFetches()
	ps.wholeBody = readFixture(t, "testdata/plays_2026_w3.json")
	store := NewPlayStore(ps.URL, 5*time.Second, noLog)

	got := store.ForcedFumbles(context.Background(), 2026, 3, w3Identities(t))

	assertCredits(t, got, forcedFumbleTurnovers(loadPlays(t, "testdata/plays_2026_w3.json"), w3Identities(t), noLog))
}

func TestForcedFumblesColdWaitRunsOut(t *testing.T) {
	ps := newPlayServer(t)
	ps.pollBody = readFixture(t, "testdata/plays_2026_w3.json")
	ps.wholeBody = jsonPlays(t, play{ID: "whole-only", UpdatedAt: 1})
	store := NewPlayStore(ps.URL, 10*time.Millisecond, noLog)

	got := make(chan map[string]int)
	go func() { got <- store.ForcedFumbles(context.Background(), 2026, 3, w3Identities(t)) }()

	select {
	case credits := <-got:
		assertCredits(t, credits, forcedFumbleTurnovers(loadPlays(t, "testdata/plays_2026_w3.json"), w3Identities(t), noLog))
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for ForcedFumbles to stop waiting on the blocked whole-week fetch")
	}
	ps.releaseWholeFetches()
	store.wait()
	if !slices.Contains(snapshotIDs(store, 2026, 3), "whole-only") {
		t.Error("store is missing the whole-week play merged after the wait ran out")
	}
}

func TestForcedFumblesColdWaitEndsWithCallerContext(t *testing.T) {
	ps := newPlayServer(t)
	store := NewPlayStore(ps.URL, time.Minute, noLog)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	got := make(chan map[string]int)
	go func() { got <- store.ForcedFumbles(ctx, 2026, 3, w3Identities(t)) }()

	select {
	case credits := <-got:
		assertCredits(t, credits, map[string]int{})
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for ForcedFumbles to give up on a done context")
	}
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
	store := NewPlayStore(ps.URL, 0, logs.logf)

	got := store.ForcedFumbles(context.Background(), 2026, 3, w3Identities(t))
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
	store := NewPlayStore(ps.URL, 0, logs.logf)
	store.merge(2026, 3, loadPlays(t, "testdata/plays_2026_w3.json"))

	got := store.ForcedFumbles(context.Background(), 2026, 3, w3Identities(t))
	store.wait()

	assertCredits(t, got, forcedFumbleTurnovers(loadPlays(t, "testdata/plays_2026_w3.json"), w3Identities(t), noLog))
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
	store := NewPlayStore(ps.URL, 0, logs.logf)
	store.merge(2026, 3, []play{{ID: "old", UpdatedAt: 100}})

	store.ForcedFumbles(context.Background(), 2026, 3, w3Identities(t))
	store.wait()
	store.merge(2026, 3, []play{{ID: "newer", UpdatedAt: 300}})
	ps.pollBody = jsonPlays(t, play{ID: "newest", UpdatedAt: 400})
	store.ForcedFumbles(context.Background(), 2026, 3, w3Identities(t))
	store.wait()

	assertIDs(t, snapshotIDs(store, 2026, 3), "new", "newer", "newest", "old")
	if !strings.Contains(logs.joined(), "whole-week") {
		t.Errorf("logs = %q, want the whole-week failure", logs.joined())
	}
	if got := ps.wholeFetches(); got != 2 {
		t.Errorf("whole-week fetches = %d, want 2 (retry after failure)", got)
	}
}

func TestForcedFumblesColdWaitEndsWhenWholeWeekFetchFails(t *testing.T) {
	ps := newPlayServer(t)
	ps.releaseWholeFetches()
	ps.wholeStatus = http.StatusBadGateway
	ps.pollBody = readFixture(t, "testdata/plays_2026_w3.json")
	logs := &logRecorder{}
	store := NewPlayStore(ps.URL, time.Minute, logs.logf)

	got := make(chan map[string]int)
	go func() { got <- store.ForcedFumbles(context.Background(), 2026, 3, w3Identities(t)) }()

	select {
	case credits := <-got:
		assertCredits(t, credits, forcedFumbleTurnovers(loadPlays(t, "testdata/plays_2026_w3.json"), w3Identities(t), noLog))
	case <-time.After(5 * time.Second):
		t.Fatal("timed out: ForcedFumbles kept waiting after the whole-week fetch failed")
	}
	if !strings.Contains(logs.joined(), "whole-week") {
		t.Errorf("logs = %q, want the whole-week failure", logs.joined())
	}
}

func TestForcedFumblesBoundsHungPoll(t *testing.T) {
	ps := newPlayServer(t)
	ps.pollBlock = make(chan struct{})
	t.Cleanup(func() { close(ps.pollBlock) })
	logs := &logRecorder{}
	store := NewPlayStore(ps.URL, 0, logs.logf)
	store.pollTimeout = 10 * time.Millisecond

	returned := make(chan struct{})
	go func() {
		store.ForcedFumbles(context.Background(), 2026, 3, nil)
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
	store := NewPlayStore(ps.URL, 0, noLog)
	store.now = func() time.Time { return changed.Add(2 * time.Hour) }
	store.merge(2026, 3, []play{{ID: "a", UpdatedAt: changed.UnixMilli()}})

	store.ForcedFumbles(context.Background(), 2026, 3, w3Identities(t))
	store.wait()

	if got := ps.wholeFetches(); got != 1 {
		t.Errorf("whole-week fetches = %d, want 1 post-game refresh", got)
	}

	store.ForcedFumbles(context.Background(), 2026, 3, w3Identities(t))
	store.wait()

	if got := ps.wholeFetches(); got != 1 {
		t.Errorf("whole-week fetches after a second read = %d, want still 1", got)
	}
}

func (s *PlayStore) wait() { s.bg.Wait() }

func TestForcedFumblesLogsPoll(t *testing.T) {
	ps := newPlayServer(t)
	ps.pollBody = jsonPlays(t, play{ID: "old", UpdatedAt: 100}, play{ID: "new", UpdatedAt: 200})
	logs := &logRecorder{}
	store := NewPlayStore(ps.URL, 0, logs.logf)
	store.now = func() time.Time { return time.UnixMilli(200) }
	store.merge(2026, 3, []play{{ID: "old", UpdatedAt: 100}})

	store.ForcedFumbles(context.Background(), 2026, 3, w3Identities(t))

	if !strings.Contains(logs.joined(), "sleeper plays poll 2026 w3: 2 plays in 0s") {
		t.Errorf("logs = %q, want the poll line", logs.joined())
	}
}

func TestForcedFumblesLogsColdWholeWeekStart(t *testing.T) {
	ps := newPlayServer(t)
	logs := &logRecorder{}
	store := NewPlayStore(ps.URL, 0, logs.logf)

	store.ForcedFumbles(context.Background(), 2026, 3, w3Identities(t))

	if !strings.Contains(logs.joined(), "sleeper whole-week 2026 w3: started (cold)") {
		t.Errorf("logs = %q, want the cold start line", logs.joined())
	}
}

func TestForcedFumblesLogsGapWholeWeekStart(t *testing.T) {
	ps := newPlayServer(t)
	ps.pollBody = jsonPlays(t, play{ID: "new", UpdatedAt: 200})
	logs := &logRecorder{}
	store := NewPlayStore(ps.URL, 0, logs.logf)
	store.now = func() time.Time { return time.UnixMilli(200) }
	store.merge(2026, 3, []play{{ID: "old", UpdatedAt: 100}})

	store.ForcedFumbles(context.Background(), 2026, 3, w3Identities(t))

	if !strings.Contains(logs.joined(), "sleeper whole-week 2026 w3: started (gap)") {
		t.Errorf("logs = %q, want the gap start line", logs.joined())
	}
}

func TestForcedFumblesLogsPostGameWholeWeekStart(t *testing.T) {
	changed := time.Date(2026, 9, 28, 20, 0, 0, 0, time.UTC)
	ps := newPlayServer(t)
	ps.pollBody = jsonPlays(t, play{ID: "a", UpdatedAt: changed.UnixMilli()})
	logs := &logRecorder{}
	store := NewPlayStore(ps.URL, 0, logs.logf)
	store.now = func() time.Time { return changed.Add(2 * time.Hour) }
	store.merge(2026, 3, []play{{ID: "a", UpdatedAt: changed.UnixMilli()}})

	store.ForcedFumbles(context.Background(), 2026, 3, w3Identities(t))

	if !strings.Contains(logs.joined(), "sleeper whole-week 2026 w3: started (postgame)") {
		t.Errorf("logs = %q, want the postgame start line", logs.joined())
	}
}

func TestForcedFumblesJoiningWholeWeekFetchLogsNoStart(t *testing.T) {
	ps := newPlayServer(t)
	logs := &logRecorder{}
	store := NewPlayStore(ps.URL, 0, logs.logf)

	store.ForcedFumbles(context.Background(), 2026, 3, w3Identities(t))
	awaitSignal(t, ps.wholeStarted, "first whole-week fetch to start")
	store.ForcedFumbles(context.Background(), 2026, 3, w3Identities(t))

	if got := strings.Count(logs.joined(), "started ("); got != 1 {
		t.Errorf("start lines = %d, want 1; logs = %q", got, logs.joined())
	}
}

func TestForcedFumblesLogsWholeWeekFetch(t *testing.T) {
	ps := newPlayServer(t)
	ps.releaseWholeFetches()
	ps.wholeBody = jsonPlays(t, play{ID: "a", UpdatedAt: 100}, play{ID: "b", UpdatedAt: 200})
	logs := &logRecorder{}
	store := NewPlayStore(ps.URL, 0, logs.logf)
	store.now = func() time.Time { return time.UnixMilli(200) }

	store.ForcedFumbles(context.Background(), 2026, 3, w3Identities(t))
	store.wait()

	if !strings.Contains(logs.joined(), "sleeper whole-week 2026 w3: 2 plays in 0s") {
		t.Errorf("logs = %q, want the whole-week line", logs.joined())
	}
}
