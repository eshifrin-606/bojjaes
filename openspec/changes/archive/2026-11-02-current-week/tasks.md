Each behaviour is one red-green pair. A RED task is done only when the test has been run and has
failed **on a value mismatch or a missing error**, not on a build error. Section 1 adds compiling
stubs that are deliberately wrong, so every later RED fails for the behavioural reason. A "Guard"
is a test that already passes: break the production code on purpose, watch it fail, then restore.

Out of scope: any route, page, form, `internal/web` interface, or `cmd/server` wiring.

## 1. Compiling stubs to fail against

- [x] 1.1 New package `internal/nflweek`: `Phase` with `Preseason Regular Postseason Offseason
      Other`, `Current`, `New`, and `Settable` returning `0, 0, false`.
- [x] 1.2 `internal/sleeper`: `V1BaseURL` constant and `Client.V1BaseURL` field;
      `state_endpoint.go` with `stateRow`, a `fetchState` that returns a zero row and nil, and
      `statePhase(string) nflweek.Phase` returning `Other`;
      `Client.CurrentWeek` in `client.go` returning `nflweek.Current{}, nil`.
- [x] 1.3 `internal/weekcache`: `CurrentWeekTTL = time.Hour`; `Single[V]`, `NewSingle`, and an
      `AsOf` that calls the source on every request and returns `time.Time{}`. Run
      `go build ./... && go test ./...`: green.

## 2. A settable week (internal/nflweek)

- [x] 2.1 RED: `New(2026, 9, Regular).Settable()` → `2026, 9, true`. Confirm it fails on
      `false`.
- [x] 2.2 GREEN: return the season and week when the phase is `Regular`.
- [x] 2.3 Guard: regular week 18 → settable (no league cap here).
- [x] 2.4 RED: regular week 0 → not settable. GREEN: lower bound at 1.
- [x] 2.5 Guard (table): `Preseason`, `Postseason`, `Offseason`, `Other`, each week 1 → not
      settable. Break the phase check to see it fail, then restore.

## 3. Reading Sleeper's state (internal/sleeper)

- [x] 3.1 Save `curl https://api.sleeper.app/v1/state/nfl` as `testdata/state_nfl.json`, then
      edit it to season `"2026"`, `season_type` `regular`, week 9 so the test is stable.
- [x] 3.2 RED: an `httptest.Server` serving the fixture at `/state/nfl`, `Client{V1BaseURL: …}`
      `.CurrentWeek(ctx).Settable()` → `2026, 9, true`. Confirm it fails on `false`.
- [x] 3.3 GREEN: `fetchState` builds `{V1BaseURL}/state/nfl`, decodes via `httpjson.Get`;
      parse the string season with `strconv.Atoi`, ignoring its error for now; map `regular` in
      `statePhase`; `CurrentWeek` returns `nflweek.New`.
- [x] 3.4 RED (table): `pre`, `post`, `off`, `exhibition` → the matching `Phase` via `statePhase`.
      Confirm `pre`/`post`/`off` fail on `Other`. GREEN: the remaining cases.
- [x] 3.5 RED: season `""` → error. Confirm it fails because nil is returned. GREEN: return the
      `strconv.Atoi` error, wrapped to name the state read.
- [x] 3.6 Guards: 503 → error; body `[]` → error. Both already pass through `httpjson.Get`;
      break the error return in `CurrentWeek` to see them fail, then restore.
- [x] 3.7 RED: with `Logf` captured, a successful read logs
      `sleeper state: 2026 regular w9 in …`. Confirm no line is logged. GREEN: time and log like
      `WeekGames`. Guard: a failed read logs no state line.

## 4. Caching the state (internal/weekcache)

- [x] 4.1 RED: `NewSingle` with a counting source and `WithClock`; two `AsOf` calls ten minutes
      apart → source called once, same value and fetch instant. Confirm it fails on a count of 2.
- [x] 4.2 GREEN: `Single` wraps a `*Cache[V]` whose source ignores season and week; `AsOf` reads
      one fixed key.
- [x] 4.3 Guard: sixty-one minutes apart with `CurrentWeekTTL` → called twice.
- [x] 4.4 Guard: a failing first call, then a second a minute later → called twice. Break
      `Cache`'s failure delete to see it fail, then restore.
- [x] 4.5 RED: a `Single` hit logs `nflstate hit: current, 10m0s old`. Confirm it fails because the
      line reads `0 week 0`. GREEN: unexported key formatter on `Cache`, default
      `{season} week {week}`, `current` for `Single`; use it in the hit and miss lines.
- [x] 4.6 RED-or-guard: a `Single` miss logs `nflstate miss: current fetched in …`.
- [x] 4.7 Guard: existing `statscache hit: 2026 week 5, …` tests still pass unchanged. Run
      `go test ./...`: green.

## 5. Docs

- [x] 5.1 `docs/package-dependencies.md`: add `internal/nflweek` (domain) with
      `sleeperpkg --> nflweek`, and note `Single` under `internal/weekcache`.
- [x] 5.2 Tick the "Current NFL week" row in `backlog.md`.
