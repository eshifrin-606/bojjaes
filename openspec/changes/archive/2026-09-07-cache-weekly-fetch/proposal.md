## Why

Every request for a matchup page is a Sleeper fetch. Now that the page refreshes itself every five
minutes, three open tabs are three upstream calls every five minutes, and the number scales with
readers rather than with time. Against an undocumented API whose rate-limit tolerance is still
unprobed ([ADR 0003](../../../docs/adr/0003-sleeper-as-initial-stat-provider.md)), that is the wrong
shape.

[ADR 0004](../../../docs/adr/0004-web-frontend-stack.md) decision #5 settled the fix — "the server
holds a short TTL cache (~5 min) over the Sleeper fetch, guarded by single-flight so concurrent
misses collapse into one upstream call" — and its consequences call this "a correctness requirement
of going public, not an optimisation". The client half landed in `refresh-page-while-visible`; this
is the server half, and it is the piece that has to exist before the scoreboard is deployed
anywhere reachable.

## What Changes

- A caching `StatsSource` decorator: it wraps another `StatsSource`, holds each `(season, week)`
  result for approximately five minutes, and serves hits without touching the upstream.
- Single-flight per `(season, week)`: concurrent misses for the same key make exactly one upstream
  call and all share its result. Three readers reloading in the same second cost one fetch.
- Errors are not cached. A failed fetch leaves the key empty so the next request retries, rather
  than pinning a five-minute outage onto a provider that has already recovered.
- `cmd/server/main.go` wraps the `sleeper.Client` in the cache once and hands the same wrapped
  source to both `api.BatchHandler` and `web.Handler`, so the page and the scripts share one cache
  and one upstream budget.
- Upstream request volume becomes bounded by the TTL and the number of distinct weeks being viewed,
  not by viewer count, and drops to zero when nobody is looking.

Out of scope:

- **A background poller, a scheduler, or any refresh-ahead.** Freshness stays pull-driven, per ADR
  0004 decision #5. The cache never fetches on its own.
- **Cross-process or persistent caching.** The cache is in-process and dies with the machine; ADR
  0004 already accepts that a Fly auto-stop discards it and names moving it out of process as an
  additive follow-up.
- **The as-of timestamp.** It is the other open freshness line and it interacts with this change —
  a cached page's as-of is the *fetch* time, not the request time — but it changes what
  `matchup-page` renders and is its own backlog line. This change must not make that harder: the
  cached entry carries the time its fetch completed so the timestamp change has something honest to
  read.
- **Probing Sleeper's rate limits.** Still open from ADR 0003, and still a separate line.
- Serving stale data on upstream failure, negative caching, per-entry TTL tuning, cache metrics or
  an eviction policy beyond expiry.

## Capabilities

### New Capabilities

- `weekly-stats-cache`: the caching layer over a weekly stat fetch — what a hit is, how long an
  entry lives, that concurrent misses for the same season and week collapse into one upstream call,
  that failures are not cached, and that the cache never fetches on its own.

### Modified Capabilities

None. `matchup-page`'s "the stat provider is called once for that season and week" is a statement
about the handler calling its `StatsSource` once per request, which is exactly as true with a cache
behind it; the page's observable behaviour is unchanged except that its data may be up to a TTL old,
which no current requirement contradicts.

## Impact

- New `internal/statscache` package (name settled in design): the decorator, its TTL, its
  single-flight. It depends on `internal/score` and declares the one-method `StatsSource` interface
  it wraps, so no provider package is named and the dependency direction in
  `docs/package-dependencies.md` is unchanged.
- `cmd/server/main.go`: one wrap at the composition root. No handler is edited — the map already
  predicts this ("a TTL cache over the weekly fetch is additive: a caching `StatsSource` that wraps
  another one, constructed in `main`, with no consumer edited").
- `docs/package-dependencies.md`: a row and an arrow for the new package.
- No change to `internal/sleeper`, `internal/web`, `internal/api`, `internal/score`,
  `internal/roster`, the roster CSV format, `scripts/**`, or any HTTP surface.
- `go.mod` may gain `golang.org/x/sync` if `singleflight` is used rather than written; the design
  decides. This would be the project's first non-stdlib dependency.
