## Context

`cmd/server` constructs one `sleeper.Client` and hands it to `api.BatchHandler` and `web.Handler`,
each of which declares its own one-method `StatsSource` interface. Every request therefore costs one
call to Sleeper's weekly aggregate — roughly half a megabyte, 200 ms – 1.2 s
([ADR 0003](../../../docs/adr/0003-sleeper-as-initial-stat-provider.md)) — regardless of how recently
the same season and week were fetched.

`refresh-page-while-visible` made that worse on purpose: an open tab now re-requests itself every
five minutes, so upstream volume scales with readers. Its risk register named this change as the
answer and said "if we deploy publicly before the cache exists, the cache goes first."

[ADR 0004](../../../docs/adr/0004-web-frontend-stack.md) decision #5 fixes the shape: a short TTL
cache over the fetch, guarded by single-flight, with no scheduler and no background poller. The
package map already predicts the mechanism — "a caching `StatsSource` that wraps another one,
constructed in `main`, with no consumer edited."

Two properties of the domain shape the design more than the caching does:

- **The cache key is the whole result.** One fetch returns every player in the league, so there is
  no partial hit, no per-player entry, and no merge. A key is `(season, week)` and a value is one
  `score.WeekStats`.
- **A `WeekStats` is already immutable by contract.** `NewWeekStats`'s doc says the caller must not
  retain the map because a `WeekStats` is read concurrently. Sharing one value across many readers
  is what it was built for, so the cache needs no copying.

## Goals / Non-Goals

**Goals:**

- A second request for the same season and week within the TTL makes no upstream call.
- Concurrent misses on the same key make exactly one upstream call, and every caller gets its
  result.
- A failed fetch is not remembered, so recovery is immediate.
- The wrapper is a `StatsSource` and nothing else, so `main` is the only file that changes.
- The cached entry records when its fetch completed, so the as-of timestamp change can read it later
  without redesigning this.

**Non-Goals:**

- Any fetch the cache initiates itself: no poller, no scheduler, no refresh-ahead, no warming.
- Persistence, sharing across processes or machines, or surviving a Fly auto-stop.
- Serving a stale entry when a refresh fails.
- Caching errors, tuning TTL per key, eviction beyond expiry, metrics, or instrumentation.
- Rendering the as-of timestamp. This change stores the time; it displays nothing.

## Decisions

### A decorator in a new `internal/statscache` package

`statscache.New(source, ttl)` returns a value whose `WeekStats(ctx, season, week)` satisfies the
same one-method interface `internal/api` and `internal/web` already declare. It names no provider —
it declares the `StatsSource` interface it wraps, exactly as the two transports do — so the
dependency direction in `docs/package-dependencies.md` is unchanged and `internal/sleeper` learns
nothing.

Rejected: caching inside `sleeper.Client`. It would make the adapter responsible for a policy that
has nothing to do with Sleeper, and would mean a second provider had to reimplement it. Also
rejected: caching in each handler, which would give the page and the scripts separate caches and
separate upstream budgets for the same data.

`main` wraps once and passes the same value to both handlers. One cache, one budget — a
`scripts/scores.sh` run during a Sunday warms the page and vice versa.

### Single-flight written here, not `golang.org/x/sync/singleflight`

The mechanism is a per-key entry holding a `done chan struct{}`: the first caller for a cold key
installs an entry under the map mutex and fetches; later callers find it, release the mutex, and
wait on the channel; the leader stores the result and closes the channel. It is about thirty lines
and every branch is drivable by a test.

`x/sync/singleflight` is the obvious alternative and does the same job well. It is rejected here
because it would be the project's first non-stdlib dependency, taken to save thirty lines, and its
`Do` takes no context — the detachment decision below would have to be built around it anyway. If
the shape ever grows (`Forget`, duplicate-call accounting), swapping it in is contained.

The mutex is held only for map lookups and the entry install, never across the fetch. A slow
upstream must not block a request for a different week.

### The upstream fetch runs on a context detached from the caller's

