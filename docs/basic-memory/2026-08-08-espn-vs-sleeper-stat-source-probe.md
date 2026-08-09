---
title: 2026-08-08-espn-vs-sleeper-stat-source-probe
type: note
permalink: bojjaes-memory/2026-08-08-espn-vs-sleeper-stat-source-probe
tags:
- nfl-data
- espn
- sleeper
- provider
- adr-0002
- scoring
- player-id-mapping
---

# 2026-08-08 ESPN vs Sleeper stat-source probe

Offseason probe of the two free candidates from ADR 0002 (`docs/adr/0002-live-scoreboard-backend.md`),
run against 2025 regular-season data. Full write-up with tables lives in `docs/probe-espn-sleeper.md`
— this note records the *discoveries and reversals*, not the raw inventory.

Tiers 1 (raw-stat coverage), 2 (play-level detail) and 4 (player-ID mapping) are complete.
Tier 3 (live freshness) is blocked until the season starts.

## Two expectations inverted

Going in, the working assumption was: Sleeper is the ergonomic ID-mapping layer, ESPN is the
richer stat source, and Sleeper's bonus buckets would be 50+ and useless for HMFFL's 40+ TD
bonus. Both halves of that turned out backwards.

## Observations

- [discovery] Sleeper pre-computes distance-bucketed bonus stats at **exactly** HMFFL's
  thresholds: `pass_td_40p`, `rec_td_40p`, `rush_td_40p` (40+, not 50+), plus `fgm_50p` /
  `fgm_50_59` / `fgm_60p`. This clears 4 of the 5 "needs play-by-play" rules from aggregates
  alone. #discovery
- [discovery] Sleeper carries the 40+ bucket **separately for passer and receiver**, which
  exactly matches the confirmed HMFFL rule that a 45-yard TD pass pays both the QB and the
  receiver +1. No derivation needed. #discovery
- [failed-assumption] The plan to use Sleeper's cross-ID dictionary as the mapping bridge to
  ESPN does not work. `espn_id` is populated for only **37% of players with scoreable Week 1
  production** (170/458) — and the gaps are starters, not fringe: Brandon Aubrey, James Cook,
  Kyle Pitts, Khalil Shakir, Bucky Irving, Omarion Hampton. #failed-assumption
- [failed-assumption] `gsis_id` is only ~31% populated, which also weakens the plan to use
  nflverse (GSIS-keyed) as a clean validation oracle. Validation will need name-based joins.
- [discovery] `sportradar_id` is the only near-complete cross-ID at 99.5%, but ESPN does not
  expose Sportradar IDs — so it does not bridge *this* pair. It may matter if a future provider
  does. #discovery
- [gotcha] Sleeper uses **sparse keys**: a stat absent from the payload means zero, not an
  error. `idp_safe` / `safe` / `rush_2pt` are simply missing in weeks where they did not occur
  (absent wk 1, 2, 9; present wk 5, 14, 18). A strict parser will break on this. #gotcha
- [gotcha] Sleeper's weekly stats feed mixes **team aggregate rows** (`TEAM_BUF`, `TEAM_LV`)
  in with player rows. Filter them before scoring. #gotcha
- [gotcha] Sleeper **strips name suffixes** ("Marvin Harrison", "Michael Penix", "Travis
  Etienne" — only 6 suffixed names in 12,217 entries) while ESPN **keeps** them ("Kenneth
  Murray Jr."). Combined with 83 exact `full_name` collisions among active roster-eligible
  players, name-only joins are unsafe — use name + position + team. #gotcha
- [discovery] ESPN's core plays endpoint is better than its reputation: a whole game returns
  unpaginated at `limit=400`, plays carry numeric `statYardage`, and participants are **typed by
  role** (`passer`, `receiver`, `rusher`, `scorer`, `forcedBy`, `recoverer`, `sackedBy`,
  `tackler`…) which maps almost 1:1 onto HMFFL's rule set. #discovery
- [ergonomics] ESPN participant athlete refs are `$ref` URLs, but the athlete ID is embedded in
  the path (`.../athletes/4361579?lang=en`) and can be regexed out without a second request.
  The usual ESPN `$ref`-chasing tax is avoidable.
- [gotcha] Some ESPN `$ref` values point at the internal host `sports.core.api.espn.pvt`
  instead of `.com`. Harmless when only parsing IDs out of them; broken if dereferenced. #gotcha
- [discovery] The `summary?event=` endpoint has **no** athlete participants (team-level only) —
  the typed participants exist only on the `sports.core.api` plays endpoint. Easy to conclude
  ESPN is text-only if you probe just the summary endpoint. #discovery
- [verification] Cross-validation between the two sources was clean on every value spot-checked:
  half-sacks (Gary 1.5, Van Ness 0.5, Bowman 0.5), whole sacks (Parsons 1.0), and Aubrey's
  kicking line. Both sources represent 0.5 sacks correctly, which the 1.5-point HMFFL rule
  requires.
- [discovery] ESPN's box score **cannot** support the FG 50+ bonus, confirmed empirically:
  Aubrey's 41- and 53-yard makes collapse to `FG 2/2, LONG 53`. Distances only exist in PBP.
- [cost] Sleeper serves the entire league's weekly stats in **one** ~570 KB / 1.8 s call. ESPN
  needs ~13 summary calls plus ~13 PBP calls per Sunday slate. Real advantage to Sleeper on
  complexity. **Corrected 2026-08-09: the *freshness* half of this claim does not hold up** —
  Sleeper's stats endpoint carries a one-hour Cloudflare edge TTL while ESPN's live surfaces
  carry 1–8 seconds. See
  [[2026-08-09-sleeper-stats-endpoint-has-1h-edge-ttl-espn-is-seconds]].
- [open-question] Sleeper's `idp_ff` / `idp_fum_rec` appear to be **raw counts, not
  turnover-qualified**. HMFFL pays 4 only for an FF *that results in a turnover* and 2 for a
  recovery *that results in a turnover*. This is the single gap where ESPN's PBP (via
  `forcedBy` + `recoverer` + `isTurnover` on the same play) does something Sleeper's aggregates
  cannot. Top blocker on the provider decision. #open-question
- [decision-input] ADR 0002 lists documentation quality as a selection criterion. It should be
  dropped to near-zero weight: Sleeper's stats endpoint is **undocumented** (its public docs
  cover leagues/drafts/players only), so both candidates are unofficial surfaces.

## Current leaning

Sleeper as the primary provider; the hybrid "Sleeper for IDs, ESPN for stats" design from
ADR 0002 is rejected as originally conceived because the ID bridge does not exist. Resolving the
FF/FR turnover question decides whether a narrow ESPN PBP supplement is worth the fuzzy-join
cost, or whether we accept a known-wrong edge case on two low-frequency rules.

## Relations

- relates_to [[2026-07-10-mcp-vs-skills-agents-commands-portability]]
- supersedes_assumption_in [[0002-live-scoreboard-backend]]
- narrowed_by [[2026-08-09-sleeper-idp-ff-is-raw-idp-fum-rec-is-turnover-qualified]]
- narrowed_by [[2026-08-09-sleeper-stats-endpoint-has-1h-edge-ttl-espn-is-seconds]]

---

Informative, not authoritative. The decision itself belongs in ADR 0002 or a successor once
Tier 3 runs against live games.
