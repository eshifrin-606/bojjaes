package sleeper

import (
	"context"
	"sync"
	"time"
)

type weekKey struct{ season, week int }

type PlayStore struct {
	mu          sync.Mutex
	baseURL     string
	logf        func(format string, args ...any)
	pollTimeout time.Duration
	coldWait    time.Duration
	now         func() time.Time
	bg          sync.WaitGroup // background fetches, so tests can wait for them

	weeks          map[weekKey]map[string]play
	lastWholeFetch map[weekKey]time.Time
	wholeInFlight  map[weekKey]chan struct{}
}

func (s *PlayStore) merge(season, week int, plays []play) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.weeks == nil {
		s.weeks = map[weekKey]map[string]play{}
	}
	k := weekKey{season, week}
	if s.weeks[k] == nil {
		s.weeks[k] = map[string]play{}
	}
	for _, p := range plays {
		if held, ok := s.weeks[k][p.ID]; ok && p.UpdatedAt <= held.UpdatedAt {
			continue
		}
		s.weeks[k][p.ID] = p
	}
}

func (s *PlayStore) snapshot(season, week int) []play {
	s.mu.Lock()
	defer s.mu.Unlock()

	var out []play
	for _, p := range s.weeks[weekKey{season, week}] {
		out = append(out, p)
	}
	return out
}

func (s *PlayStore) holdsPlays(season, week int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.weeks[weekKey{season, week}]) > 0
}

func (s *PlayStore) needsWholeFetch(season, week int, polled []play) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	held := s.weeks[weekKey{season, week}]
	if len(held) == 0 {
		return true
	}
	return len(polled) > 0 && oldestUpdate(polled) > s.newestHeld(season, week)
}

func (s *PlayStore) newestHeld(season, week int) int64 {
	var newest int64
	for _, p := range s.weeks[weekKey{season, week}] {
		newest = max(newest, p.UpdatedAt)
	}
	return newest
}

func oldestUpdate(plays []play) int64 {
	oldest := plays[0].UpdatedAt
	for _, p := range plays[1:] {
		oldest = min(oldest, p.UpdatedAt)
	}
	return oldest
}

func (s *PlayStore) markWholeFetched(season, week int, at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lastWholeFetch == nil {
		s.lastWholeFetch = map[weekKey]time.Time{}
	}
	s.lastWholeFetch[weekKey{season, week}] = at
}

// quietPeriod is how long a week must go unchanged before it counts as over,
// and how long after its last change a whole fetch must run to be trusted.
const quietPeriod = time.Hour

func (s *PlayStore) needsPostGameRefresh(season, week int, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	newest := time.UnixMilli(s.newestHeld(season, week))
	if now.Sub(newest) <= quietPeriod {
		return false
	}
	return s.lastWholeFetch[weekKey{season, week}].Before(newest.Add(quietPeriod))
}

const pollLimit = 300

func NewPlayStore(baseURL string, coldWait time.Duration, logf func(format string, args ...any)) *PlayStore {
	return &PlayStore{baseURL: baseURL, coldWait: coldWait, logf: logf, pollTimeout: 5 * time.Second, now: time.Now}
}

func (s *PlayStore) ForcedFumbles(ctx context.Context, season, week int, ids identities) map[string]int {
	pollCtx, cancel := context.WithTimeout(ctx, s.pollTimeout)
	defer cancel()
	start := s.now()
	polled, err := fetchRecentPlays(pollCtx, s.baseURL, season, week, pollLimit)
	if err != nil {
		s.logf("sleeper plays poll failed: %v", err)
	} else {
		s.logf("sleeper plays poll %d w%d: %d plays in %s", season, week, len(polled), s.now().Sub(start))
	}
	cold := !s.holdsPlays(season, week)
	var wholeDone <-chan struct{}
	if reason := s.wholeFetchReason(season, week, cold, polled); reason != "" {
		wholeDone = s.startWholeFetch(season, week, reason)
	}
	s.merge(season, week, polled)
	if cold && s.coldWait > 0 {
		select {
		case <-wholeDone:
		case <-time.After(s.coldWait):
		case <-ctx.Done():
		}
	}
	return forcedFumbleTurnovers(s.snapshot(season, week), ids, s.logf)
}

// wholeFetchReason names why a read needs a whole-week fetch, or "" if it
// needs none.
func (s *PlayStore) wholeFetchReason(season, week int, cold bool, polled []play) string {
	switch {
	case cold:
		return "cold"
	case s.needsWholeFetch(season, week, polled):
		return "gap"
	case s.needsPostGameRefresh(season, week, s.now()):
		return "postgame"
	}
	return ""
}

const (
	wholeWeekLimit   = 5000
	wholeFetchBudget = 30 * time.Second
)

// startWholeFetch runs at most one whole-week fetch per week at a time. The
// returned channel closes when that fetch finishes, whether this call started
// it or joined one already running.
func (s *PlayStore) startWholeFetch(season, week int, reason string) <-chan struct{} {
	k := weekKey{season, week}
	s.mu.Lock()
	defer s.mu.Unlock()
	if done, ok := s.wholeInFlight[k]; ok {
		return done
	}
	if s.wholeInFlight == nil {
		s.wholeInFlight = map[weekKey]chan struct{}{}
	}
	done := make(chan struct{})
	s.wholeInFlight[k] = done
	s.logf("sleeper whole-week %d w%d: started (%s)", season, week, reason)
	s.bg.Go(func() {
		s.fetchWholeWeek(season, week)
		s.mu.Lock()
		delete(s.wholeInFlight, k)
		s.mu.Unlock()
		close(done)
	})
	return done
}

// fetchWholeWeek runs detached from any request: a cold-week waiter only
// observes it, so a waiter giving up never cancels it.
func (s *PlayStore) fetchWholeWeek(season, week int) {
	ctx, cancel := context.WithTimeout(context.Background(), wholeFetchBudget)
	defer cancel()
	start := s.now()
	plays, err := fetchRecentPlays(ctx, s.baseURL, season, week, wholeWeekLimit)
	if err != nil {
		s.logf("sleeper whole-week plays fetch failed: %v", err)
		return
	}
	s.logf("sleeper whole-week %d w%d: %d plays in %s", season, week, len(plays), s.now().Sub(start))
	s.merge(season, week, plays)
	s.markWholeFetched(season, week, s.now())
}
