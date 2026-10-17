## 1. Baseline

- [x] 1.1 Record the `=== RUN` count of `go test -v ./internal/sleeper/...` and confirm
      `go test -race ./...` is green before any move (baseline: 144)

## 2. Shared HTTP helper

- [x] 2.1 Move `getJSON` and the timed `http.Client` from `internal/sleeper/fetch.go` into
      `internal/sleeper/internal/httpjson` as `Get`, with the error wording unchanged, and point the
      stats, plays, and games calls at it; `go test ./...` green

## 3. plays package

- [x] 3.1 Create `internal/sleeper/plays` with `attribution.go` (exporting `Identity`, `Identities`,
      and `NewIdentity`), `fetch.go` (play JSON shapes and `fetchRecentPlays` via `httpjson.Get`),
      and `store.go` (`Store`, `NewStore`, `OnWholeWeekChanged`, `ForcedFumbles`), removing the
      originals from `sleeper`
- [x] 3.2 Move `plays_2026_w2.json`, `plays_2026_w3.json`, and `ff-test-players.csv` to
      `plays/testdata/`
- [x] 3.3 Move `attribution_test.go`, `playfetch_test.go`, `playstore_test.go`, and
      `wholeweekchanged_test.go` to `plays`, replacing `decodeWeekly` with a plays-local identity
      fixture helper that reads `../testdata/stats_2026_w*.json` through `NewIdentity`
- [x] 3.4 In `sleeper`, build identities with `plays.NewIdentity` in `indexWeekly`, rename the
      `Client.PlayStore` field to `Plays *plays.Store`, and repoint `client_test.go` at
      `plays/testdata/plays_2026_w3.json` and `TestDecodeWeeklyIdentifiesPlayersByWeekTeam` at
      `plays.NewIdentity`
- [x] 3.5 In `cmd/server/main.go`, construct the store with `plays.NewStore`, rename the local to
      `playStore`, and set `Client.Plays`; `go test ./...` and `go vet ./...` green

- [x] 3.6 Regroup `sleeper` by concern: rename `fetch.go` to `stats.go` and move `statLineFrom`,
      `FetchWeekStats`, and `fetchPlayers` into it; move `BaseURL` into `sleeper.go`; rename
      `sleeper_test.go` to `stats_test.go`; RUN count still 144

## 4. Docs

- [x] 4.1 Update `docs/package-dependencies.md`: add `internal/sleeper/plays` and
      `internal/sleeper/internal/httpjson` to the map and table, and move the play-store prose
      under the new package; check the edges against `go list`

## 5. Verify

- [x] 5.1 The `=== RUN` count across `./internal/sleeper/...` equals the baseline, and
      `go test -race ./...` is green
- [x] 5.2 `grep` finds no `NewPlayStore`, `PlayStore`, or `getJSON` left under `internal/` or
      `cmd/`
