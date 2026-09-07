## Context

`weekly-stats-cache` put a five-minute TTL in front of the Sleeper fetch and, on the way, stored
`entry.fetchedAt` — the instant the upstream call completed — expressly so this change would be "an
accessor, not a rework." That change renders nothing from it. `refresh-page-while-visible` reloads
an open tab every five minutes, so the common case is now a page whose numbers are a few minutes
old with nothing saying so.

The wiring: `cmd/server` builds one `sleeper.Client`, wraps it once in `statscache.New`, and hands
the same value to `web.Handler` and `api.BatchHandler`. Each consumer declares its own one-method
`StatsSource` interface (`WeekStats(ctx, season, week) (score.WeekStats, error)`); the cache
satisfies both by structural typing.

`weekly-stats-cache`'s design explicitly rejected widening that shared interface "for a value
neither renders yet," and noted `sleeper` has no honest fetch time of its own to return — an
un-cached fetch's "as of" is just the request time. That constraint still holds: only the page
wants the timestamp, and only the cache has an honest one.

## Goals / Non-Goals

**Goals:**

- The matchup page states the instant its stats were fetched, as machine-readable RFC 3339 plus a
  human-readable `America/Chicago` wall-clock time.
- On a cache hit the page reports the original fetch instant, not the request it was served to.
- Only `internal/web` and `internal/statscache` change. `internal/api`, `internal/sleeper`, and the
  `cmd/server` wiring are untouched.
- The `<time datetime>` value is left machine-readable so relative formatting can be added later
  without touching Go.

**Non-Goals:**

- Relative-time rendering ("4m ago") and any client-side JavaScript.
- Reporting freshness on `POST /scores` or anywhere in `internal/api`.
- Making `internal/sleeper` invent a fetch time.
- Measuring how far the fetch instant lags the real upstream stat movement — a live-Sunday
  observation, recorded in `notes.md` as deferred verification.

## Decisions

### `web`'s own `StatsSource` becomes the timed interface (Option C)

`internal/web` changes its consumer interface to:

```go
type StatsSource interface {
    WeekStatsAsOf(ctx context.Context, season, week int) (score.WeekStats, time.Time, error)
}
```

`statscache.Cache` grows a `WeekStatsAsOf` method returning `(score.WeekStats, time.Time, error)`,
and `WeekStats` becomes a thin delegate (`stats, _, err := c.WeekStatsAsOf(...)`) that
`internal/api` keeps calling unchanged. The `cmd/server` wiring does not change: the same wrapped
value satisfies `api`'s `WeekStats` interface and `web`'s `WeekStatsAsOf` interface at once.

**Alternatives considered:**

- **Widen the shared interface** — one `WeekStatsAsOf` on every `StatsSource`, including
  `sleeper`'s. Rejected: it forces `sleeper` to return a dishonest fetch time (really the return
  time), threads a value `internal/api` never renders through that whole package, and touches every
  test fake in three packages. This is the option `weekly-stats-cache` already declined.
- **Optional interface, type-asserted in the handler** — keep the shared `WeekStats`, and in the
  handler do `if s, ok := source.(interface{ WeekStatsAsOf(...) }); ok`. Rejected: there is no real
  "web page with no as-of line" case, so the `else` branch is dead code whose only effect is to
  turn a wiring mistake into a silently missing timestamp instead of a compile error. Option C
  costs the same (two methods on the cache) and states the requirement in the type.

### `WeekStatsAsOf` surfaces `entry.fetchedAt` on all three return paths

`Cache.WeekStats` has three ways out: a fresh miss the caller drove, a cache hit, and a waiter
released from an in-flight fetch. `WeekStatsAsOf` returns `entry.fetchedAt` on each. The field is
already populated under `c.mu` before `close(flight.done)` on the success path, so a released
waiter reads a stable value; the hit path reads the stored entry; the miss path reads the entry it
just finalised. `WeekStats` delegates and discards the time, so single-flight, TTL, error handling,
and "never fetches on its own" are literally the same code.

### Absolute time in `America/Chicago`, rendered server-side into a `<time>` element

The handler loads `America/Chicago` (via `time.LoadLocation`, once at package init like the
template parse — a missing zone database is a startup failure, not a per-request one) and formats
`fetchedAt.In(loc)`. The template emits:

```html
<p class="as-of">Sleeper stats fetched <time datetime="{{.FetchedAtRFC3339}}">{{.FetchedAtText}}</time></p>
```

Both strings are computed in Go, where they are testable, matching how `starter.Points` is already
formatted in Go rather than the template. `FetchedAtRFC3339` is `fetchedAt.Format(time.RFC3339)`
(UTC instant preserved); `FetchedAtText` is the Chicago wall-clock rendering, e.g.
`Sun, Sep 7 2026 1:24 PM CDT` — the zone abbreviation is kept so the reader knows which clock.

**Alternatives considered:**

- **Relative time server-side** ("4 minutes ago") — goes stale between the 5-minute client
  reloads just as silently as an absolute time does, and it is a one-way computation the reader
  cannot sanity-check against their own clock. Deferred to a JS change that reads the `datetime`
  attribute.
- **UTC on the page** — correct but unreadable for a league that thinks in Central. The RFC 3339
  attribute already carries the unambiguous instant for machines.

### The as-of line sits outside both columns and names no total

`matchup-page`'s "implies no winner" requirement covers the new element: it renders once, above or
below `<main class="matchup">`, references neither `column`, and gets no styling keyed on which
total is higher. It states a fetch time and nothing about the game.

## Risks / Trade-offs

**The stamp is the fetch time, not when the stats moved upstream** → By design, and the label says
"fetched" for exactly this reason. Sleeper can update a stat line minutes after the play; our fetch
sees it only on the next miss, up to a TTL later. How large that lag actually runs is the open
question `notes.md` parks for a live Sunday, and it feeds the still-open rate-limit / TTL probe
rather than blocking this change.

**`statscache.Cache` now has two read methods that look almost identical** → `WeekStats` is four
lines delegating to `WeekStatsAsOf`. A regression test pins that `WeekStats` still returns
`(stats, err)` unchanged so the delegation cannot rot.

**A future un-cached source behind `web` would not compile** → Correct behaviour, not a risk: an
un-cached page has no honest as-of, so requiring the timed interface at the boundary is the point.
If that case ever arises it needs a deliberate adapter, not a silent fallback.

**`time.LoadLocation("America/Chicago")` needs a zone database** → The Go binary can embed one with
`import _ "time/tzdata"` if the deploy image lacks `/usr/share/zoneinfo`. Loading at package init
surfaces a missing zone as a boot failure with a clear message rather than a per-request 500. The
Dockerfile is a later backlog line; this change adds the `tzdata` import so the binary is
self-sufficient regardless.

## Open Questions

- Exact wording and date format of `FetchedAtText`. The spec constrains it (labelled as fetched,
  not "live", `America/Chicago`, zone shown); the precise string is settled in the red-green loop
  against a readable assertion.
