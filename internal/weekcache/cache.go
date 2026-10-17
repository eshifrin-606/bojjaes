// Package weekcache holds a short-lived cache over a weekly stat fetch, so
// that upstream volume is bounded by the time-to-live and the number of weeks
// being read rather than by the number of readers.
//
// It declares the StatsSource interface it wraps, exactly as the transports
// do, so no provider package is named here and the cache is a StatsSource
// itself: it is installed by wrapping, and nothing downstream is edited.
package weekcache

import (
	"context"
	"log"
	"sync"
	"time"
)

// Source fetches one season and week's value. It is a function rather than an
// interface so a provider's method is passed as a method value and the cache
// needs no interface per value type.
type Source[V any] func(ctx context.Context, season, week int) (V, error)

// key is the whole of a cache entry's identity: one fetch returns every
// player in the league, so there is no partial hit and nothing to merge.
type key struct {
	season, week int
}

type entry[V any] struct {
	value V
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

func (e *entry[V]) inFlight() bool {
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
type Cache[V any] struct {
	// label opens every log line, so lines from caches over different
	// fetches can be told apart.
	label  string
	source Source[V]
	ttl    time.Duration

	// fetchTimeout is a seam like now: a test cannot wait out the real bound.
	fetchTimeout time.Duration

	// now is a seam, not a feature: a five-minute TTL cannot be waited out in
	// a test, and a boundary is worth asserting on both sides of.
	now func() time.Time

	// logf records each miss — one upstream call spent against the Sleeper
	// budget — and each hit. A test swaps it to read the lines back.
	logf func(format string, args ...any)

	mu      sync.Mutex
	entries map[key]*entry[V]
}

// Option adjusts a Cache at construction. Options exist for tests — main
// constructs a Cache with none, so the clock and the fetch timeout stay
// unexported seams rather than configuration.
type Option func(*cacheOptions)

// WithClock replaces the wall clock the Cache reads fetch instants and expiry
// from. A test in another package that needs to move time or pin a fetch
// instant against a known value passes one.
func WithClock(now func() time.Time) Option {
	return func(o *cacheOptions) { o.now = now }
}

// cacheOptions is what an Option sets. It is not generic, so an Option is one
// type for every Cache rather than one per value type.
type cacheOptions struct {
	now func() time.Time
}

func New[V any](label string, source Source[V], ttl time.Duration, opts ...Option) *Cache[V] {
	o := cacheOptions{now: time.Now}
	for _, opt := range opts {
		opt(&o)
	}
	return &Cache[V]{
		label:        label,
		source:       source,
		ttl:          ttl,
		fetchTimeout: defaultFetchTimeout,
		now:          o.now,
		logf:         log.Printf,
		entries:      make(map[key]*entry[V]),
	}
}

// WeekStatsAsOf returns a week's stats together with the instant the fetch that
// produced them completed. Every caller served from one entry — the caller who
// drove the fetch, a cache hit, a released waiter — gets that entry's recorded
// fetchedAt, so a page served from cache can date its data to the fetch rather
// than to the request.
func (c *Cache[V]) WeekStatsAsOf(ctx context.Context, season, week int) (V, time.Time, error) {
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
			return cached.value, cached.fetchedAt, cached.err
		case <-ctx.Done():
			// A waiter leaving takes only itself: the flight runs on, and the
			// other waiters are still served by it.
			var zero V
			return zero, time.Time{}, ctx.Err()
		}
	}
	if ok && !c.expired(cached) {
		age := c.now().Sub(cached.fetchedAt)
		c.mu.Unlock()
		c.logf("%s hit: %d week %d, %s old", c.label, season, week, age)
		return cached.value, cached.fetchedAt, nil
	}
	flight := &entry[V]{done: make(chan struct{})}
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
	flight.value, flight.err = c.source(fetchCtx, season, week)
	cancel()

	// One line per miss: a miss is one upstream call spent, and misses arrive
	// about once per week per TTL, so the log stays a readable record of the
	// Sleeper budget. Hits are logged too, but cost nothing upstream.
	c.logMiss(season, week, c.now().Sub(start), flight.err)

	// A failure is never stored. Remembering one would turn a momentary
	// upstream blip into a whole TTL of errors on a page that refreshes
	// itself; dropping the entry costs one upstream call per arrival while
	// the outage lasts, which single-flight already bounds.
	c.mu.Lock()
	// An invalidation may have removed this flight and a newer one may hold
	// the key; that newer entry is not this flight's to remove.
	if flight.err != nil && c.entries[k] == flight {
		delete(c.entries, k)
	} else {
		flight.fetchedAt = c.now()
	}
	c.mu.Unlock()
	close(flight.done)

	return flight.value, flight.fetchedAt, flight.err
}

// Invalidate drops a week's entry so its next read fetches afresh. It makes no
// upstream call itself: freshness stays pull-driven.
func (c *Cache[V]) Invalidate(season, week int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key{season: season, week: week})
}

func (c *Cache[V]) logMiss(season, week int, elapsed time.Duration, err error) {
	if err != nil {
		c.logf("%s miss: %d week %d failed after %s: %v", c.label, season, week, elapsed, err)
		return
	}
	c.logf("%s miss: %d week %d fetched in %s", c.label, season, week, elapsed)
}

// expired is checked on read; nothing sweeps. The key space is a handful of
// weeks in a season, so a background goroutine would cost more machinery than
// the memory it reclaims.
//
// An age of exactly the TTL is expired: the requirement is that an entry lives
// *for* the TTL, so the instant it has fully elapsed is already outside.
func (c *Cache[V]) expired(e *entry[V]) bool {
	return c.now().Sub(e.fetchedAt) >= c.ttl
}

// FetchedAt reports the instant the week's held entry finished fetching, and
// false when no completed entry is held: never fetched, invalidated, or still
// in flight. It is a pure lookup — no fetch, no wait, no log line — because it
// is asked far more often than a week is fetched.
//
// The clock is deliberately not consulted: an entry past its TTL still reports
// its instant. Expiry decides when the next read fetches, not whether the last
// fetch happened.
func (c *Cache[V]) FetchedAt(season, week int) (time.Time, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.entries[key{season: season, week: week}]
	if !ok || e.inFlight() {
		return time.Time{}, false
	}
	return e.fetchedAt, true
}
