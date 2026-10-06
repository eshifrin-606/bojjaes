Each task is one red-green behavior unless marked otherwise. When a test needs a new symbol, first
add it as a stub that compiles and returns the zero value. That way the red shows a value mismatch,
not a build error.

## 1. Prerequisite

- [ ] 1.1 Archive `score-forced-fumbles` (complete, 62/62) so `forced-fumble-attribution` exists in
      `openspec/specs/` before this change's delta applies (design D7). Not a code task.

## 2. Domain: provisional weekly stats

- [ ] 2.1 Stub: `WeekStats.Provisional() bool` returning false, plus a way to build a provisional
      value (design D1) that does not yet set the flag. `NewWeekStats` keeps its signature.
- [ ] 2.2 Red-green: a `WeekStats` built as provisional reports `Provisional()` true.
- [ ] 2.3 Confirm green: `NewWeekStats` values report false, and a provisional value's `Player`
      lookups match the non-provisional one's.

## 3. Play store: short poll is the whole week (design D3)

- [ ] 3.1 Red-green: a successful poll with fewer than `pollLimit` plays into an empty store starts no
      whole-week fetch.
- [ ] 3.2 Red-green: a successful empty poll starts no whole-week fetch on two consecutive reads (the
      future-week case).
- [ ] 3.3 Red-green: a short poll records a whole-week fetch at the poll time, so a quiet week whose
      newest change is more than an hour before that time gets no post-game refresh.
- [ ] 3.4 Red-green: a failed poll with nothing held still starts a whole-week fetch (the short-poll
      rule needs a nil error).

## 4. Play store: holes outlive failed fetches (design D2)

- [ ] 4.1 Red-green: after a cold whole-week fetch fails, a read more than `holeRetryAfter` later,
      whose full-limit poll overlaps the held plays, starts a whole-week fetch.
- [ ] 4.2 Red-green: a read less than `holeRetryAfter` after that failure starts no whole-week fetch.
- [ ] 4.3 Red-green: once a whole-week fetch succeeds, an overlapping full-limit poll starts none (the
      hole is closed).
- [ ] 4.4 Confirm green: the existing gap, overlap, single-flight, and post-game tests still pass.

## 5. Play store: report provisional (design D2)

- [ ] 5.1 Stub: `ForcedFumbles` returns `(map[string]int, bool)` with false. Update the caller.
- [ ] 5.2 Red-green: a cold read with a full-limit poll returns provisional true. Use a whole-week
      handler that blocks until released.
- [ ] 5.3 Red-green: a read after the hole's whole-week fetch has succeeded returns provisional false.
- [ ] 5.4 Red-green: a read while the hole fill is still running, whose poll overlaps, returns
      provisional true.
- [ ] 5.5 Red-green: a post-game refresh in flight with no open hole returns provisional false.
- [ ] 5.6 Confirm green: a short poll returns false, and a failed fill not yet eligible for retry
      returns false.

## 6. Client: carry the flag into weekly stats

- [ ] 6.1 Red-green: `Client.WeekStats` returns provisional stats when the play store reports
      provisional, using an `httptest` server whose whole-week response blocks.
- [ ] 6.2 Confirm green: a nil `Plays` and a non-provisional store both return non-provisional
      stats.

## 7. Stats cache: TTL per entry (design D4)

- [ ] 7.1 Stub: exported `ProvisionalTTL` set to 15 s, not yet used.
- [ ] 7.2 Red-green: a provisional entry requested 20 s after its fetch causes a second upstream
      call.
- [ ] 7.3 Confirm green: a provisional entry requested 5 s after its fetch is a hit.
- [ ] 7.4 Red-green, or confirm green: a non-provisional refetch after a provisional entry expires is
      served from cache for the full TTL (a request at 4 min is a hit).
- [ ] 7.5 Confirm green: a provisional hit reports its original `fetchedAt`.

## 8. Matchup page (design D5)

- [ ] 8.1 Red-green: a page served from provisional stats contains the forced-fumbles-loading note
      next to the as-of line, inside an element with `data-provisional`.
- [ ] 8.2 Red-green, or confirm green: a non-provisional page contains neither the note nor
      `data-provisional`.
- [ ] 8.3 Red-green: the script selects a 15 s interval when `[data-provisional]` is present and
      keeps `5 * 60 * 1000` otherwise. Assert the selector and both literals.
- [ ] 8.4 Confirm green: the script element is byte-identical on provisional and non-provisional
      pages, and the existing script-carries-no-page-data tests pass.
- [ ] 8.5 Confirm green: the layout test still passes with the note present. Add a provisional
      fixture case if the note can wrap the as-of line.

## 9. Deployment and docs

- [ ] 9.1 Set `min_machines_running = 1` in `fly.toml`. Not a code task.
- [ ] 9.2 Update the `docs/architecture.md` deployment table row so `min_machines_running = 1` is
      built, while the volume mount and `LINEUP_VOLUME` stay planned.
- [ ] 9.3 Run `go test ./...` and `go vet ./...`.
