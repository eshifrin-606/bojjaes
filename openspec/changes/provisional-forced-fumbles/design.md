## Context

`score-forced-fumbles` D4 stages play-by-play as a synchronous 300-play poll plus a background
whole-week fetch (`limit=5000`, 3–12 s). Its rejected alternative was blocking on the whole-week
fetch. The cost of not blocking is that a cold or gapped read returns forced fumbles counted from the
poll alone. The stats cache (`statscache.TTL`, 5 min) then pins that result, and the page reloads
only every 5 min, so the background fetch's result goes unseen for up to 5 minutes.

Two more facts shape this design:

- After a cold start, `needsWholeFetch` only re-detects a hole while the store is empty or the poll
  leaves a gap. If the cold whole-week fetch fails, the poll's 300 plays fill the store, later polls
  overlap them, and the week stays partial until the post-game refresh.
- An empty store triggers a whole-week fetch on every read, so a future week (zero plays) costs a
  5000-limit fetch every 5 minutes for as long as someone views it.

## Goals / Non-Goals

**Goals:**

- Correct forced fumbles reach an open page within about 15 s of the whole-week fetch landing.
- A failed hole fill is retried rather than forgotten, at a bounded rate.
- No whole-week fetch for a week the poll already covers.
- The machine stays up between game-day reads.

**Non-Goals:**

- Blocking a request on the whole-week fetch, as D4 rejected.
- Pushing updates to the page. It still reloads, per `matchup-page`.
- Pre-warming weeks at boot.
- Suspend/resume (see D6).

## Decisions

### D1. The provisional flag lives on `score.WeekStats`

Both the cache (for its TTL) and the page (for its interval and note) need the flag. The page
receives only `score.WeekStats` and a fetch time from `WeekStatsAsOf`. So the flag rides on
`WeekStats`, read through a `Provisional() bool` method. `NewWeekStats` keeps its signature, so
existing adapters and tests are untouched. The sleeper adapter marks a value provisional through a
second constructor or a value-returning method; the implementer chooses.

"Provisional" is provider-neutral: it says the week's data is known to be incomplete and will be
filled soon. It does not mention plays.

*Alternative:* an optional interface the cache type-asserts on the source's result. Rejected: it does
not reach the page, so the page would need a second path.

*Alternative:* a third return value on `StatsSource.WeekStats`. Rejected: every interface declaring
`WeekStats` (web, statscache, benches) and every fake would change for one bit of data.

### D2. The play store tracks holes, not just in-flight fetches

Per week, the store adds:

- `holeOpen`: set when a full-limit poll finds an empty store or a gap, or when the poll fails with
  nothing held. Cleared when a whole-week fetch succeeds, or by a short poll (D3).
- `lastWholeFailure`: when a whole-week fetch last failed.

`ForcedFumbles` returns `(counts, provisional bool)`. On each read:

1. Poll. If the poll succeeded and is short, apply D3.
2. Otherwise open the hole if one is detected.
3. Start a whole-week fetch if the hole is open, none is running, and `lastWholeFailure` is older
   than `holeRetryAfter` (about 5 min). The post-game refresh condition is unchanged.
4. Merge the poll. Under one lock, take the snapshot and compute
   `provisional = holeOpen && wholeInFlight`.

The snapshot and the flag come from the same lock acquisition. Then a fetch that merges between the
two cannot produce partial counts labelled complete for 5 minutes.

`holeRetryAfter` is the sleeper package's own constant, matching the cache TTL in spirit. It does not
import `statscache`. It bounds a failing fill to one 5000-limit retry per ~5 min, the same cadence
as the poll before this change. Because a failed fill is not provisional (step 4), the entry built
after a failure gets the full TTL. The retry cadence and cache cadence then line up instead of
retrying every 15 s.

*Alternative:* `provisional = wholeInFlight` alone. Rejected: a post-game refresh would make every
finished week reload at 15 s once, for no visible change.

### D3. A short poll is a whole-week read

A poll that succeeds with fewer than `pollLimit` plays has returned the week. The store merges it,
records `markWholeFetched(now)`, and closes the hole, so it serves as a post-game refresh too. A
failed poll returns no plays but is never short. The check requires a nil error.

The recent endpoint is CDN-cached for 300 s whatever the limit, so a short poll is no staler than a
whole-week fetch at the same moment.

### D4. The cache sets each entry's TTL when the fetch completes

`entry` gains a `ttl` set alongside `fetchedAt`: `ProvisionalTTL` (15 s) when
`stats.Provisional()`, otherwise the cache's configured TTL. `expired` reads the entry's TTL. Single
flight, failure handling, and `fetchedAt` are unchanged. A provisional hit still reports its true
fetch time.

`ProvisionalTTL` is exported next to `TTL`, so the page's interval and the cache's TTL are both named
in one place, even though the template writes its interval as a literal.

### D5. The page reads provisional status from the HTML, not the script

The view gains `Provisional bool`. When it is true, the as-of paragraph carries a
`data-provisional` attribute and a short note: "forced fumbles still loading". The script picks its
interval with a selector:

```
var refreshInterval = document.querySelector("[data-provisional]") ? 15 * 1000 : 5 * 60 * 1000;
```

This keeps "the refresh script carries no page data": the script is byte-identical on both kinds of
page. The return-to-tab rule uses the same variable, so a provisional page that was hidden for more
than 15 s reloads on return.

### D6. Always-on machine, not suspend

`fly.toml` sets `min_machines_running = 1`, about $0.45 a week. ADR 0005 already decides this for the
volume, so this change lands it early and updates the `docs/architecture.md` deployment row. It needs
no new ADR. `auto_stop_machines` stays `'stop'` so any extra machines still scale down.

*Alternative:* `auto_stop_machines = 'suspend'` (free, memory survives). Rejected: `statscache`
measures expiry with `time.Time.Sub`, which uses Go's monotonic clock. The monotonic clock may not
advance across a Firecracker snapshot, so an entry fetched before suspend could be served as fresh
for up to 5 minutes after a resume hours later. Fixing that is cheap, but the trap would stay for
every future time comparison.

### D7. Archive `score-forced-fumbles` first

The `forced-fumble-attribution` delta here modifies requirements that exist only in that change. It
is complete (62/62), so it is archived before this change is applied, and this change's deltas apply
on top.

## Risks / Trade-offs

- **[A slow fill reloads the page several times]** → A 15 s interval against a fetch with a 30 s
  budget means at most about two provisional reloads per reader. Each is a single-flighted aggregate
  fetch, plus a poll the CDN has cached.
- **[A permanently failing fill]** → The hole stays open, with one retry per ~5 min and a log line
  for each. Readers see the D5 note only while a retry is in flight, and otherwise see the
  poll-derived counts as today.
- **[Short-poll misjudgment]** → If Sleeper ever returns fewer than `limit` plays while more exist
  (paging, truncation), the week is wrongly marked whole. The post-game refresh condition is keyed
  off the newest change, so a later change still triggers one. Recordings show `limit=5000`
  returning whole weeks, which supports reading a short result as complete.
- **[Flag on a domain type]** → `WeekStats` now carries freshness metadata beside stat lines. This is
  acceptable because it describes the data's completeness, not its transport.

## Migration Plan

Deploy normally. The play store is in memory, so the new hole state starts empty, and the first read
of each week behaves as a cold read. Rollback is a redeploy of the previous image. `fly.toml`'s
`min_machines_running` can be set back to 0 on its own.

## Open Questions

- Wording of the loading note. "forced fumbles still loading" is the placeholder.
