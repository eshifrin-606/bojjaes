Each task is one red-green behavior unless marked otherwise. When a test needs a new symbol, first
add it as a stub that compiles and returns the zero value. That way the red shows a value mismatch,
not a build error.

## 0. Prerequisite

- [x] 0.1 Archive `score-forced-fumbles` so `openspec/specs/forced-fumble-attribution/` exists for
      this delta to modify. Confirm with the user before archiving.

## 1. Constructor carries the wait (refactor, no behavior change)

- [x] 1.1 Change `NewPlayStore(baseURL, logf)` to `NewPlayStore(baseURL, coldWait, logf)`, stored as
      a field. Update every caller: `0` in tests, a placeholder `0` in `main.go` until section 4.
      `go test ./...` stays green.
- [x] 1.2 Change `wholeInFlight` to `map[weekKey]chan struct{}`, closed when the fetch finishes.
      `startWholeFetch` returns that channel (new or already running). Existing single-flight and
      failure tests stay green.

## 2. Cold-week wait in `ForcedFumbles`

- [x] 2.1 Red-green: with a nonzero wait, a cold week credits a forced fumble that is only in the
      whole-week response (outside the 300-play poll) on the *first* call. Use the existing
      fake-server pattern in `playfetch_test.go`.
- [x] 2.2 Confirm green: with wait `0`, the same setup credits it only after `store.wait()` and a
      second call (the existing `...SeedsEmptyWeekInBackground` / `...OnNextCall` tests cover this).
- [x] 2.3 Red-green: the wait runs out. Block the fake whole-week response and use a wait of a few
      ms. The call returns with poll-only credits. After release and `store.wait()`, the store
      holds the whole-week plays.
- [x] 2.4 Red-green: caller ctx already done → the call returns with poll-only credits and does not
      wait for the blocked whole-week response.
- [x] 2.5 Red-green, or confirm green: the whole-week fetch fails within the wait → the call returns
      without waiting out the timer (assert well under the wait), and the failure is logged.
- [x] 2.6 Red-green: a gap (store non-empty, poll older-edge newer than held) with a nonzero wait
      does not wait. Block the fake whole-week response, and the call still returns.

## 3. `PLAYS_COLD_WAIT` resolution (`cmd/server`)

- [x] 3.1 Stub `resolvePlaysColdWait(getenv) (time.Duration, error)` returning `0, nil` in a new
      file beside `addr.go`.
- [x] 3.2 Red-green: `10s` → 10 s.
- [x] 3.3 Red-green: unset → 30 s. Then confirm green: empty → 30 s.
- [x] 3.4 Red-green, or confirm green: `0` → 0.
- [x] 3.5 Red-green: `ten` → error mentioning `PLAYS_COLD_WAIT`.
- [x] 3.6 Red-green: `-5s` → error mentioning `PLAYS_COLD_WAIT`.

## 4. Wiring and deploy config

- [x] 4.1 `main.go`: resolve the wait before constructing the store, `log.Fatalf` on error, and pass
      it to `NewPlayStore`. Log the resolved wait once at startup.
- [x] 4.2 `fly.toml`: add `PLAYS_COLD_WAIT = '0'` under `[env]`.
- [x] 4.3 Manual check: `go run ./cmd/server`, load a current week cold, and confirm turnover forced
      fumbles show on first load and the log has the startup wait line.

## 5. Docs

- [x] 5.1 Amend `score-forced-fumbles`' archived design D4 in place (or its successor in
      `docs/architecture.md`, wherever D4's "never wait" now lives) so it points to the bounded
      cold-week wait. Per the doc-consolidation preference, don't add a separate superseding note.
- [x] 5.2 Mention `PLAYS_COLD_WAIT` in `docs/architecture.md` beside `LINEUP_VOLUME` (its local `30s`
      and Fly `0` values).
