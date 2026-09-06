## Why

`internal/score` is four concerns in one package: the domain stat line, the HMFFL scoring rules, the
Sleeper adapter, and the JSON HTTP layer. [ADR 0002](../../../docs/adr/0002-live-scoreboard-backend.md)
decision #1 asked for the opposite — "all source-specific code (HTTP shape, player IDs, parsing)
lives behind [an interface]. The scoring and serving layers depend only on the interface, never on a
concrete vendor" — and today that separation is held up by a comment rather than by the compiler.
Nothing prevents `Points` from reading a Sleeper key tomorrow.

It becomes load-bearing now because the matchup page is the second consumer of scoring. A second
package cannot score a roster without either going through HTTP or having Sleeper's
`map[string]map[string]float64` handed to it, and publishing that map as the package's interface is
exactly what ADR 0002 rules out.

## What Changes

- Split `internal/score` into three packages along the seams that already exist in the files:
  - `internal/score` keeps the vendor-neutral core — `StatLine`, `Points`, and a new `Week` — with
    no I/O and no HTTP.
  - `internal/sleeper` takes the adapter: the HTTP client, the base URL, and the stat-key transform.
    It is the only package where a vendor stat key appears.
  - `internal/api` takes the JSON layer: the `POST /scores` handler, its request and response
    types, and its validation.
- Add `score.Week`, a provider-neutral snapshot of one season and week, built by
  `sleeper.FetchWeek`. Every player in the payload is transformed at fetch time, so Sleeper's raw
  map never leaves the adapter.
- **BREAKING**: delete `GET /score`. It is a walking-skeleton smoke test over a hardcoded player and
  week that also prints to stdout; the fixed-player fixture belongs in the adapter's tests, which is
  where it goes. `POST /scores` covers everything it did.
- `POST /scores` moves with its behaviour byte-identical, and is marked in code and README as
  interim — it exists for `scripts/scores.sh` and `scripts/fantasycast.sh` and retires with them.
- Consumers depend on a small interface for fetching a week, not on `internal/sleeper`. Only
  `cmd/server` names the concrete adapter.

No scoring rule, no stat mapping, and no `POST /scores` request or response shape changes. The shell
scripts keep working untouched.

## Capabilities

### New Capabilities

None. This is a restructuring of code that already implements existing specs.

### Modified Capabilities

- `player-week-score`: the `GET /score` endpoint requirement is removed, and a requirement is added
  for the weekly snapshot value — that one fetch yields a provider-neutral snapshot of the whole
  week from which any player can be read, with absence still reported as a value rather than an
  error.

## Impact

- `internal/score`: `sleeper.go`, `handler.go`, `batch.go` and their tests leave the package.
  `stats.go` and `calc.go` stay, joined by `week.go`.
- New `internal/sleeper` and `internal/api` packages, each carrying its existing tests.
- `cmd/server/main.go`: becomes the composition root — it constructs the adapter and registers the
  one remaining route.
- `internal/roster`: untouched.
- `scripts/**`: untouched, and `POST /scores` must stay wire-compatible for them.
- `openspec/changes/serve-matchup-page`: its design names `score.FetchWeek`, which becomes
  `sleeper.FetchWeek` returning `score.Week`. That change is updated to sit on top of this one.
