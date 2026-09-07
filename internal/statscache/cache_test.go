package statscache

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/eshifrin/bojjaes/internal/api"
	"github.com/eshifrin/bojjaes/internal/score"
	"github.com/eshifrin/bojjaes/internal/web"
)

// The cache is only useful if it can stand in for the provider wherever one is
// taken, so both transports' interfaces are asserted here rather than
// discovered at the composition root.
var (
	_ api.StatsSource = (*Cache)(nil)
	_ web.StatsSource = (*Cache)(nil)
)

// testTTL stands in for the production TTL; expiry tests move the clock
// relative to it rather than depending on its value.
const testTTL = 5 * time.Minute

// fakePlayerID is the single player every fake fetch returns. Tests read it
// back to tell one fetch's stats from another's.
const fakePlayerID = "p1"

// fakeSource counts its calls and stamps each result with that call's ordinal,
// so a test can assert not just how many fetches happened but which fetch a
// caller was served from.
type fakeSource struct {
	mu    sync.Mutex
	calls int

	// err, when set, fails every fetch. blocks holds, per week, a channel a
	// fetch for that week waits on, so one week can be held open while
	// another runs.
	err    error
	blocks map[int]chan struct{}
}

func (f *fakeSource) WeekStats(ctx context.Context, season, week int) (score.WeekStats, error) {
	f.mu.Lock()
	f.calls++
	ordinal := f.calls
	err, block := f.err, f.blocks[week]
	f.mu.Unlock()

	if block != nil {
		select {
		case <-block:
		case <-ctx.Done():
			return score.WeekStats{}, ctx.Err()
		}
	}
	if err != nil {
		return score.WeekStats{}, err
	}
	return score.NewWeekStats(season, week, map[string]score.StatLine{
		fakePlayerID: {PlayerID: fakePlayerID, PassYd: ordinal},
	}), nil
}

// block holds every fetch for one week open until release is called.
func (f *fakeSource) block(week int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.blocks == nil {
		f.blocks = make(map[int]chan struct{})
	}
	f.blocks[week] = make(chan struct{})
}

func (f *fakeSource) release(week int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	close(f.blocks[week])
	delete(f.blocks, week)
}

// fail makes every subsequent fetch return err; fail(nil) restores success.
func (f *fakeSource) fail(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = err
}

func (f *fakeSource) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// fetchOrdinal reports which of the fake's fetches produced these stats.
func fetchOrdinal(t *testing.T, stats score.WeekStats) int {
	t.Helper()

	line, ok := stats.Player(fakePlayerID)
	if !ok {
		t.Fatalf("stats hold no line for %s", fakePlayerID)
	}
	return line.PassYd
}

func TestWeekStatsReachesTheSource(t *testing.T) {
	source := &fakeSource{}
	cache := New(source, testTTL)

	stats, err := cache.WeekStats(context.Background(), 2025, 15)
	if err != nil {
		t.Fatalf("WeekStats: %v", err)
	}

	if got := source.callCount(); got != 1 {
		t.Errorf("upstream called %d times, want 1", got)
	}
	line, ok := stats.Player(fakePlayerID)
	if !ok {
		t.Fatalf("stats hold no line for %s", fakePlayerID)
	}
	if line.Season != 2025 || line.Week != 15 {
		t.Errorf("stats are for %d week %d, want 2025 week 15", line.Season, line.Week)
	}
}

func TestSecondRequestWithinTTLIsServedFromCache(t *testing.T) {
	source := &fakeSource{}
	cache, clock := newTestCache(source, testTTL)

	first, err := cache.WeekStats(context.Background(), 2025, 15)
	if err != nil {
		t.Fatalf("first WeekStats: %v", err)
	}

	clock.advance(time.Minute)

	second, err := cache.WeekStats(context.Background(), 2025, 15)
	if err != nil {
		t.Fatalf("second WeekStats: %v", err)
	}

	if got := source.callCount(); got != 1 {
		t.Errorf("upstream called %d times, want 1", got)
	}
	if got, want := fetchOrdinal(t, second), fetchOrdinal(t, first); got != want {
		t.Errorf("second caller was served fetch %d, want fetch %d", got, want)
	}
}

