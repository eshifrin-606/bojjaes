## Why

When a week's whole-week plays fetch takes longer than the cold-week wait (3s on Fly, 10s in
`config/local.env`; the fetch takes 3–12s), the stats cache stores a result scored from the
300-play poll alone. Turnover forced fumbles from earlier in the week stay missing for the rest of
the 5-minute TTL, even though the play store already has them. For example, Odafe Oweh showed 5
points in 2026 week 4 until the entry expired. `plays-cold-wait` left this as its follow-up.

## What Changes

- When a whole-week plays fetch changes the play store for a week, that week's stats cache entry is
  dropped. The next read of that week, whether a reload or a switch back from another week, fetches
  again and is scored from the whole week.
- The stats cache gains a way to drop one week's entry. Dropping it makes no upstream call, so the
  cache still never fetches on its own.
- A cached result scored before the drop is never stored after it. This covers a fetch that was
  still in flight when the whole-week plays landed.
- A whole-week fetch that fails, or changes no plays, drops nothing.
- The cold-week wait and its values stay as they are.

Out of scope: the page refreshing itself; Fly still showing stale values after the TTL (a separate
investigation).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `weekly-stats-cache`: an entry can be invalidated before its TTL. A fetch in flight when its key is
  invalidated does not install its result.
- `forced-fumble-attribution`: a whole-week fetch that changes a week's plays invalidates that week's
  cached stats.

## Impact

- `internal/statscache/cache.go`: new `Invalidate(season, week)`. A flight stores its result, or
  deletes its failed entry, only if it is still the current entry for its key.
- `internal/sleeper/playstore.go`: `merge` reports whether anything changed. The store gets a
  callback, run after a whole-week fetch that changed plays.
- `cmd/server/main.go`: wires the play store's callback to the cache's `Invalidate`. `sleeper` and
  `statscache` stay unaware of each other.
- Upstream cost: at most one extra aggregate stats call per week each time a whole-week fetch
  changes plays, and only if someone reads that week again.
