---
title: 2026-10-04-sleeper-com-stats-endpoint-is-row-array-with-names-teams-and-shorter-edge-ttl
type: note
permalink: bojjaes-memory/basic-memory/2026-10-04-sleeper-com-stats-endpoint-is-row-array-with-names-teams-and-shorter-edge-ttl
tags:
- sleeper
- freshness
- endpoint
---

# Sleeper .com stats endpoint: row array with names and game-time teams, shorter edge TTL

## Observations
- [finding] `https://api.sleeper.com/stats/nfl/{season}/{week}?season_type=regular` returns the same stats as `api.sleeper.app/v1/stats/nfl/regular/{season}/{week}`, as a row array carrying `team` and `player{first_name, last_name, ...}`. Checked 2026-10-04: 2026 wk 3 has 2357 players on both with identical `stats`; 2025 wk 14 identical. #parity
- [finding] An unplayed week is `{}` from `.app` and `[]` from `.com`.
- [finding] Payload is ~2.1 MB raw / ~280 KB gzipped vs ~0.57 MB / ~85 KB; latency differs ~30 ms.
- [gotcha] For 36 wk 3 rows the row `team` differs from `player.team`. Row `team` is the team at game time; `player.team` is current. Use the row's for forced-fumble attribution.
- [finding] Edge cache (`s-maxage` / `stale-while-revalidate`, s): current week `.app` 30/300 vs `.com` 4/600; last completed 600/300 vs 300/600; old week 3600/300 vs 3600/600. #freshness
- [tradeoff] `.com` is fresher at the edge, but the 600 s stale-while-revalidate means a rarely read week can be served up to ~10 min stale once.

## Relations
- extends [[2026-08-09-sleeper-stats-endpoint-has-1h-edge-ttl-espn-is-seconds]]
- relates_to [[0003-sleeper-as-initial-stat-provider]]
- relates_to [[2026-08-15-sleeper-weekly-stats-absence-is-routine-ambiguous-and-sometimes-stale]]
