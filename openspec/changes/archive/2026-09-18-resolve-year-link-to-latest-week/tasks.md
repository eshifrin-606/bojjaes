## 1. `Tree.LatestWeek`

- [x] 1.1 Add a failing test: a tree with `2026/1/` and `2026/2/` reports latest week 2 for season
      2026 (`ok == true`).
- [x] 1.2 Implement `Tree.LatestWeek(season int) (week int, ok bool)` via `fs.ReadDir` on the season
      directory, parsing each directory entry's name as an integer and keeping the max, to make 1.1
      pass.
- [x] 1.3 Add a failing test: weeks 1 and 3 present, week 2 absent — latest week is 3, not 2.
- [x] 1.4 Add a failing test: no directories under a season — `ok == false`.
- [x] 1.5 Add a failing test: the season directory holds a non-directory entry named `notaweek` (or
      similar) alongside `1/` — latest week is still 1, the non-integer entry is ignored.
- [x] 1.6 Add a failing test: a week directory that is not a matchup (three roster files) still
      counts toward the latest week.
- [x] 1.7 Add a failing test asserting no roster file is opened while computing the latest week
      (e.g. over an `fstest.MapFS` with a roster file that would panic or error if read).

## 2. `GET /{season}` handler

- [x] 2.1 Add a failing test: `GET /2026` against a tree with 2026 weeks 1 and 2 responds with a
      redirect (`Location: /2026/2`).
- [x] 2.2 Implement the handler — validate the season with `score.ValidateSeasonWeek(season, 1)`,
      call `tree.LatestWeek`, redirect with `http.StatusFound` — to make 2.1 pass.
- [x] 2.3 Register `GET /{season}` in `cmd/server/mux.go`.
- [x] 2.4 Add a failing test: a non-numeric season segment responds `400 Bad Request`.
- [x] 2.5 Add a failing test: a season outside `internal/score`'s bounds responds `400 Bad Request`.
- [x] 2.6 Add a failing test: a season with no week directories responds `404 Not Found`.
- [x] 2.7 Add a failing test: a gap in the season's weeks (weeks 1 and 3, not 2) redirects to week 3.
- [x] 2.8 Add a failing test: the season's only week is not a well-formed matchup (three rosters) —
      `GET /{season}` still redirects to it.
- [x] 2.9 Add a failing test asserting the stats provider (a fake `StatsSource` that fails the test
      if called) is never invoked by `GET /{season}`.
- [x] 2.10 Confirm `GET /{season}/{week}` still routes to the existing handler unchanged (add a
      regression test if none already pins this).

## 3. Verify

- [x] 3.1 Run `go test ./...` and confirm all tests pass.
- [x] 3.2 Run `go vet ./...`.
