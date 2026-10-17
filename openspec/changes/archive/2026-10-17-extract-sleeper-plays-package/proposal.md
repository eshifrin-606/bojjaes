## Why

`internal/sleeper` holds four concerns in one directory — the REST transport, the weekly stat
mapping, games, and the play store with forced-fumble attribution — and the file names do not say
which is where. The play side is half the package (about 340 lines of code and 900 of tests), has
its own state, goroutines, and lifecycle, and meets the rest of the package through one narrow call.
It is the one concern that earns a package boundary; the others are small enough that a boundary
would cost more than it reveals.

## What Changes

- Move the play store, the plays endpoint call, the play JSON shapes, and forced-fumble attribution
  into a new package, `internal/sleeper/plays`, with their tests and fixtures.
- Move the shared HTTP helper (`getJSON` and its timed `http.Client`) into
  `internal/sleeper/internal/httpjson`, so `sleeper` and `plays` can both use it without an import
  cycle and nothing outside `internal/sleeper` can.
- `sleeper.Client` keeps `WeekStats` and `WeekGames`. Its play-store field becomes a `*plays.Store`,
  and the identities it hands the store become `plays.Identities`.
- `cmd/server` constructs the store with `plays.NewStore` instead of `sleeper.NewPlayStore`.
- Regroup what stays in `sleeper` by concern: `sleeper.go` holds the package doc, `BaseURL`, and
  `Client`; `stats.go` (renamed from `fetch.go`) holds the stats endpoint and mapping; the tests in
  `sleeper_test.go` become `stats_test.go`.
- Update `docs/package-dependencies.md` to show the new packages and edges.
- No behavior changes: same endpoints, same log lines, same attribution, same caching.

Not in this change: splitting games or stats into packages, or removing `FetchWeekStats` (which has
no caller outside its own tests).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `forced-fumble-attribution`: the ground-truth requirement names its fixture by path, and that path
  moves to `internal/sleeper/plays/testdata/ff-test-players.csv`. The requirement itself is
  unchanged.

## Impact

- Code: `internal/sleeper` (`fetch.go`, `sleeper.go`, `playstore.go`, `attribution.go`, and the
  plays-side tests), new `internal/sleeper/plays` and `internal/sleeper/internal/httpjson`,
  `cmd/server/main.go`.
- Public API: `sleeper.NewPlayStore` and `sleeper.PlayStore` are replaced by `plays.NewStore` and
  `plays.Store`. `main` is the only caller.
- Docs: `docs/package-dependencies.md`.
- Dependencies: none added.
