// Package statscache holds a short-lived cache over a weekly stat fetch, so
// that upstream volume is bounded by the time-to-live and the number of weeks
// being read rather than by the number of readers.
//
// It declares the StatsSource interface it wraps, exactly as the transports
// do, so no provider package is named here and the cache is a StatsSource
// itself: it is installed by wrapping, and nothing downstream is edited.
package statscache

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/eshifrin/bojjaes/internal/score"
)

// StatsSource supplies one season and week's stats.
type StatsSource interface {
	WeekStats(ctx context.Context, season, week int) (score.WeekStats, error)
}

// key is the whole of a cache entry's identity: one fetch returns every
// player in the league, so there is no partial hit and nothing to merge.
type key struct {
	season, week int
}

type entry struct {
	stats score.WeekStats
	err   error

	// fetchedAt is when the upstream call completed, not when the entry was
	// last read: it is what expiry is measured from, and it is what a later
	// as-of timestamp must report so a cached page dates its data honestly.
	fetchedAt time.Time

	// done is closed when the fetch that installed this entry has finished.
	// An entry with done still open is a flight in progress, not a result:
	// later callers for the same key wait on it instead of fetching again.
	done chan struct{}
}

func (e *entry) inFlight() bool {
	select {
	case <-e.done:
		return false
	default:
		return true
	}
}

// TTL is how long a fetched week is served from memory. It is short enough
// that a live week's scores are never far behind, and long enough that a page
// refreshing itself every few seconds costs the same upstream as one that does
// not. Callers name it rather than a duration, so there is one place to change.
const TTL = 5 * time.Minute

// defaultFetchTimeout bounds how long one upstream call may hold a flight
// open. It is the cache's own bound, not one borrowed from whatever it wraps:
// a decorator must not assume its source has a timeout at all. It is set
// generously — outliving it fails the flight, and the next request retries.
const defaultFetchTimeout = 30 * time.Second

// Cache is safe for concurrent use, and only useful shared: one instance
// wrapping one source is one upstream budget, so a caller that constructs its
// own has opted out of the bound rather than gaining a private copy of it.
type Cache struct {
	source StatsSource
	ttl    time.Duration

	// fetchTimeout is a seam like now: a test cannot wait out the real bound.
	fetchTimeout time.Duration

	// now is a seam, not a feature: a five-minute TTL cannot be waited out in
	// a test, and a boundary is worth asserting on both sides of.
	now func() time.Time

	// logf records each miss — one upstream call spent against the Sleeper
	// budget. A test swaps it to read the lines back.
	logf func(format string, args ...any)

	mu      sync.Mutex
	entries map[key]*entry
}

// Option adjusts a Cache at construction. Options exist for tests — main
// constructs a Cache with none, so the clock and the fetch timeout stay
// unexported seams rather than configuration.
type Option func(*Cache)

// WithClock replaces the wall clock the Cache reads fetch instants and expiry
// from. A test in another package that needs to move time or pin a fetch
// instant against a known value passes one.
func WithClock(now func() time.Time) Option {
	return func(c *Cache) { c.now = now }
}

func New(source StatsSource, ttl time.Duration, opts ...Option) *Cache {
	c := &Cache{
		source:       source,
		ttl:          ttl,
		fetchTimeout: defaultFetchTimeout,
		now:          time.Now,
		logf:         log.Printf,
		entries:      make(map[key]*entry),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// WeekStats returns a week's stats, discarding the fetch instant. It is the
// path for callers that do not report freshness (internal/api); it must stay a
// thin delegate so single-flight, TTL, and error handling cannot diverge from
// WeekStatsAsOf.
func (c *Cache) WeekStats(ctx context.Context, season, week int) (score.WeekStats, error) {
	stats, _, err := c.WeekStatsAsOf(ctx, season, week)
	return stats, err
}

// WeekStatsAsOf returns a week's stats together with the instant the fetch that
// produced them completed. Every caller served from one entry — the caller who
// drove the fetch, a cache hit, a released waiter — gets that entry's recorded
// fetchedAt, so a page served from cache can date its data to the fetch rather
// than to the request.
func (c *Cache) WeekStatsAsOf(ctx context.Context, season, week int) (score.WeekStats, time.Time, error) {
	k := key{season: season, week: week}

	c.mu.Lock()
	cached, ok := c.entries[k]
	if ok && cached.inFlight() {
		c.mu.Unlock()
		select {
		case <-cached.done:
			// fetchedAt is written under c.mu before close(done) on the success
			// path, so a released waiter reads a stable value; on the failure
			// path it stays zero and err is non-nil.
			return cached.stats, cached.fetchedAt, cached.err
		case <-ctx.Done():
			// A waiter leaving takes only itself: the flight runs on, and the
			// other waiters are still served by it.
			return score.WeekStats{}, time.Time{}, ctx.Err()
		}
	}
	if ok && !c.expired(cached) {
		c.mu.Unlock()
		return cached.stats, cached.fetchedAt, nil
	}
	flight := &entry{done: make(chan struct{})}
	c.entries[k] = flight
	c.mu.Unlock()

	// The mutex is deliberately not held here. A slow upstream for one week
	// must not block a request for a different one.
	//
	// The fetch is detached from the caller that happened to start it: the
	// first reader closing their tab must not cancel the call four other
	// readers are waiting on — a failure that gets more likely the better
	// single-flight works. The leader is the one caller that cannot walk away,
	// so its exposure is bounded by the cache's own timeout instead.
	fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), c.fetchTimeout)
	start := c.now()
	flight.stats, flight.err = c.source.WeekStats(fetchCtx, season, week)
	cancel()

	// One line per miss: a miss is one upstream call spent, and misses arrive
	// about once per week per TTL, so the log stays a readable record of the
	// Sleeper budget. Hits are not logged — every reader's refresh is one.
	c.logMiss(season, week, c.now().Sub(start), flight.err)

	// A failure is never stored. Remembering one would turn a momentary
	// upstream blip into a whole TTL of errors on a page that refreshes
	// itself; dropping the entry costs one upstream call per arrival while
	// the outage lasts, which single-flight already bounds.
	c.mu.Lock()
	if flight.err != nil {
		delete(c.entries, k)
	} else {
		flight.fetchedAt = c.now()
	}
	c.mu.Unlock()
	close(flight.done)

	return flight.stats, flight.fetchedAt, flight.err
}

func (c *Cache) logMiss(season, week int, elapsed time.Duration, err error) {
	if err != nil {
		c.logf("statscache miss: %d week %d failed after %s: %v", season, week, elapsed, err)
		return
	}
	c.logf("statscache miss: %d week %d fetched in %s", season, week, elapsed)
}

// expired is checked on read; nothing sweeps. The key space is a handful of
// weeks in a season, so a background goroutine would cost more machinery than
// the memory it reclaims.
//
// An age of exactly the TTL is expired: the requirement is that an entry lives
// *for* the TTL, so the instant it has fully elapsed is already outside.
func (c *Cache) expired(e *entry) bool {
	return c.now().Sub(e.fetchedAt) >= c.ttl
}
