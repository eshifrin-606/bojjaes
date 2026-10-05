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
	now         func() time.Time
	bg          sync.WaitGroup // background fetches, so tests can wait for them

	weeks          map[weekKey]map[string]play
	lastWholeFetch map[weekKey]time.Time
	wholeInFlight  map[weekKey]bool
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

func NewPlayStore(baseURL string, logf func(format string, args ...any)) *PlayStore {
	return &PlayStore{baseURL: baseURL, logf: logf, pollTimeout: 5 * time.Second, now: time.Now}
}

func (s *PlayStore) ForcedFumbles(ctx context.Context, season, week int, ids identities) map[string]int {
	pollCtx, cancel := context.WithTimeout(ctx, s.pollTimeout)
	defer cancel()
	polled, err := fetchRecentPlays(pollCtx, s.baseURL, season, week, pollLimit)
	if err != nil {
		s.logf("sleeper plays poll failed: %v", err)
	}
	if s.needsWholeFetch(season, week, polled) || s.needsPostGameRefresh(season, week, s.now()) {
		s.startWholeFetch(season, week)
	}
	s.merge(season, week, polled)
	return forcedFumbleTurnovers(s.snapshot(season, week), ids, s.logf)
}

const (
	wholeWeekLimit   = 5000
	wholeFetchBudget = 30 * time.Second
)

// startWholeFetch runs at most one whole-week fetch per week at a time.
func (s *PlayStore) startWholeFetch(season, week int) {
	k := weekKey{season, week}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.wholeInFlight[k] {
		return
	}
	if s.wholeInFlight == nil {
		s.wholeInFlight = map[weekKey]bool{}
	}
	s.wholeInFlight[k] = true
	s.bg.Go(func() {
		s.fetchWholeWeek(season, week)
		s.mu.Lock()
		delete(s.wholeInFlight, k)
		s.mu.Unlock()
	})
}

// fetchWholeWeek runs detached from any request: a page must never wait on it.
func (s *PlayStore) fetchWholeWeek(season, week int) {
	ctx, cancel := context.WithTimeout(context.Background(), wholeFetchBudget)
	defer cancel()
	plays, err := fetchRecentPlays(ctx, s.baseURL, season, week, wholeWeekLimit)
	if err != nil {
		s.logf("sleeper whole-week plays fetch failed: %v", err)
		return
	}
	s.merge(season, week, plays)
	s.markWholeFetched(season, week, s.now())
}

func (s *PlayStore) wait() { s.bg.Wait() }
