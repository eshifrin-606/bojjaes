## Context

`internal/sleeper` is the only package that knows Sleeper's shapes (commit f649da7). Its `Client`
is the one entry point; each `*_endpoint.go` holds one endpoint's wire struct and its translation
into a neutral domain type (`score.WeekStats`, `games.Week`); `internal/httpjson` is the shared
timed GET-and-decode. `cmd/server` wraps each `Client` method in a `weekcache.Cache`, and
`internal/web` declares the interface it consumes (`StatsSource`, `GamesSource`). This change adds a
third source in the same shape. The state endpoint differs in two ways: it lives on the documented
v1 host (`api.sleeper.app/v1`), not `api.sleeper.com`, and it takes no season or week.

## Goals / Non-Goals

**Goals:** a provider-neutral current-week value with an explicit "no settable week"; a Sleeper
adapter for it; an hour-long cache built from `weekcache`; log lines matching the existing ones.

**Non-Goals:** any route, page, form, or `cmd/server` wiring; the consumer interface itself (the
lineup form change declares it); adjusting Sleeper's flip day.

## Decisions

### 1. Domain type `nflweek.Current` in new package `internal/nflweek` — mirrors `internal/games`

```go
type Phase int // Preseason, Regular, Postseason, Offseason, Other
type Current struct{ season, week int; phase Phase }
func New(season, week int, phase Phase) Current
func (c Current) Settable() (season, week int, ok bool)
```

Unexported fields and one constructor, like `games.Week`; `Other` carries an unrecognised phase
like `games.Other`. `Settable` is the only question callers ask, so "no settable week" is the
`ok == false` branch, never a zero week a caller might use by mistake. The package describes the NFL
calendar only: the league's last week (17) is a league rule, so it does not live here and is left
to the lineup form change. A new package rather than `games`,
because `games` is a week's games keyed by season/week and this is a point in the calendar.

### 2. Sleeper adapter `Client.CurrentWeek(ctx) (nflweek.Current, error)` — mirrors `WeekGames`

`state_endpoint.go` holds `stateRow` (`season` string, `season_type`, `week`), `fetchState`, and
the phase mapping (`pre`/`regular`/`post`/`off`, else `Other`), shaped like `scores_endpoint.go`'s
`gameRow`/`gameState`. It calls `httpjson.Get`, so the 15s client and error wording are shared.
The method lives in `client.go` beside the other two, times the call, and logs through `c.logf`.
The season-as-string quirk and the `season_type` strings never leave the package.

The v1 host is a second constant `V1BaseURL = "https://api.sleeper.app/v1"` and a matching
`Client.V1BaseURL` field, so tests point it at an `httptest.Server` exactly as they do `BaseURL`.
Rejected: reusing `BaseURL` with a path (`api.sleeper.com/state/nfl` is undocumented; the decision
names the v1 URL) and hardcoding the URL in the endpoint (untestable).

### 3. Cache: `weekcache.Single[V]` over the existing `Cache[V]` — reuses `weekcache`

```go
func NewSingle[V any](label string, source func(context.Context) (V, error), ttl time.Duration, opts ...Option) *Single[V]
func (s *Single[V]) AsOf(ctx context.Context) (V, time.Time, error)
```

`Single` holds a `*Cache[V]` whose source ignores the key and always reads one fixed key. So
single-flight, detached fetch with its own timeout, failures not stored, `fetchedAt`, and
`WithClock` are inherited, not rewritten. `AsOf` returns the fetch instant like `Cache.AsOf`, so a
future consumer interface has the same shape as `StatsSource`/`GamesSource`.

The only change to `Cache` is an unexported key formatter used by the hit/miss lines: keyed caches
keep `2026 week 5`; `Single` sets `current`. Without it every state line would read `0 week 0`.
Rejected: a hand-rolled `sync.Mutex` + timestamp in the consumer (a second cache with weaker
failure and cancellation behaviour), and making `Cache` generic over its key (larger change for
one caller).

### 4. TTL `weekcache.CurrentWeekTTL = time.Hour`

The value changes once a week, overnight. An hour bounds lateness after the flip to an hour nobody
is setting a lineup in, and caps the endpoint at 24 calls a day. `weekcache.TTL` (5 min) exists for
live scores, a different freshness need. Named beside `TTL` so main names it, not a literal.

### 5. Consumer side (for the lineup form change)

`internal/web` will declare its own interface, e.g.
`CurrentWeekSource interface { AsOf(ctx) (nflweek.Current, time.Time, error) }`, and `cmd/server`
will pass `weekcache.NewSingle("nflstate", client.CurrentWeek, weekcache.CurrentWeekTTL)`. Nothing
here names that interface, as `Client` names none today.

## Risks / Trade-offs

- [Sleeper flips the week mid-week, not Tuesday] → accepted by decision; the form sets whatever
  week Sleeper reports.
- [League season ends at NFL week 17, but week 18 is settable here] → the lineup form applies the
  league's last week; this package does not know it.
- [Fixture shape guessed] → task 3.1 saves a real response from the endpoint as the test fixture.
- [`api.sleeper.app` down while `api.sleeper.com` is up] → the read fails as an error and is not
  cached; the next request retries.
