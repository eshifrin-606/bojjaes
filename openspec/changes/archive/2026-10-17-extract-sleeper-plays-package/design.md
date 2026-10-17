## Context

`internal/sleeper` is one package of five source files. The play side (`playstore.go`,
`attribution.go`, and the plays half of `fetch.go`) meets the rest through two points:

- `Client.WeekStats` calls `PlayStore.ForcedFumbles(ctx, season, week, ids)` and gets back
  `map[string]int`.
- `indexWeekly` in `fetch.go` builds the `identities` that call needs, from the stats rows. Identity
  exists only for attribution, but the data comes from the stats endpoint.

Both halves use `getJSON` and the timed `http.Client` in `fetch.go`. `cmd/server/main.go` is the
only code outside the package that names it.

All tests are in-package. The plays-side tests build identities from the stats fixtures through
`decodeWeekly`, a test helper in `sleeper_test.go` that runs the stats indexing without HTTP.

## Goals / Non-Goals

**Goals:**

- The play store, its endpoint, its JSON shapes, and attribution live in their own package, so
  "where is the forced-fumble code" has a one-directory answer.
- The package's public surface is only what `sleeper.Client` and `main` call.
- No behavior change, checked by the existing tests running unchanged in substance.

**Non-Goals:**

- Separate packages for games or stats.
- Renaming or regrouping the files that stay in `sleeper`.
- Removing `FetchWeekStats`.

## Decisions

### 1. `internal/sleeper/plays`, not `internal/plays`

The package holds Sleeper's play JSON and parses Sleeper's description text, so it is part of the
Sleeper adapter. Nesting it keeps the invariant in `docs/package-dependencies.md` true as "Sleeper
shapes appear only under `internal/sleeper/`", and `internal/games` keeps its name free of a
provider package with the same short name.

### 2. The shared HTTP helper moves to `internal/sleeper/internal/httpjson`

`plays` cannot import `sleeper`, because `sleeper` imports `plays` and Go rejects the cycle. The
helper has to live in a leaf package both can import. The `internal/` directory limits importers to
the `internal/sleeper` tree, the nearest Go has to package-private across several packages. The
function becomes `httpjson.Get(ctx, url, what, out)`, with the same error wording and the same
15-second client.

`BaseURL` stays in `sleeper`. `main` reads it from there and passes it to `plays.NewStore`, as it
does today.

Alternatives:
- Copy `getJSON` into `plays`. That gives two clients and two timeouts that can drift apart.
- Inject a fetch function into `NewStore`. That changes the constructor and every store test
  harness, and the helper still needs a home.

### 3. Public surface of `plays`

| Name | Replaces |
| --- | --- |
| `Store` | `PlayStore` (`plays.PlayStore` stutters) |
| `NewStore(baseURL, coldWait, logf)` | `NewPlayStore` |
| `(*Store).OnWholeWeekChanged` | same |
| `(*Store).ForcedFumbles(ctx, season, week, Identities) map[string]int` | same |
| `Identity`, `Identities` | `identity`, `identities` |
| `NewIdentity(first, last, team string) Identity` | `identity{name: abbrevName(first, last), team: team}` in `indexWeekly` |

`Identity` keeps unexported fields, so `NewIdentity` is the only way to build one and name
normalization stays in one place. Exported `Name`/`Team` fields with an exported `AbbrevName` were
considered and rejected: a caller could then build an identity with an un-normalized name, which
matches no description and drops credits without an error.

Everything else stays unexported: `play`, `playRow`, `playMeta`, `fetchRecentPlays`,
`forcedFumbleTurnovers`, `weekKey`, and the store's internals.

### 4. Names at the call sites

`sleeper.Client`'s `PlayStore` field becomes `Plays *plays.Store`. In `main`, the local variable
`plays` would shadow the package, so it becomes `playStore`. Log text is unchanged, including the
`sleeper plays …` and `sleeper whole-week …` prefixes the `upstream-call-logging` spec reads.

### 5. Tests move with their code and stay in-package

`attribution_test.go`, `playfetch_test.go`, `playstore_test.go`, and `wholeweekchanged_test.go` move
to `plays` as `package plays`, so they keep their access to unexported internals.

They can no longer reach `decodeWeekly`: a `package plays` test that imports `sleeper` is a cycle.
`plays` gets its own fixture helper that decodes only `player_id`, `team`, and the player's names
from a stats fixture and builds `Identities` with `NewIdentity`. The test `row()` helper builds its
identity the same way.

`TestDecodeWeeklyIdentifiesPlayersByWeekTeam` stays in `sleeper`. It compares against
`plays.NewIdentity("Maxx", "Crosby", "LV")` instead of the literal `m.crosby`. The literal stays
asserted in `plays` by `TestAbbrevName`, so the normalization is still pinned.

`client_test.go`, `bench_test.go`, `fetchweekstats_test.go`, `sleeper_test.go`, and `games_test.go`
stay in `sleeper`.

### 6. Fixtures go with the package whose payload they are

`plays_2026_w2.json`, `plays_2026_w3.json`, and `ff-test-players.csv` move to `plays/testdata/`.
`stats_2026_w2.json` and `stats_2026_w3.json` stay in `sleeper/testdata/`. Each side reads the other
side's fixture by relative path (`../testdata/stats_2026_w3.json` from `plays`,
`plays/testdata/plays_2026_w3.json` from `sleeper`). Go runs each package's tests from the package
directory, so those paths are stable. Copying instead would duplicate about 400 KB of recorded
payloads that could then diverge.

### 7. TDD for a move

The project's red-green loop is for new behavior, and this change adds none. Any new test written
for the move would pass the first time it ran. The existing suite is the specification here: each
step ends with `go test ./...` green, and the test count before and after must match. `NewIdentity`
only composes `abbrevName`, which is already tested. If a step turns out to need new logic, that
logic gets its own red-green test.

## Risks / Trade-offs

- [A test is lost in the move] → Record the `=== RUN` count of `go test -v ./internal/sleeper/...`
  before starting and compare it after. The counts must be equal.
- [The store's goroutines behave differently under the new package boundary] → Nothing about
  scheduling changes, but run `go test -race ./internal/sleeper/...` at the end.
- [Cross-package fixture paths] → A mild smell, accepted in exchange for one copy of each
  recording. If `plays` later moves out of `internal/sleeper`, the paths break loudly, not silently.
- [The dependency doc goes stale] → `docs/package-dependencies.md` is updated in this change and
  checked against `go list`.