func TestDifferentWeeksDoNotShareAnEntry(t *testing.T) {
	source := &fakeSource{}
	cache := New(source, testTTL)

	fifteen, err := cache.WeekStats(context.Background(), 2025, 15)
	if err != nil {
		t.Fatalf("week 15 WeekStats: %v", err)
	}
	sixteen, err := cache.WeekStats(context.Background(), 2025, 16)
	if err != nil {
		t.Fatalf("week 16 WeekStats: %v", err)
	}

	if got := source.callCount(); got != 2 {
		t.Errorf("upstream called %d times, want 2", got)
	}
	if line, _ := fifteen.Player(fakePlayerID); line.Week != 15 {
		t.Errorf("week 15's caller was served week %d", line.Week)
	}
	if line, _ := sixteen.Player(fakePlayerID); line.Week != 16 {
		t.Errorf("week 16's caller was served week %d", line.Week)
	}

	// Week 16's fetch must not have displaced week 15's entry.
	again, err := cache.WeekStats(context.Background(), 2025, 15)
	if err != nil {
		t.Fatalf("second week 15 WeekStats: %v", err)
	}
	if got := source.callCount(); got != 2 {
		t.Errorf("upstream called %d times after re-reading week 15, want 2", got)
	}
	if got, want := fetchOrdinal(t, again), fetchOrdinal(t, fifteen); got != want {
		t.Errorf("week 15's re-read was served fetch %d, want fetch %d", got, want)
	}
}

// fakeClock drives the cache's expiry without sleeping.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{t: time.Date(2025, time.December, 14, 13, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

// newTestCache wires a cache to a clock the test controls.
func newTestCache(source StatsSource, ttl time.Duration) (*Cache, *fakeClock) {
	clock := newFakeClock()
	cache := New(source, ttl)
	cache.now = clock.now
	return cache, clock
}

func TestRequestAfterTTLFetchesAgain(t *testing.T) {
	source := &fakeSource{}
	cache, clock := newTestCache(source, testTTL)

	if _, err := cache.WeekStats(context.Background(), 2025, 15); err != nil {
		t.Fatalf("first WeekStats: %v", err)
	}

	clock.advance(6 * time.Minute)

	second, err := cache.WeekStats(context.Background(), 2025, 15)
	if err != nil {
		t.Fatalf("second WeekStats: %v", err)
	}

	if got := source.callCount(); got != 2 {
		t.Errorf("upstream called %d times, want 2", got)
	}
	if got := fetchOrdinal(t, second); got != 2 {
		t.Errorf("second caller was served fetch %d, want fetch 2", got)
	}
}

// The TTL boundary is asserted on both sides and on the boundary itself. An
// age of exactly the TTL is deliberately a miss: an entry lives for the TTL,
// so the instant it has fully elapsed is already outside.
func TestEntryLivesUpToTheTTLAndNotBeyond(t *testing.T) {
	tests := []struct {
		name      string
		age       time.Duration
		wantCalls int
	}{
		{name: "just before expiry", age: testTTL - time.Nanosecond, wantCalls: 1},
		{name: "exactly at expiry", age: testTTL, wantCalls: 2},
		{name: "just after expiry", age: testTTL + time.Nanosecond, wantCalls: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := &fakeSource{}
			cache, clock := newTestCache(source, testTTL)

			if _, err := cache.WeekStats(context.Background(), 2025, 15); err != nil {
				t.Fatalf("first WeekStats: %v", err)
			}

			clock.advance(tt.age)

			stats, err := cache.WeekStats(context.Background(), 2025, 15)
			if err != nil {
				t.Fatalf("second WeekStats: %v", err)
			}

			if got := source.callCount(); got != tt.wantCalls {
				t.Errorf("upstream called %d times, want %d", got, tt.wantCalls)
			}
			if got := fetchOrdinal(t, stats); got != tt.wantCalls {
				t.Errorf("caller was served fetch %d, want fetch %d", got, tt.wantCalls)
			}
		})
	}
}

