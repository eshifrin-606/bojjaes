## Why

`score-forced-fumbles` needs each player ID's name and team at game time. The REST plays feed's rows
carry only `player_id`, and so does the weekly stats endpoint we call today,
`api.sleeper.app/v1/stats/nfl/regular/S/W`, which returns a map of player ID to stats.
`api.sleeper.com/stats/nfl/S/W?season_type=regular` returns the same stats as an array of rows that
also carry `team` and `player{first_name, last_name, ...}`. Switching endpoints on its own, before
forced fumbles, keeps a change that should move no scores separate from one that does.

Checked live on 2026-10-04:

- 2026 wk 3: both endpoints return 2357 players, and every player's `stats` object is identical.
- 2025 wk 14: every player in `.app` has identical stats in `.com`.
- An unplayed week returns `{}` from `.app` and `[]` from `.com`.
- The payload is ~2.1 MB raw / ~280 KB gzipped, against ~0.57 MB / ~85 KB, and the response time
  differs by ~30 ms.
- For 36 wk 3 rows, the row's `team` differs from `player.team`. The row's `team` is the team at game
  time; `player.team` is the player's current team.

## What Changes

- `sleeper.Client` reads weekly stats from `api.sleeper.com/stats/nfl/S/W?season_type=regular`.
  `BaseURL` becomes `https://api.sleeper.com`.
- The decoder reads the row array. Rows with non-numeric `player_id` (team rows such as `GB` and
  `TEAM_GB`) still produce stat lines exactly as their map entries did before; nothing is filtered
  that was not filtered before.
- The stat-key mapping, the absence semantics, and every score are unchanged.
- `testdata/week14.json` is converted to the row-array shape, entry for entry, including the
  hand-built entries, so existing expectations hold without re-deriving them.
- The server's only upstream becomes `https://api.sleeper.com`. The forced-fumble plays feed is on
  the same host, so it stays the only upstream after that change too.
- Docs amended in place: ADR 0003, `docs/architecture.md`, and the cache-header finding (below).

Names and teams are not exposed from the client in this change. The forced-fumble change adds them
to the row type when it has a consumer.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `server-image`: the upstream whose CA roots the image carries changes from
  `https://api.sleeper.app` to `https://api.sleeper.com`.

## Impact

- `internal/sleeper`: `sleeper.go`, its tests, `bench_test.go`, `testdata/week14.json`.
- `cmd/server/main.go` is untouched; it reads `sleeper.BaseURL`.
- Freshness: the edge cache differs by endpoint. Measured `s-maxage` / `stale-while-revalidate`:

  | Week | `.app` | `.com` |
  |---|---|---|
  | current (2026 wk 4) | 30 / 300 | 4 / 600 |
  | last completed (2026 wk 3) | 600 / 300 | 300 / 600 |
  | old (2025 wk 14) | 3600 / 300 | 3600 / 600 |

  Fresher at the edge, but the stale-while-revalidate window is longer, so a rarely read week can be
  served up to ~10 min stale once before it refreshes.
- Docs: `docs/adr/0003-sleeper-as-initial-stat-provider.md`, `docs/architecture.md`,
  `openspec/specs/server-image/spec.md` Purpose.
