Each task is one red-green behavior unless marked otherwise. When a red would only fail to compile,
add the bare type or symbol first so the red shows a value mismatch.

## 1. Fixture and decoder

- [x] 1.1 Convert `internal/sleeper/testdata/week14.json` to a row array: one `{player_id, stats}`
      row per existing entry, same order, same stats, hand-built entries included (design D3). Do this
      together with 1.2's red, since the fixture alone breaks the map decoder.
- [x] 1.2 Red-green: `fetchWeekly` decodes the row-array fixture into the same map as before.
      `TestFetchWeekly` and the `fixtureWeekly` helper read rows; existing expectations are unchanged.
- [x] 1.3 Red-green: a row with `"stats": null` is absent from `FetchWeekStats`
      (`TestFetchWeekStatsSkipsNullEntries` body becomes `[{"player_id":"9493","stats":null}]`).
- [x] 1.4 Red-green: an empty array is not an error (`{}` tests become `[]`).
- [x] 1.5 Confirm green: every `statLineFrom`, `TestFixtureScores`, and `FetchWeekStats` test passes
      with no expectation changed. Any expectation change is a stop-and-report, not a fix.

## 2. Endpoint

- [x] 2.1 Red-green: `fixtureServer` requires path `/stats/nfl/2025/14` and query
      `season_type=regular`; `fetchWeekly` requests it.
- [x] 2.2 Set `BaseURL = "https://api.sleeper.com"` and update the `fetchWeekly` doc comment (payload
      size, `[]` for an unplayed week).
- [x] 2.3 Update `bench_test.go` for the row shape. Run `BenchmarkDecode` and `BenchmarkTransform`
      before (on `main`) and after; record both in the PR description.
- [x] 2.4 Smoke check, not a test: run the server locally and load `/2026/3`; scores render.
- [x] 2.5 Run `go test ./...` and `go vet ./...`.

## 3. Docs and spec

- [x] 3.1 `openspec/specs/server-image/spec.md`: Purpose paragraph names `https://api.sleeper.com`.
      The requirement delta lands via sync/archive.
- [x] 3.2 `docs/architecture.md`: the sequence diagram's request and payload size.
- [x] 3.3 ADR 0003, amended in place: a dated note that weekly stats moved to the `.com` endpoint, why
      (names and game-time teams for forced-fumble attribution), the parity check, and the
      cache-header table from the proposal.
- [x] 3.4 Basic Memory: per CLAUDE.md, search for related notes, then add a journal note on the
      `.com` endpoint and its cache headers, linking
      `2026-08-09-sleeper-stats-endpoint-has-1h-edge-ttl-espn-is-seconds` and
      `0003-sleeper-as-initial-stat-provider`.
- [x] 3.5 Run `openspec validate switch-sleeper-stats-endpoint` and fix any findings.
