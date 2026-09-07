Each behaviour below is one red-green pair. A RED task is not done until the test has been run and
has failed **for the expected behavioural reason** — a wrong count, a wrong value, a hang that the
test's own timeout reports — not a build error and not a missing symbol. Section 1 exists to get the
package compiling with a deliberately wrong implementation, so that every RED after it fails on a
mismatch.

The wrong implementations are the point. A pass-through with no cache, then a cache with no
single-flight, then a cache that stores errors: each is what the next test fails against.

Tests use a counting fake `StatsSource` and an injected clock. No `httptest` server and no network:
this package knows nothing about Sleeper, and a test that reaches one has tested the wrong thing.

## 1. A package that compiles and does nothing

- [x] 1.1 Create `internal/statscache` with a `StatsSource` interface declared locally (the same
      one-method shape `internal/api` and `internal/web` declare) and a `Cache` value built by
      `New(source, ttl)` whose `WeekStats` calls straight through to the wrapped source. No map, no
      clock, no locking. Confirm it builds and satisfies both transports' interfaces with a
      compile-time assertion in the test file.
- [x] 1.2 Add the test fake: a `StatsSource` that counts calls, returns a `score.WeekStats` built
      with `score.NewWeekStats` for the season and week it was asked for, and can be told to fail or
      to block until released. Write one test that a `WeekStats` call reaches the source once and
      returns its stats. Run; confirm pass. This test must keep passing through every task below.

## 2. Caching a fetched week

- [x] 2.1 RED: test that two calls for the same season and week call the source once. Run; confirm
      it fails reporting **two** upstream calls, against the pass-through from 1.1.
- [x] 2.2 GREEN: add a mutex-guarded map keyed by season and week, storing the result on the way
      out and returning a stored entry on the way in. No expiry yet. Re-run; confirm pass and
      confirm 1.2 still passes.
- [x] 2.3 RED: test that two different weeks each cause their own call and each caller gets its own
      week's stats. Run; confirm — if it passes immediately, keep it as the regression test that
      pins the key and say so. Then deliberately break the key to one field, watch this test fail,
      and restore it. A key test that never failed has not tested the key.

## 3. Expiry

- [x] 3.1 Add the clock seam: a `now func() time.Time` field defaulting to `time.Now`, and a
      test-only way to set it. Nothing reads it yet. Confirm the package builds and section 2's
      tests still pass.
- [x] 3.2 RED: test that a request six minutes after the fetch causes a second upstream call. Run;
      confirm it fails reporting one call, against the never-expiring cache from 2.2.
- [x] 3.3 GREEN: store the fetch time on the entry and treat an entry older than the TTL as a miss.
      Re-run; confirm pass, and confirm 2.1 (one minute later, still one call) still passes.
- [x] 3.4 RED: test both sides of the boundary — a read at TTL-minus-epsilon is a hit, a read at
      TTL-plus-epsilon is a miss. Run; confirm which side fails if either does, and settle the
      comparison deliberately rather than by whichever operator makes the test green.
- [x] 3.5 RED: test that the entry's recorded time is the fetch time and not the read time —
      advance the clock between the fetch and a cache hit and assert the stored time did not move.
      Run; confirm. This is the hook the as-of timestamp change will read; nothing renders it here.
- [x] 3.6 Confirm by reading that no goroutine sweeps expired entries. The design chose
      check-on-read; if the implementation grew a sweeper, remove it or change the design, not both
      silently.

## 4. Single-flight

- [x] 4.1 RED: test that N concurrent calls for one cold key make exactly one upstream call — start
      the fetch blocked, launch the callers, release, and assert the count and that every caller got
      the stats. Run; confirm it fails reporting N calls, against the store-on-the-way-out cache
      from 2.2. Run it under `-race` from here on.
- [x] 4.2 GREEN: install an in-flight entry holding a `done` channel before releasing the mutex;
      later callers find it, wait on the channel, and read the result the leader stored. Re-run;
      confirm pass under `-race`, and confirm sections 2 and 3 still pass.
