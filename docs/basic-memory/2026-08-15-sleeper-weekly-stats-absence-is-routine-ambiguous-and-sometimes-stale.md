---
title: 2026-08-15-sleeper-weekly-stats-absence-is-routine-ambiguous-and-sometimes-stale
type: note
permalink: bojjaes-memory/basic-memory/2026-08-15-sleeper-weekly-stats-absence-is-routine-ambiguous-and-sometimes-stale
tags:
- sleeper
- stats-endpoint
- absence
- gotcha
- schedule-endpoint
- scoring
- api-design
---

Measured while designing the multi-player scoring endpoint. The question was narrow — what should
`/v1/stats/nfl/{type}/{season}/{week}` mean when a requested player ID is not in the payload? — and
the answer overturned an assumption already encoded in our code.

Extends the endpoint characterisation in [[2026-08-08-espn-vs-sleeper-stat-source-probe]] and
[[probe-espn-sleeper]]. Staleness interacts with [[2026-08-09-sleeper-stats-endpoint-has-1h-edge-ttl-espn-is-seconds]].
The schedule endpoint below is directly relevant to [[0002-live-scoreboard-backend]].

## Observations

- [finding] **Absence in the weekly stats payload is routine, not exceptional.** In *completed*
  2025 regular week 14, 28 active QB/RB/WR/TE were absent entirely — inactives and healthy
  scratches. Treating absence as an error reports valid player IDs as server failures. #gotcha
  #sleeper

- [finding] **A player who dressed and recorded nothing is present**, carrying a stub entry with no
  stat keys: `{gms_active: 1, tm_off_snp: 60, tm_def_snp: 63, tm_st_snp: 27, pos_rank_*: 999}`.
  Missing keys read as zero, so scoring a genuine 0.0 needs no special handling. #sleeper

- [gotcha] **`gp` does not mean "played."** T.J. Hockenson (MIN) carried `gp: 1` with no stats while
  Minnesota had not kicked off. Sleeper seeds part of the slate before games start. Anyone reaching
  for `gp` as a did-they-play signal will be wrong for pre-kickoff players. #gotcha #sleeper

- [finding] **A player whose game has not started is usually absent, but not always.** Of 353 active
  skill players on `pre_game` teams in 2026 preseason week 1: 244 absent, 109 present with a seeded
  stub, 0 with stats. Absence is a timing artifact, not a fact about the player. #sleeper

- [finding] **A week that has not happened returns `200 {}`** — an empty object, not a 404. So "week
  not played" looks like every player missing rather than like an error. #sleeper

- [finding] **The payload cannot distinguish a bad player ID from a player who has not played.**
  Both are simply absent. Only the player index (`/v1/players/nfl`, **14.6 MB**) can validate an ID,
  and it needs a cache and a daily refresh to be usable. #api-design

- [finding] **`/schedule/nfl/{season_type}/{season}` is the cheap disambiguator** — **4.7 KB**,
  one row per game with `status` of `complete` / `in_game` / `pre_game`, plus `date`, `home`,
  `away`, `week`. Combined with a player→team map it separates "game hasn't started" from
  "inactive" without touching the 14.6 MB index for that purpose. Useful to
  [[0002-live-scoreboard-backend]] independently of scoring. #schedule-endpoint

- [gotcha] **Absence can also be *stale*.** Per
  [[2026-08-09-sleeper-stats-endpoint-has-1h-edge-ttl-espn-is-seconds]], the stats endpoint carries a
  one-hour edge TTL. A player whose game just finished can read absent, or read with partial stats,
  purely because of the CDN. So absence has at least four causes: not kicked off, inactive, unknown
  ID, and cache lag. #gotcha #freshness

- [gotcha] **Non-numeric IDs share the payload, in two shapes.** Confirming and extending the
  `TEAM_*` gotcha in [[2026-08-08-espn-vs-sleeper-stat-source-probe]]: both `TEAM_BUF` *and* bare
  abbreviations like `HOU`, `NE`, `BAL` appear as keys. 48 non-numeric keys in 2026 preseason week 1,
  56 in 2025 regular week 14. Player IDs are numeric strings; anything else is an aggregate row.
  #gotcha

- [decision] The multi-player endpoint reports absent players in a separate `no_stats` bucket of IDs
  and returns 200, rather than erroring or substituting a zero stat line. It needs no new machinery,
  never invents a score, and never rejects valid input; the caller determines root cause. Recorded in
  the `add-multi-player-week-score` change design. #api-design

- [method] The cross-reference is reproducible for any in-progress week, but the *measurement* is
  not — once preseason week 1 completes, its `pre_game` rows can never be observed again. Re-run
  during a live Sunday to characterise the regular season:

  ```python
  import json, urllib.request, collections
  def get(u): return json.load(urllib.request.urlopen(u))
  B = "https://api.sleeper.app"
  TYPE, SEASON, WEEK = "pre", 2026, 1
  P = get(f"{B}/v1/players/nfl")
  S = get(f"{B}/schedule/nfl/{TYPE}/{SEASON}")
  D = get(f"{B}/v1/stats/nfl/{TYPE}/{SEASON}/{WEEK}")
  status = {}
  for g in (g for g in S if g["week"] == WEEK):
      status[g["home"]] = status[g["away"]] = g["status"]
  rows = collections.Counter()
  for pid, p in P.items():
      if p.get("position") not in ("QB", "RB", "WR", "TE"): continue
      if not p.get("team") or p.get("status") != "Active": continue
      e = D.get(pid)
      kind = "absent" if e is None else "stats" if any(
          k.startswith(("rush_", "rec_", "pass_")) for k in e) else "stub"
      rows[(status.get(p["team"], "BYE"), kind)] += 1
  for k in sorted(rows): print(k, rows[k])
  ```
  #method

## Cross-reference: 2026 preseason week 1, measured 2026-08-15

Active QB/RB/WR/TE only. Game status from `/schedule/nfl/pre/2026`; 18 teams `complete`, 12
`pre_game`, 2 `in_game`.

| game status | absent | entry, no stats | entry with stats |
| --- | --- | --- | --- |
| `pre_game` | 244 | 109 | 0 |
| `in_game` | 2 | 1 | 54 |
| `complete` | 28 | 124 | 388 |

Payload sizes: 2026 preseason wk1 = 514 KB / 2290 keys. 2025 regular wk14 = 2142 keys. Future weeks
(2026 regular wk1, wk18) = `{}`, 2 bytes.

## Open questions

- Does the pre-kickoff seeding ratio hold in the regular season, or is it a preseason artifact of
  unsettled rosters? Re-run the script above on a live Sunday.
- Is `tm_off_snp` / `tm_def_snp` / `tm_st_snp` presence a reliable "this player's game has been
  played" signal? It held for 117 of 124 `complete` stubs and for 0 of 109 `pre_game` stubs, but it
  is undocumented and had 7 exceptions.

## Relations

- extends [[2026-08-08-espn-vs-sleeper-stat-source-probe]]
- extends [[probe-espn-sleeper]]
- relates_to [[2026-08-09-sleeper-stats-endpoint-has-1h-edge-ttl-espn-is-seconds]]
- relates_to [[0002-live-scoreboard-backend]]
- relates_to [[0003-sleeper-as-initial-stat-provider]]

---

Informative, not authoritative. The absence contract it drove is specified in
`openspec/specs/player-week-score/spec.md`; that spec is authoritative for what we build.
