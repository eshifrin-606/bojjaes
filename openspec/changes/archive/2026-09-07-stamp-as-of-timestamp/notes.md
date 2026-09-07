# Notes

## Why the stamp is the fetch time, not when the stats moved

The page says `Sleeper stats fetched …`, and "fetched" is load-bearing. The instant is
`entry.fetchedAt` — when our upstream call to Sleeper completed — not when the underlying game data
last changed. Those are different by an unknown amount:

- Sleeper can revise a stat line minutes after the play (see
  `docs/basic-memory/2026-08-15-sleeper-weekly-stats-absence-is-routine-ambiguous-and-sometimes-stale.md`).
- Our fetch sees a revision only on the next cache miss for that week — up to a TTL (5 min) after it
  landed upstream, longer if nobody asks.

So the honest claim the server can make is "this is the data as we last pulled it", and that is
exactly what the label says. Claiming freshness of the *game* would be a claim we cannot support.
The RFC 3339 `datetime` attribute carries the same instant for machines; a later change can layer a
relative "4 min ago" onto it without touching the handler.

## Deferred: the live-Sunday drift observation

Not code. The maintainer runs it once, during games, and records the result.

**Procedure.** With the page open on a real Sunday slate, pick a player whose line is about to
change (a TD in progress, a stat correction). Note the wall-clock time the change is visible on
Sleeper's own app/site. Reload our page until the new value appears; note our `<time>` value and
the real time of that reload. The gaps to watch:

- **visible-on-Sleeper → present in our fetch**: bounded above by the TTL plus fetch latency, but
  the typical value is the interesting one.
- **our fetch instant → the reload that showed it**: how stale the page a reader is looking at can
  be, in practice, between the 5-minute client refreshes.

**What it feeds.** The still-open backlog line "Probe Sleeper's rate-limit tolerance at the cadence
we're actually going to deploy at." If drift is routinely small, the 5-minute TTL is more
conservative than it needs to be and could be shortened; if it is large and dominated by Sleeper's
own edge caching, shortening the TTL buys nothing and the probe should look at concurrency instead.
The TTL was chosen for acceptable staleness of a live week, not from measurement — this is the
measurement.

## Layout decision the spec left open (task 4.2)

The spec fixed the constraints — labelled as *fetched*, `America/Chicago`, zone abbreviation shown,
not "live"/"current", one element outside both columns, no relative phrasing — and left the exact
string to the red-green loop. Settled:

- **Visible text layout:** `Mon, Jan 2 2006 3:04 PM MST` → e.g. `Mon, Sep 7 2026 1:24 PM CDT`.
  Weekday is kept because the reader is orienting a stale page against "now"; the zone abbreviation
  (`CDT`/`CST`) is kept so there is no doubt which clock, and it falls out of the DST-aware
  `America/Chicago` location automatically.
- **`datetime` attribute:** `time.RFC3339` on the **original** instant, not `.In(chicagoLoc)`. Both
  denote the same moment, but formatting the original keeps the offset the handler actually holds
  and avoids implying the attribute is a Chicago-local value. `TestTheDatetimeAttributeIsAnUnambiguousInstant`
  pins that it round-trips through `time.Parse`.
- **Markup:** `<p class="as-of">Sleeper stats fetched <time datetime="…">…</time></p>`, a sibling of
  `<main class="matchup">` placed after it. Both strings are computed in Go (`FetchedAtRFC3339`,
  `FetchedAtText`), matching how `starter.Points` is already formatted in Go rather than the
  template.
- **Zone database:** `import _ "time/tzdata"` in `internal/web`, so the binary carries its own copy
  and `America/Chicago` loads the same whether or not the deploy image ships `/usr/share/zoneinfo`.
  The location loads once at package init next to the template parse — a missing zone is a boot
  failure with a clear message, not a per-request 500.

## Test seam added

`statscache.New` gained a variadic `Option` and `statscache.WithClock`, so `internal/web`'s
end-to-end test can drive a real `Cache` past a three-minute gap and see that a served cache hit
still dates to the fetch. `main` passes no options; the clock stays a seam, not configuration. The
in-package `newTestCache` still sets `cache.now` directly and is unchanged.

## Scenario coverage

Third column is the strength of the claim: a test that failed for the behaviour it names is
stronger evidence than one written green. **Genuine red** — failed against the previous step for
the predicted reason. **Verified red** — passed on first run, so production code was deliberately
broken, the failure observed, and the code restored.

| Requirement | Scenario | Test | Claim |
| --- | --- | --- | --- |
| Cache exposes the fetch time | Miss returns the fetch instant | `TestWeekStatsAsOfOnAMissReturnsTheFetchInstant` | Genuine red (2.1: zero instant) |
| Cache exposes the fetch time | A hit reports the fetch, not the read | `TestWeekStatsAsOfOnAHitReturnsTheOriginalFetchInstant` | Genuine red (2.3: zero instant), then verified against `c.now()` |
| Cache exposes the fetch time | Every waiter on one flight gets the same instant | `TestWeekStatsAsOfGivesEveryWaiterTheSameFetchInstant` | Genuine red (2.4: waiters got zero) |
| Stats-only path unaffected | `WeekStats` still `(stats, err)`, same caching | `TestWeekStatsStillDelegatesWithoutChangingCaching` | Verified red (2.6: broke delegation → two calls) |
| Page states the fetch instant | Exactly one `<time>`, `datetime` in RFC 3339 | `TestThePageStampsTheFetchInstantAsRFC3339` | Genuine red (3.2: no `<time>` element) |
| Page states the fetch instant | `datetime` is an unambiguous instant | `TestTheDatetimeAttributeIsAnUnambiguousInstant` | Verified red (3.4: bare wall-clock layout) |
| Page states the fetch instant | Visible text is the Chicago wall-clock time | `TestThePageShowsTheFetchInstantInChicagoTime` | Genuine red (4.1: only RFC 3339 rendered) |
| Label does not overclaim | Text says "fetched"; no "live"/"current" | `TestTheAsOfLineIsLabelledAsFetchedAndDoesNotOverclaimCurrency` | Genuine red (4.3: bare timestamp) |
| As-of line implies no winner | 56 vs 42: no total, no margin, outside both columns | `TestTheAsOfLineImpliesNoWinner` | Verified red (placement) |
| Cache hit reports the fetch | End to end: two requests 3 min apart, same `datetime`, one upstream call | `TestAServedCacheHitDatesStatsToTheFetchNotTheRequest` | Verified red (5: broke hit path to `c.now()`) |
| Zone database | `America/Chicago` loads at init | `TestChicagoLocationLoadsAtInit` | Setup, passes green |

`TestAServedCacheHitDatesStatsToTheFetchNotTheRequest` carries both section-5 scenarios: the equal
`datetime` across the gap (5.1) and the unchanged upstream count (5.2).
