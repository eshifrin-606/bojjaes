## Context

`PlayStore.ForcedFumbles` waits at most `PLAYS_COLD_WAIT` (500ms in production) for a cold week's
whole-week fetch. When that fetch lands later, `main.go` calls `stats.Invalidate`, so the next read is
fresh. The browser makes that next read only when its five-minute timer fires or the reader
refreshes.

```
t=0      GET /2026/5 (cold)   poll 300 plays, whole-week fetch starts
t=0.5s   cold wait ends       page renders, entry cached at instant T
t=~3s    whole-week lands     merge, Invalidate(2026, 5)      <- browser is not told
t=5m     timer                reload
```

Constraints that shape the design:

- The `matchup-page` spec requires the refresh to be a reload of the same URL, with no fragment
  fetch, no markup swap and no persistent connection.
- The refresh script may contain no interpolated page value.
- `weekly-stats-cache` is pull-driven: it never fetches on its own.
- `web` names no provider package; it declares the interfaces it consumes and `main` wires them.
- Fly stops the machine when idle (`auto_stop_machines = 'stop'`, `min_machines_running = 0`), and
  both the cache and the play store live in memory.

## Goals / Non-Goals

**Goals:**

- An open, visible page reloads within about one check interval of its week's stats being superseded.
- The mechanism is general: it reacts to any invalidation, not to the whole-week fetch specifically.
- The server decides the check cadence, so it can vary later without a script change.
- Checks cost nothing upstream.

**Non-Goals:**

- Fresher live scores. The server still fetches only when a page asks; that cadence is the
  five-minute interval and the cache TTL, both unchanged.
- Varying the cadence by week, by game state or by time of day.
- Time caps, idle caps or backoff on checking.
- Server-sent events or WebSockets.
- Patching scores into the page in place.
- Fixing the cold start after Fly stops the machine.

## Decisions

### D1. Poll a version endpoint; do not push

The page asks `GET /{season}/{week}/version` on a delay and reloads on a mismatch.

- **Why:** each check is an ordinary short request, so the 30s `writeTimeout` and 60s `idleTimeout`
  stand, there is no subscriber registry, and a missed check is repaired by the next one.
- **Alternative, server-sent events:** instant, but needs a per-route write deadline, a concurrent
  registry, a catch-up comparison on reconnect anyway, and reverses the spec's ban on a persistent
  connection.
- **Alternative, one-shot long-poll for the in-flight whole-week fetch:** fewer requests, but handles
  only that one cause.

### D2. The version is the fetch instant as an opaque token

The token is the cache entry's `fetchedAt` as Unix nanoseconds in decimal. `web` formats it in one
function used by both the page handler and the version handler.

- **Why not reuse the `<time datetime>` value:** it is RFC 3339 at one-second granularity. Two
  fetches in the same second would compare equal and the page would miss the change; a server
  comparing a truncated instant against a nanosecond one would report a change forever and the page
  would reload on every check.
- **Why opaque:** the script compares strings and never parses. The token can later fold in other
  inputs, such as a lineup file's modification time, without touching the script.

### D3. Compare in the browser, not on the server

The endpoint reports the current version; the script compares it with its own.

- **Why:** the endpoint takes no input beyond the path, so it cannot disagree with the page about
  formatting, and one response serves every viewer of the week.
- **Alternative, `?since=<version>` answered with changed or not:** puts page-supplied input into the
  handler for no gain.

### D4. "No completed entry" reports an empty version, which is a mismatch

A week never fetched, invalidated, in flight, or lost to a restart all report `""`. The page treats
any difference as superseded and reloads.

- **Why:** each of those means the page's stats are no longer what the server would serve. The reload
  either starts the fresh fetch or waits on the one in flight.
- **Consequence:** returning to a tab after Fly stopped the machine reloads the page at the first
  check. That reload is a cold read, as a manual refresh would be.

### D5. An expired entry is not a change

`Cache.FetchedAt` reports an entry's instant even after the TTL has passed, so an entry merely ageing
does not trigger a reload. The five-minute timer remains the only thing that refreshes on age.

- **Why:** it keeps the two mechanisms independent. "Superseded" comes from the check; "old" comes
  from the timer. The cache method stays a pure lookup with no clock involved.
- **Alternative, treat expiry as changed:** reloads would align with the TTL and the timer would be
  redundant. Worth revisiting if the timer is ever removed; not needed now.

### D6. Cadence travels as data, with one default

- The page render puts the first delay in an attribute; each response carries the next delay.
- `0` means stop checking until the next load.
- One Go constant in `web`, about ten seconds, feeds both.

The script uses a chained `setTimeout`, not `setInterval`, so each delay is the one the server last
gave. If a check fails, it reuses the previous delay.

### D7. Where the values live on the page

Both attributes go on the existing as-of paragraph, next to the `<time>` they relate to:

```html
<p class="as-of" data-version="1760212345123456789" data-next-check="10">
```

The script reads them with `dataset` and builds the check URL from `location.pathname`. Nothing is
interpolated into the script, so "The refresh script carries no page data" holds unmodified.

### D8. Response shape

```json
{"version": "1760212345123456789", "nextCheckSeconds": 10}
```

`Content-Type: application/json`, `Cache-Control: no-store`. JSON over a bare body because there are
two fields today and cadence is expected to grow.

### D9. The hidden-tab guard matches the existing timer

When a check comes due and the tab is hidden, the script makes no request and schedules the next
check at the same delay. It does not hook `visibilitychange`.

- **Why:** simplest form that keeps hidden tabs silent; a returning reader waits at most one delay.
- **Alternative, check at once on return:** faster by a few seconds, at the cost of a second code
  path. Easy to add later.

### D10. Wiring

- `statscache.Cache` gains `FetchedAt(season, week int) (time.Time, bool)`: a lookup under the mutex,
  `false` when absent or in flight, no logging, no clock.
- `web` declares the small interface it needs for that and exports the version handler.
- `newMux` registers `GET /{season}/{week}/version`; `main` passes the same `stats` cache for both.
- The version handler validates season and week as the page does and never opens the lineup tree.

## Risks / Trade-offs

- **A visible tab keeps the Fly machine running** → accepted; the ceiling is about $2 a month.
  Server-directed cadence (D6) is the lever if this ever matters.
- **Reload loop if page and server never agree on a version** → the shared format function (D2) and
  browser-side comparison (D3) remove the known causes. Worst case is one reload per delay per
  visible tab, each a cache miss, which single-flight collapses to one upstream call.
- **A check finds a fetch in flight and reloads into it** → the reload waits on that flight like any
  reader; if the fetch fails the reader gets the 502 page, as the timed reload does today.
- **The script is tested by reading its text, not by running it** → add one headless-Chrome test that
  drives a real reload, under the same skip rules as the layout tests, plus a manual check.
- **Request volume in Fly's logs** → one line per visible tab per delay from the proxy. The cache
  writes nothing for these (see the cache delta spec).
- **Fly's default concurrency limits** → checks are short, so they do not hold connections the way a
  stream would. Not expected to matter at league scale.

## Migration Plan

Deploy as usual. A page loaded before the deploy has no check script and keeps its five-minute
timer until it next reloads. Rollback is a redeploy of the previous image; a page loaded from the new
build would then get 404s from the version endpoint, which the script treats as a failed check and
ignores.

## Open Questions

- Is ten seconds the right default? It bounds how late the corrected page appears.
- Should a returning tab check immediately (D9 alternative) once this has been used in a real game?
- Does Fly stop the machine inside the five-minute timer interval for a lone viewer whose tab is
  hidden? If so, every return is a cold read; that is a separate change.
