package sleeper

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"
)

type weekRecorder struct {
	mu    sync.Mutex
	weeks []weekKey
}

func (r *weekRecorder) record(season, week int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.weeks = append(r.weeks, weekKey{season, week})
}

func (r *weekRecorder) calls() []weekKey {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]weekKey(nil), r.weeks...)
}

func TestWholeWeekFetchThatAddsPlaysReportsTheWeek(t *testing.T) {
	ps := newPlayServer(t)
	ps.releaseWholeFetches()
	ps.wholeBody = readFixture(t, "testdata/plays_2026_w3.json")
	store := NewPlayStore(ps.URL, 0, noLog)
	changed := &weekRecorder{}
	store.OnWholeWeekChanged(changed.record)

	store.ForcedFumbles(context.Background(), 2026, 3, w3Identities(t))
	store.wait()

	got := changed.calls()
	if len(got) != 1 || got[0] != (weekKey{2026, 3}) {
		t.Errorf("whole-week changes reported = %v, want one for 2026 w3", got)
	}
}

func TestWholeWeekFetchThatChangesNothingReportsNothing(t *testing.T) {
	changedAt := time.Date(2026, 9, 28, 20, 0, 0, 0, time.UTC)
	held := play{ID: "a", UpdatedAt: changedAt.UnixMilli()}
	ps := newPlayServer(t)
	ps.releaseWholeFetches()
	ps.pollBody = jsonPlays(t, held)
	ps.wholeBody = jsonPlays(t, held)
	store := NewPlayStore(ps.URL, 0, noLog)
	store.now = func() time.Time { return changedAt.Add(2 * time.Hour) } // quiet, so a post-game refresh runs
	store.merge(2026, 3, []play{held})
	changed := &weekRecorder{}
	store.OnWholeWeekChanged(changed.record)

	store.ForcedFumbles(context.Background(), 2026, 3, w3Identities(t))
	store.wait()

	if got := ps.wholeFetches(); got != 1 {
		t.Fatalf("whole-week fetches = %d, want the post-game refresh", got)
	}
	if got := changed.calls(); len(got) != 0 {
		t.Errorf("whole-week changes reported = %v, want none", got)
	}
}

func TestFailedWholeWeekFetchReportsNothing(t *testing.T) {
	ps := newPlayServer(t)
	ps.releaseWholeFetches()
	ps.wholeStatus = http.StatusBadGateway
	ps.wholeBody = readFixture(t, "testdata/plays_2026_w3.json")
	store := NewPlayStore(ps.URL, 0, noLog)
	changed := &weekRecorder{}
	store.OnWholeWeekChanged(changed.record)

	store.ForcedFumbles(context.Background(), 2026, 3, w3Identities(t))
	store.wait()

	if got := changed.calls(); len(got) != 0 {
		t.Errorf("whole-week changes reported = %v, want none", got)
	}
}

func TestPollThatAddsPlaysReportsNothing(t *testing.T) {
	ps := newPlayServer(t)
	ps.pollBody = jsonPlays(t, play{ID: "old", UpdatedAt: 100}, play{ID: "new", UpdatedAt: 200})
	store := NewPlayStore(ps.URL, 0, noLog)
	store.now = func() time.Time { return time.UnixMilli(200) } // live week, overlapping poll: no whole-week fetch
	store.merge(2026, 3, []play{{ID: "old", UpdatedAt: 100}})
	changed := &weekRecorder{}
	store.OnWholeWeekChanged(changed.record)

	store.ForcedFumbles(context.Background(), 2026, 3, w3Identities(t))
	store.wait()

	if got := ps.wholeFetches(); got != 0 {
		t.Fatalf("whole-week fetches = %d, want 0", got)
	}
	if got := changed.calls(); len(got) != 0 {
		t.Errorf("whole-week changes reported = %v, want none", got)
	}
}
