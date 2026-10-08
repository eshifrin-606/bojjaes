## Context

`statscache.Cache` wraps `sleeper.Client`, and the client holds a `*PlayStore`. A cache miss runs
`Client.WeekStats`, which polls 300 plays, may start a background whole-week fetch, waits for it up to
`PLAYS_COLD_WAIT` if the week was cold, and then scores forced fumbles from the store. The cache then
holds that result for 5 minutes. A whole-week fetch that lands after the wait updates the store, but
nothing reaches the cache, so the page shows poll-only forced fumbles until the TTL expires.

`cmd/server/main.go` is the only package that names both `sleeper` and `statscache`, and that should
stay true.

## Goals / Non-Goals

**Goals:**
- After a whole-week fetch changes a week's plays, the next read of that week is correct.
- No result scored before the change is cached after it, including one still in flight.
- Upstream cost stays bounded: invalidation never fetches.

**Non-Goals:**
- A correct first load when the fetch outlasts the wait. That needs a longer wait.
- Pushing updates to an open page, or the page refreshing itself.
- Fly serving stale data after the TTL (separate investigation).
- Recomputing only forced fumbles on a partial hit. The whole week is refetched instead.

## Decisions

### D1. Drop the entry; the next read is an ordinary miss

`Cache.Invalidate(season, week)` deletes the key's entry under `c.mu`. It doesn't fetch, so "the
cache never fetches on its own" still holds.

*Alternative:* a "plays incomplete" flag on the entry, with only forced fumbles re-scored on read
(the "partial hit"). Rejected for now: `WeekStats` would have to keep forced fumbles separate from the
aggregate, and the cache would stop being agnostic about what it holds. That saves one aggregate call
per invalidation, which is rare.

### D2. A flight installs its result only if it is still the current entry

Today a flight writes `fetchedAt` on success, and on failure `delete(c.entries, k)`. Once an
invalidation can remove a flight from the map while it runs, both writes become conditional on
`c.entries[k] == flight`:

- On success with the flight removed, the waiters still get the stats through the `flight` pointer,
  but the map no longer refers to it, so nobody else is served from it. Its `fetchedAt` is still set,
  so waiters get a real timestamp.
- On failure with the flight removed, it must not delete a newer entry installed under the same key.

That handles the race where whole-week plays land between the play snapshot and the cache storing the
result: the invalidation removes the in-flight entry, and its result is never cached.

Deleting an in-flight entry means a request arriving before the flight finishes starts a second
upstream call. That's acceptable: it only happens when a whole-week fetch lands during a flight.

*Alternative:* a per-key generation counter. It's equivalent here, but pointer identity needs no new
state.

### D3. `PlayStore` takes a callback, set after construction

`main` builds the store, then the client, then the cache that wraps the client. The cache's
`Invalidate` doesn't exist until after the store does, so the callback can't be a `NewPlayStore`
argument without a forward-declared closure. Instead:

```go
plays := sleeper.NewPlayStore(sleeper.BaseURL, coldWait, log.Printf)
stats := statscache.New(sleeper.Client{..., PlayStore: plays, ...}, statscache.TTL)
plays.OnWholeWeekChanged(stats.Invalidate)
```

It's set once in `main`, before the server starts listening, so there's no read/write race on the
field. A nil callback (every existing test) does nothing. `sleeper` declares the callback as a plain
`func(season, week int)`, so it doesn't import `statscache`.

*Alternative:* `NewPlayStore` takes the callback, and `main` passes a closure over a `var stats`
assigned later. Rejected: it only works because of the order of statements in `main`.

### D4. Only a changing whole-week merge invalidates

`merge` returns whether it added or replaced any play. `fetchWholeWeek` calls the callback after
`markWholeFetched`, and only when that's true. Calling the callback outside `s.mu` keeps the
store's lock and the cache's lock from nesting.

- **Failed fetch:** no merge, so no call.
- **Post-game refresh with no corrections:** no change, so no call. That avoids an extra aggregate
  call every time a finished week goes quiet.
- **Poll merges never invalidate.** A poll runs inside the flight it feeds, so invalidating would
  throw away the result being computed and force a needless refetch.

*Alternative:* invalidate only when the forced-fumble counts change. Rejected: the store doesn't
know the week's `identities`, and comparing scores means computing them twice. "Plays changed" is a
cheap, conservative stand-in.

## Risks / Trade-offs

- **A live week with frequent gap fills invalidates repeatedly.** Each reload after a gap fill costs
  one aggregate call. → A gap fill needs a read to trigger it, and the next read is the only one that
  pays. The cost is still bounded by reads, not by time.
- **First load can still be wrong.** → Accepted (non-goal). The page is correct after one reload or
  week switch, about 12s after a cold load at worst.
- **A cold read that waits out the full fetch also invalidates.** The fetch lands inside the wait, so
  the result about to be cached is already complete, but the invalidation drops the in-flight entry
  anyway. The next read then costs one extra upstream call. → Acceptable for correctness and
  simplicity. If it matters, the waiter could skip invalidation when it observed completion, but
  that couples the cache to the play store's timing.

## Migration Plan

Deploy as usual. There's no configuration change. To roll back, revert. Removing the
`OnWholeWeekChanged` line in `main` turns it off.

## Open Questions

- Fly shows stale forced fumbles even after the TTL. That's unexplained, may share a cause, and should
  be investigated after this lands.
