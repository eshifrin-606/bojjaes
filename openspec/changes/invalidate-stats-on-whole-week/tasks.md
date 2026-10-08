Each task is one red-green behavior unless marked otherwise. When a test needs a new symbol, first
add it as a stub that compiles and does nothing. That way the red shows a value mismatch, not a build
error.

## 1. `statscache`: invalidation

- [x] 1.1 Stub `func (c *Cache) Invalidate(season, week int)` as a no-op.
- [x] 1.2 Red-green: fetch week 4, invalidate it, request it again within the TTL. Expect two upstream
      calls, with the second caller getting the second fetch's stats.
- [x] 1.3 Confirm green: invalidate alone makes no upstream call. Invalidating week 4 leaves week 5
      cached. Invalidating a never-fetched week does nothing.

## 2. `statscache`: in-flight invalidation (D2)

- [x] 2.1 Red-green: block the fake source, start a week 4 fetch, invalidate week 4, release it. The
      waiter gets the stats. A later request within the TTL makes a new upstream call.
- [x] 2.2 Red-green: the first flight is invalidated, a second request starts a new flight, and the
      first flight then fails. The second flight's result is stored, so a third request within the
      TTL is a hit.
- [x] 2.3 Confirm green: run the whole `statscache` suite with `-race`.

## 3. `PlayStore`: report changed whole-week merges (D4)

- [x] 3.1 Refactor: `merge` returns whether it added or replaced a play. Nothing uses the result
      yet, so the suite stays green.
- [x] 3.2 Stub `func (s *PlayStore) OnWholeWeekChanged(fn func(season, week int))` storing nothing.
- [x] 3.3 Red-green: a cold week's whole-week fetch adds plays. After `store.wait()`, the callback has
      been called once with that season and week.
- [x] 3.4 Red-green: a whole-week fetch that returns only plays already held, with the same
      `updated_at`, doesn't call the callback (for example, a post-game refresh after a full seed).
- [x] 3.5 Red-green, or confirm green: a failed whole-week fetch doesn't call the callback.
- [x] 3.6 Confirm green: a poll that adds plays doesn't call the callback. With a nil callback,
      nothing panics (the existing tests cover this).

## 4. Wiring

- [x] 4.1 `main.go`: keep the `*PlayStore` in a variable, build the cache, then call
      `plays.OnWholeWeekChanged(stats.Invalidate)` before listening.
- [x] 4.2 End-to-end test, if a seam exists (otherwise a manual check; no seam exists, so 4.3 covers it): a cold read with a short wait
      and a blocked whole-week response returns poll-only forced fumbles. After release and
      `store.wait()`, the next read through the cache credits the whole-week forced fumble.
- [x] 4.3 Manual check: `go run ./cmd/server` with `config/local.env`, load 2026 week 4 cold, wait
      about 15s, then reload. Oweh's forced fumble shows. The log has a second `statscache miss` for
      week 4 after the `whole-week` line.