// fetchedAt reads an entry's recorded fetch time. Nothing in production reads
// it yet; it is stored so the as-of timestamp can report when the data was
// fetched rather than when the page was requested.
func fetchedAt(t *testing.T, cache *Cache, season, week int) time.Time {
	t.Helper()

	cache.mu.Lock()
	defer cache.mu.Unlock()

	e, ok := cache.entries[key{season: season, week: week}]
	if !ok {
		t.Fatalf("no cached entry for %d week %d", season, week)
	}
	return e.fetchedAt
}

func TestEntryRecordsItsFetchTimeNotItsReadTime(t *testing.T) {
	source := &fakeSource{}
	cache, clock := newTestCache(source, testTTL)

	fetchTime := clock.now()
	if _, err := cache.WeekStats(context.Background(), 2025, 15); err != nil {
		t.Fatalf("first WeekStats: %v", err)
	}
	if got := fetchedAt(t, cache, 2025, 15); !got.Equal(fetchTime) {
		t.Fatalf("entry recorded %v, want the fetch time %v", got, fetchTime)
	}

	clock.advance(3 * time.Minute)

	if _, err := cache.WeekStats(context.Background(), 2025, 15); err != nil {
		t.Fatalf("second WeekStats: %v", err)
	}
	if got := source.callCount(); got != 1 {
		t.Fatalf("upstream called %d times, want 1", got)
	}
	if got := fetchedAt(t, cache, 2025, 15); !got.Equal(fetchTime) {
		t.Errorf("a cache hit moved the recorded time to %v, want the fetch time %v", got, fetchTime)
	}
}

// Freshness is pull-driven: an expired entry is not refreshed until someone
// asks for it. Nothing sweeps and nothing polls.
func TestIdleCacheMakesNoCalls(t *testing.T) {
	source := &fakeSource{}
	cache, clock := newTestCache(source, testTTL)

	if _, err := cache.WeekStats(context.Background(), 2025, 15); err != nil {
		t.Fatalf("WeekStats: %v", err)
	}

	clock.advance(time.Hour)

	if got := source.callCount(); got != 1 {
		t.Errorf("upstream called %d times over an idle hour, want 1", got)
	}
}