- [x] 4.3 RED: test that a fetch in flight for week 15 does not delay a call for week 16 — block
      week 15's fetch, call week 16, and assert it returns before week 15 is released. Run; confirm
      it fails (a hang the test's own deadline reports) if the mutex is held across the fetch.
- [x] 4.4 GREEN: hold the mutex only for the lookup and the entry install, never across the upstream
      call. Re-run; confirm pass.
- [x] 4.5 RED: test that a caller arriving just after a flight completes is a hit, not a new flight
      — release the fetch, wait for the waiters, then call again and assert the count is still one.
      Run; confirm. This is what catches an in-flight entry that is removed rather than replaced
      with its result.
- [x] 4.6 Read the locking back as a whole and write down, in the change notes, the order in which
      the mutex and the channel are taken and why it cannot deadlock. If that paragraph is hard to
      write, the code is wrong, not the paragraph.

## 5. Failures are not remembered

- [x] 5.1 RED: test that after a failed fetch the next call for the same key attempts a fresh
      upstream call and can succeed. Run; confirm the failure it reports — depending on 4.2's shape
      this may fail as a stored error, a stored zero value, or a permanently closed channel. Record
      which; each is a different bug and the fix differs.
- [x] 5.2 GREEN: on error, clear the entry before closing the channel, so waiters see the error and
      the next arrival starts a new flight. Re-run; confirm pass.
- [x] 5.3 RED: test that every waiter on a failed flight gets an error and none gets stats. Run;
      confirm. Then assert the upstream was called exactly once for that flight — a failure must not
      turn N waiters into N retries.
- [x] 5.4 RED: test that a successful fetch after a failure is cached — call, fail, call, succeed,
      call again, assert two upstream calls total. Run; confirm.

## 6. Cancellation

- [x] 6.1 RED: test that cancelling the *leader's* context mid-fetch does not fail the waiters —
      block the fetch, launch a leader and two waiters, cancel the leader's context, release, and
      assert the waiters got stats. Run; confirm it fails against a fetch that inherits the caller's
      context.
- [x] 6.2 GREEN: run the upstream call under `context.WithoutCancel` of the caller's context, with
      the cache's own timeout. Re-run; confirm pass under `-race`.
- [x] 6.3 RED: test that a *waiter* whose own context is cancelled returns promptly with a
      cancellation error while the flight completes and the other waiters still get stats. Run;
      confirm it fails as a hang if the waiter only selects on the done channel.
- [x] 6.4 GREEN: have waiters select on both the done channel and their own `ctx.Done()`. Re-run;
      confirm pass.
- [x] 6.5 RED: test that a fetch exceeding the cache's timeout returns an error to its callers and
      leaves nothing cached. Run with a short injected timeout; confirm, and confirm the next call
      retries.

## 7. Wiring it up

- [x] 7.1 Wrap `sleeper.Client` in `statscache.New` in `cmd/server/main.go` and pass the same value
      to `api.BatchHandler` and `web.Handler`. One cache, one upstream budget. Confirm `go build`
      and that no handler file changed.
- [x] 7.2 Confirm the TTL constant lives in `statscache` and is named for what it bounds, not for
      its value, and that `main` names no clock and no timeout — the defaults are the package's.
- [x] 7.3 Run the server against the real lineup tree. Time a cold week and three reloads inside
      five minutes with `curl -s -o /dev/null -w '%{time_total}'`, and confirm from the gap that
      only the first was a fetch. Then wait out the TTL and confirm the next load is slow again.
      (Originally written as "first load slow, the rest instant" judged by eye. That premise was
      wrong: a miss costs ~35ms through a warm connection, which is invisible in a browser. The
      gap is real but needs an instrument, not perception.)
- [x] 7.4 With the server running, hit the same week from `scripts/scores.sh` and the page and
      confirm they share the cache — the second of the two is instant regardless of which came
      first. This is the point of wrapping once in `main` rather than per handler.

## 8. Close out

- [x] 8.1 Update `docs/package-dependencies.md`: add the `internal/statscache` row and its arrows,
      and amend the `cmd/server` note that currently predicts this change in the future tense.
- [x] 8.2 Run `gofmt -l .`, `go vet ./...`, `go test ./...`, and `go test -race ./...`; confirm
      clean. Confirm `go.mod` gained no dependency, and that `internal/sleeper`, `internal/web`,
      `internal/api`, `internal/score`, `internal/roster`, `scripts/**`, and the roster CSV format
      are untouched.
- [x] 8.3 Walk the `weekly-stats-cache` delta spec scenario by scenario and record in the change
      notes which test covers each. A scenario covered by nothing is a gap; a scenario covered by a
      test that never went red is a weaker claim and should be marked as such.
- [x] 8.4 Re-read the cache as a whole. Delete any comment restating the code; keep the ones
      carrying rules — why the fetch is detached from the caller, why errors are not stored, why the
      mutex is not held across the fetch.
- [x] 8.5 Tick the backlog's "Put a ~5 minute TTL cache over the weekly fetch" line. Note in the
      change notes that the as-of timestamp is now the strongest next line, since a cache is what
      makes a page able to be quietly stale, and that the rate-limit probe is still open.
