## Why

The matchup page shows no indication of how old its scores are. Before the TTL cache that was
tolerable — a page was fetched when it was requested, so "now" was an honest enough answer to *when
is this from*. `weekly-stats-cache` removed that: a reader can now be served data up to five minutes
old, and `refresh-page-while-visible` makes a quietly stale tab the common case rather than the
edge one. The cache already records when each fetch completed and renders nothing from it; this is
the change that puts it on the page.

The backlog frames it deliberately narrow: stamp the as-of from *our own Sleeper fetch time* for
now, labelled as such. That instant is when the server pulled the data, which can lag when the
stats actually moved upstream. Measuring that lag is a live-Sunday observation, not code, and it is
recorded as deferred verification rather than attempted here.

## What Changes

- `matchup-page` gains a requirement: the rendered page states the instant its stats were fetched
  from the provider, as a machine-readable timestamp with a human-readable rendering, labelled so a
  reader understands it as *when we fetched*, not *when the game data last changed* or *now*.
- The as-of instant is the cache's recorded fetch time. On a cache hit the page reports the
  original fetch, not the time of the request that was served from cache.
- `web.StatsSource` changes shape to `WeekStatsAsOf(ctx, season, week) (score.WeekStats, time.Time,
  error)`. `statscache.Cache` implements it and keeps `WeekStats` for `internal/api`, which is
  unchanged.
- The human rendering is an absolute time in `America/Chicago`, the league's timezone, inside a
  `<time datetime="…">` element carrying the RFC 3339 instant. No relative ("4 minutes ago")
  formatting and no new client-side JavaScript in this change.
- The page still implies no winner: the timestamp is one line above or below both columns, not per
  column, and says nothing about either total.

Out of scope:

- **Relative-time rendering and any client JavaScript.** The `<time datetime>` attribute is left
  machine-readable so a later change can layer "4m ago" onto it without touching the handler. This
  change renders an absolute string and stops.
- **The live-Sunday drift measurement.** Comparing the page's as-of against when a stat visibly
  moved on Sleeper during games is an observation the maintainer runs; it is recorded in `notes.md`
  as deferred verification, alongside the still-open rate-limit probe it feeds.
- **`internal/api` / the JSON endpoint.** `POST /scores` does not report freshness and is not
  touched. If a JSON consumer ever wants the fetch time it gets its own accessor.
- **`internal/sleeper`.** It has no cache and no meaningful stored fetch time; widening the shared
  interface to make it invent one was considered and rejected in `weekly-stats-cache`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `matchup-page`: adds a requirement that the page states the fetch instant of its stats, and that
  the instant is the fetch time rather than the request time. The existing "implies no winner"
  requirement is unchanged but the new element is constrained by it.
- `weekly-stats-cache`: the requirement that an entry records its fetch time is extended — that
  time is now retrievable by a caller alongside the stats, not only stored. The caching behaviour
  itself (TTL, single-flight, error handling, no self-fetching) is unchanged.

## Impact

- `internal/web/matchup.go`: the consumer `StatsSource` interface changes to `WeekStatsAsOf`; the
  handler threads the returned `time.Time` into the view model; a new field renders through the
  template.
- `internal/web/matchup.html`: a `<time>` element and its label.
- `internal/statscache/cache.go`: a `WeekStatsAsOf` method returning `(score.WeekStats, time.Time,
  error)`; `WeekStats` becomes a thin delegate for `internal/api`. All three internal return paths
  (fresh miss, cache hit, in-flight waiter) surface `entry.fetchedAt`.
- `internal/statscache` compile-time interface assertions gain the `web` shape.
- Test doubles: `internal/web`'s fake source grows the `time.Time` return.
- `docs/package-dependencies.md` if it names the `web`→`statscache` interface shape.
- No change to `internal/api`, `internal/sleeper`, `internal/score`, `internal/roster`,
  `cmd/server/main.go` wiring (the same wrapped value satisfies both interfaces), the roster CSV
  format, `scripts/**`, or any HTTP path or status code.
