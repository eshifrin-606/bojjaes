## Why

`scripts/scores.sh` and `scripts/fantasycast.sh` were a terminal UI to use until the matchup page existed. The page exists and is deployed, but the scripts and the `POST /scores` endpoint built for them are still here. Design decisions keep having to ask "could this break the scripts?" (the lineup CSV field order in the backlog and ADR 0004 is the current example). Removing both ends that dual purpose. The endpoint's own doc comment already says it "retires with them when the matchup page replaces them".

## What Changes

- **BREAKING** Remove `scripts/scores.sh`, `scripts/fantasycast.sh`, and the stale `scripts/teams/*.csv` rosters. The real lineups live in `internal/lineup/data/`.
- **BREAKING** Remove the `POST /scores` endpoint: the `internal/api` package (handler, DTOs, validation, the 26-player cap) and its route in `cmd/server/mux.go`.
- Remove `statscache.Cache.WeekStats`. Its only production caller was `internal/api`, and the page reads through `WeekStatsAsOf`.
- Retire the `lineup-score-report` and `matchup-report` capabilities. Both describe terminal output only. The behaviours the page still depends on (the first nine records are starters, a starter with no stats shows no number and adds nothing to the total, no margin) are already stated in `lineup-source` and `matchup-page`.
- The server now has one route, so the mux requirement drops `POST /scores` and shows its method-qualified 405 on the matchup route instead.
- Amend docs in place so none of them describe the scripts or the endpoint as current: README, `docs/architecture.md` §5, `docs/package-dependencies.md`, ADR 0004, and `backlog.md`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `http-server-lifecycle`: the owned mux serves only `GET /{season}/{week}`; the wrong-method scenario moves from `/scores` to the matchup route.
- `lineup-score-report`: all requirements removed. The capability retires with `scripts/scores.sh`.
- `matchup-report`: all requirements removed. The capability retires with `scripts/fantasycast.sh`.
- `player-week-score`: the "Multi-player score endpoint" and "Multi-player request validation" requirements are removed. The scoring rules and the weekly stat snapshot stay.

## Impact

- **Code removed**: `scripts/`, `internal/api/`, `Cache.WeekStats`.
- **Code changed**: `cmd/server/mux.go` (a single stats dependency instead of a combined interface), `cmd/server/mux_test.go`, and `internal/statscache/cache_test.go` (drop the `api.StatsSource` assertion; tests call `WeekStatsAsOf`).
- **HTTP surface**: `POST /scores` returns 404 afterwards. It has one path segment, so it cannot match `/{season}/{week}`. Nothing else is known to call it.
- **Docs**: README, architecture, package-dependencies, ADR 0004, backlog.
- **Active change**: `refresh-page-while-visible` says `scripts/**` is untouched. That stays true, but the reference will be stale on paper. It needs no edit.
- **Not affected**: `internal/score`, `internal/sleeper`, `internal/lineup`, `internal/web`, the roster CSV format, deploy config.