The leader fetches under `context.WithoutCancel(ctx)` with its own timeout, not the request context
it happened to arrive on. Otherwise the first reader closing their tab mid-fetch cancels the call
that four other readers are waiting on — a failure that gets more likely the better single-flight
works.

Waiters still respect their own context: a caller select-ing on `ctx.Done()` and the entry's `done`
channel returns its own cancellation without disturbing the flight. The leader is the only one who
cannot walk away, and its exposure is bounded by the timeout.

The timeout is the cache's, not borrowed from `sleeper`'s 15 s client timeout, because the wrapper
must not assume what it wraps. It is set generously — a fetch that outlives it fails the flight, and
the next request retries.

### Time is injected, not read from `time.Now` directly

The cache holds a `now func() time.Time`, defaulting to `time.Now`. Expiry is then a test that sets
a clock forward rather than a test that sleeps: a TTL boundary is exactly the kind of thing worth
asserting on both sides of, and a five-minute TTL cannot be waited out.

Rejected: a real clock plus a one-millisecond TTL in tests. It tests the TTL comparison but makes
the tests timing-dependent, and it cannot express "1 ns before expiry".

Expiry is checked on read. There is no sweeper goroutine: the key space is a handful of weeks in a
season, entries are half a megabyte, and a background goroutine would be more machinery than the
memory it reclaims. This is worth revisiting only if the cache is ever keyed by something unbounded.

### Errors are never stored

A failed fetch clears the entry before the leader closes the channel, so the flight's waiters all
see the error and the next arrival starts a fresh flight. Caching a failure would turn a one-second
upstream blip into five minutes of `502` on a page that refreshes itself — the "sharpest edge" the
refresh design already flagged, made permanent.

The trade-off is deliberate and named under Risks: a persistently failing upstream gets a request
per arrival rather than one per TTL. Single-flight bounds the concurrent case, which is the one that
matters at three viewers.

### The entry records its fetch time; nothing reads it yet

Each entry stores when its fetch completed. Nothing exposes it in this change — the interface still
returns `(score.WeekStats, error)`. It is stored because the as-of timestamp is the next freshness
line, and with a cache in front of the fetch the honest as-of is the cached fetch time, not the
request time. Storing it now means that change is an accessor, not a rework.

Deliberately not done here: widening `StatsSource` to return a timestamp. That would touch both
transports for a value neither renders yet.

## Risks / Trade-offs

**A cached page can be up to a TTL stale with nothing saying so** → The reader cannot tell a
five-minute-old score from a current one, and the refresh timer now sometimes re-renders identical
numbers. This is the exact gap the as-of timestamp closes, and this change is what makes it worth
closing: today's page is at least always fresh at load time. Mitigation is the next backlog line,
not this one; the fetch time is stored ready for it.

**A persistently failing upstream is retried per request, not per TTL** → Not caching errors means
an outage costs one upstream call per arrival. At three viewers refreshing every five minutes that
is negligible, and single-flight collapses any simultaneity. Accepted as strictly better than
pinning a recovered outage for five minutes.

**In-process cache dies with the machine** → ADR 0004 already accepts this: Fly auto-stop discards
it and the first load after idle pays a full fetch. Continuous Sunday polling keeps it warm, which
is the case that matters. Fixes (a minimum running machine, an out-of-process cache) are additive.

**Memory grows with distinct weeks viewed** → Each entry is a full league week. Browsing back
through a season holds them all until they expire, since nothing sweeps. Bounded in practice by
`score.ValidateSeasonWeek` and by there being one page anyone actually opens; called out so a future
key with an unbounded domain does not inherit this quietly.

**A detached fetch outlives the request that started it** → By design, and bounded by the cache's
timeout. The cost is that a shutdown can be waiting on a fetch nobody wants. Irrelevant until the
graceful-shutdown backlog line lands, and worth remembering there.

**Tests now depend on an injected clock** → A clock seam is a place production and test behaviour
can diverge. Kept as narrow as possible: one function field, used only for expiry comparison, with
the real `time.Now` as the default so the wiring in `main` names no clock at all.