// waitForCalls blocks until the fake has been called at least n times, so a
// test can be sure a flight is under way before it releases or joins it.
func waitForCalls(t *testing.T, source *fakeSource, n int) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for source.callCount() < n {
		if time.Now().After(deadline) {
			t.Fatalf("upstream called %d times, waited for %d", source.callCount(), n)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestSimultaneousMissesShareOneFetch(t *testing.T) {
	const callers = 10

	source := &fakeSource{}
	source.block(15)
	cache, _ := newTestCache(source, testTTL)

	var started, done sync.WaitGroup
	started.Add(callers)
	done.Add(callers)
	results := make([]score.WeekStats, callers)
	errs := make([]error, callers)

	for i := range callers {
		go func() {
			defer done.Done()
			started.Done()
			results[i], errs[i] = cache.WeekStats(context.Background(), 2025, 15)
		}()
	}

	started.Wait()
	waitForCalls(t, source, 1)
	source.release(15)
	done.Wait()

	if got := source.callCount(); got != 1 {
		t.Errorf("upstream called %d times, want 1", got)
	}
	for i := range callers {
		if errs[i] != nil {
			t.Fatalf("caller %d: %v", i, errs[i])
		}
		if got := fetchOrdinal(t, results[i]); got != 1 {
			t.Errorf("caller %d was served fetch %d, want fetch 1", i, got)
		}
	}
}

func TestASlowFetchDoesNotBlockAnotherWeek(t *testing.T) {
	source := &fakeSource{}
	source.block(15)
	cache, _ := newTestCache(source, testTTL)

	fifteen := make(chan error, 1)
	go func() {
		_, err := cache.WeekStats(context.Background(), 2025, 15)
		fifteen <- err
	}()
	waitForCalls(t, source, 1)

	sixteen := make(chan error, 1)
	go func() {
		_, err := cache.WeekStats(context.Background(), 2025, 16)
		sixteen <- err
	}()

	select {
	case err := <-sixteen:
		if err != nil {
			t.Errorf("week 16 WeekStats: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("week 16 did not return while week 15's fetch was still in flight")
	}

	source.release(15)
	if err := <-fifteen; err != nil {
		t.Errorf("week 15 WeekStats: %v", err)
	}
}

func TestARequestAfterTheFlightCompletesIsAHit(t *testing.T) {
	source := &fakeSource{}
	source.block(15)
	cache, _ := newTestCache(source, testTTL)

	waiters := make(chan score.WeekStats, 2)
	for range 2 {
		go func() {
			stats, err := cache.WeekStats(context.Background(), 2025, 15)
			if err != nil {
				t.Errorf("waiting WeekStats: %v", err)
			}
			waiters <- stats
		}()
	}
	waitForCalls(t, source, 1)
	source.release(15)
	<-waiters
	<-waiters

	late, err := cache.WeekStats(context.Background(), 2025, 15)
	if err != nil {
		t.Fatalf("late WeekStats: %v", err)
	}
	if got := source.callCount(); got != 1 {
		t.Errorf("upstream called %d times, want 1", got)
	}
	if got := fetchOrdinal(t, late); got != 1 {
		t.Errorf("the late caller was served fetch %d, want fetch 1", got)
	}
}

// errUpstream stands in for any upstream failure; the cache must not remember
// one, so a momentary problem does not outlive itself by a whole TTL.
var errUpstream = errors.New("upstream is down")

func TestAFailureIsReturnedNotStored(t *testing.T) {
	source := &fakeSource{}
	source.fail(errUpstream)
	cache, _ := newTestCache(source, testTTL)

	if _, err := cache.WeekStats(context.Background(), 2025, 15); !errors.Is(err, errUpstream) {
		t.Fatalf("failed WeekStats returned %v, want %v", err, errUpstream)
	}

	source.fail(nil)

	stats, err := cache.WeekStats(context.Background(), 2025, 15)
	if err != nil {
		t.Fatalf("WeekStats after the failure: %v", err)
	}
	if got := source.callCount(); got != 2 {
		t.Fatalf("upstream called %d times, want 2", got)
	}
	if got := fetchOrdinal(t, stats); got != 2 {
		t.Errorf("caller was served fetch %d, want fetch 2", got)
	}
}

func TestAFailedFlightFailsAllItsWaiters(t *testing.T) {
	const callers = 5

	source := &fakeSource{}
	source.fail(errUpstream)
	source.block(15)
	cache, _ := newTestCache(source, testTTL)

	var started, done sync.WaitGroup
	started.Add(callers)
	done.Add(callers)
	results := make([]score.WeekStats, callers)
	errs := make([]error, callers)

	for i := range callers {
		go func() {
			defer done.Done()
			started.Done()
			results[i], errs[i] = cache.WeekStats(context.Background(), 2025, 15)
		}()
	}

	started.Wait()
	waitForCalls(t, source, 1)
	source.release(15)
	done.Wait()

	for i := range callers {
		if !errors.Is(errs[i], errUpstream) {
			t.Errorf("caller %d returned %v, want %v", i, errs[i], errUpstream)
		}
		if _, ok := results[i].Player(fakePlayerID); ok {
			t.Errorf("caller %d was served stats as well as an error", i)
		}
	}

	// A failure must not turn N waiters into N retries.
	if got := source.callCount(); got != 1 {
		t.Errorf("upstream called %d times for one failed flight, want 1", got)
	}
}

func TestASuccessAfterAFailureIsCached(t *testing.T) {
	source := &fakeSource{}
	source.fail(errUpstream)
	cache, _ := newTestCache(source, testTTL)

	if _, err := cache.WeekStats(context.Background(), 2025, 15); !errors.Is(err, errUpstream) {
		t.Fatalf("failed WeekStats returned %v, want %v", err, errUpstream)
	}

	source.fail(nil)

	if _, err := cache.WeekStats(context.Background(), 2025, 15); err != nil {
		t.Fatalf("recovering WeekStats: %v", err)
	}

	third, err := cache.WeekStats(context.Background(), 2025, 15)
	if err != nil {
		t.Fatalf("third WeekStats: %v", err)
	}
	if got := source.callCount(); got != 2 {
		t.Errorf("upstream called %d times, want 2", got)
	}
	if got := fetchOrdinal(t, third); got != 2 {
		t.Errorf("the third caller was served fetch %d, want the cached fetch 2", got)
	}
}

// settleWaiters gives goroutines that have just been launched time to reach
// their wait on the flight. There is nothing observable to poll for: a waiter
// blocked on the entry's channel leaves no trace in the cache.
func settleWaiters() {
	time.Sleep(50 * time.Millisecond)
}

func TestTheLeaderLeavingDoesNotFailTheWaiters(t *testing.T) {
	const waiters = 2

	source := &fakeSource{}
	source.block(15)
	cache, _ := newTestCache(source, testTTL)

	leaderCtx, cancelLeader := context.WithCancel(context.Background())
	defer cancelLeader()
	go cache.WeekStats(leaderCtx, 2025, 15)
	waitForCalls(t, source, 1)

	results := make(chan score.WeekStats, waiters)
	errs := make(chan error, waiters)
	for range waiters {
		go func() {
			stats, err := cache.WeekStats(context.Background(), 2025, 15)
			results <- stats
			errs <- err
		}()
	}
	settleWaiters()

	cancelLeader()
	source.release(15)

	for i := range waiters {
		if err := <-errs; err != nil {
			t.Errorf("waiter %d returned %v, want stats", i, err)
		}
		if got := fetchOrdinal(t, <-results); got != 1 {
			t.Errorf("waiter %d was served fetch %d, want fetch 1", i, got)
		}
	}
	if got := source.callCount(); got != 1 {
		t.Errorf("upstream called %d times, want 1", got)
	}
}

func TestAWaitersCancellationIsItsOwn(t *testing.T) {
	source := &fakeSource{}
	source.block(15)
	cache, _ := newTestCache(source, testTTL)

	go cache.WeekStats(context.Background(), 2025, 15)
	waitForCalls(t, source, 1)

	leavingCtx, cancelLeaving := context.WithCancel(context.Background())
	defer cancelLeaving()
	leaving := make(chan error, 1)
	go func() {
		_, err := cache.WeekStats(leavingCtx, 2025, 15)
		leaving <- err
	}()

	staying := make(chan score.WeekStats, 1)
	stayingErr := make(chan error, 1)
	go func() {
		stats, err := cache.WeekStats(context.Background(), 2025, 15)
		staying <- stats
		stayingErr <- err
	}()
	settleWaiters()

	cancelLeaving()
	select {
	case err := <-leaving:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("the cancelled waiter returned %v, want %v", err, context.Canceled)
		}
	case <-time.After(2 * time.Second):
		t.Error("the cancelled waiter did not return while the flight was still in progress")
	}

	source.release(15)
	if err := <-stayingErr; err != nil {
		t.Errorf("the remaining waiter returned %v, want stats", err)
	}
	if got := fetchOrdinal(t, <-staying); got != 1 {
		t.Errorf("the remaining waiter was served fetch %d, want fetch 1", got)
	}
	if got := source.callCount(); got != 1 {
		t.Errorf("upstream called %d times, want 1", got)
	}
}

func TestAFetchThatOutlastsTheTimeoutFailsAndIsNotCached(t *testing.T) {
	source := &fakeSource{}
	source.block(15)
	cache, _ := newTestCache(source, testTTL)
	cache.fetchTimeout = 20 * time.Millisecond

	_, err := cache.WeekStats(context.Background(), 2025, 15)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("the timed-out fetch returned %v, want %v", err, context.DeadlineExceeded)
	}

	// Nothing is left behind for the next caller to be served from: a fetch
	// that ran out of time is a failure like any other, and failures are not
	// remembered.
	source.release(15)
	stats, err := cache.WeekStats(context.Background(), 2025, 15)
	if err != nil {
		t.Fatalf("the retry returned %v, want stats", err)
	}
	if got := fetchOrdinal(t, stats); got != 2 {
		t.Errorf("the retry was served fetch %d, want fetch 2", got)
	}
	if got := source.callCount(); got != 2 {
		t.Errorf("upstream called %d times, want 2", got)
	}
}
