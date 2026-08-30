---
title: 2026-08-20-sleeper-aggregate-td-and-defensive-keys-sit-on-the-wrong-rows
type: note
permalink: bojjaes-memory/basic-memory/2026-08-20-sleeper-aggregate-td-and-defensive-keys-sit-on-the-wrong-rows
tags:
- sleeper
- idp
- kicking
- scoring
- gotcha
- attribution
- data-source
---

# 2026-08-20 Sleeper aggregate: TD and defensive keys sit on the wrong rows

Key-level reconnaissance done while scoping kicking and defensive scoring for the
player-week-score feature. The question going in was "do we have the data for a pick-six?" The
answer is yes — but four plausible-looking keys are attached to the wrong player, and picking one
of them produces a large, confident, wrong number on a memorable play.

Same family of hazard as the `idp_ff` feed inversion in
[[2026-08-12-sleeper-graphql-is-the-fresh-surface-and-has-play-by-play]], and it extends the
`TEAM_*` row gotcha from [[2026-08-08-espn-vs-sleeper-stat-source-probe]] and
[[2026-08-15-sleeper-weekly-stats-absence-is-routine-ambiguous-and-sometimes-stale]].

## Method

One live fetch of the REST weekly aggregate, 2025 regular season week 14 — 2,142 entries, 512 KB.
Enumerated every key, then pulled the full stat line of each player holding a TD-flavored,
kicking, or IDP key. Week 14 happens to contain **exactly one** pick-six, one `idp_def_td`, one
`def_td`, and one `idp_safe`, so attribution was readable directly rather than statistically.

This is a one-week sample. It is enough to establish *which row a key sits on* (a structural fact)
and not enough to establish frequencies or qualification semantics — the distinction that
[[2026-08-09-sleeper-idp-ff-is-raw-idp-fum-rec-is-turnover-qualified]] needed a full season to
draw.

## Observations

- [gotcha] `pass_int_td` sits on the **passer**, not the defender. Player 6770 (Joe Burrow) carries
  `pass_int: 2` and `pass_int_td: 1` — "I threw a pick-six." Feeding it into an unqualified TD term
  pays the quarterback +6 for the play the league rules dock him 3 for. A 9-point error in the
  wrong direction. #gotcha #attribution
- [finding] The pick-six **defender** (8487) carries `idp_int: 1` + `idp_def_td: 1`, which is
  exactly the 6 + 6 = 12 that [[scoring]]'s "TD scored is unqualified" clarification calls for. The
  rule is satisfiable from the aggregate alone. #idp
- [gotcha] `kr_td`, `pr_td`, `def_td`, and `safe` appear **only on team rows** in week 14
  (`TEAM_SEA`, `TEAM_NYJ`, `TEAM_DEN`, `BUF`, `JAX`). They are useless for scoring a rostered
  individual. #gotcha
- [finding] `st_td` is the **per-player** kick/punt-return touchdown key — three players held it in
  week 14, each alongside `kr_yd`/`pr_yd`. This is the key that finishes the unqualified-TD rule
  for offensive players, who can and do return kicks. #scoring
- [gotcha] `pass_sack` (50 players) is sacks **taken** by a quarterback; `idp_sack` (56) is sacks
  **recorded** by a defender. Only the latter pays. Same passer-side/defender-side split as
  `pass_int` vs `idp_int`. #gotcha #attribution
- [finding] `anytime_tds` is **not** a shortcut for the unqualified TD rule — it covers offense and
  special teams only. The week-14 pick-six defender has `idp_def_td` but no `anytime_tds`. #scoring
- [finding] `idp_sack` carries genuine half-sack granularity in the aggregate, so a stat line
  modelling sacks as an integer silently truncates 0.5 to 0 and loses 1.5 points. First stat in
  this domain that is not a whole count. #scoring
- [finding] `fgm_50p` equals `fgm_50_59` + `fgm_60p` with **zero mismatches** across week 14, so
  the FG 50+ bonus needs one key, not a sum. Kicking overall is the cleanest category in the
  ruleset: `fgm`, `xpm`, `fgm_50p` and nothing else. #scoring
- [finding] `idp_int_ret_yd` (21 players) and `idp_fum_ret_yd` (3) **are present in the REST
  aggregate**, not only in play-by-play as [[0003-sleeper-as-initial-stat-provider]]'s amendment
  implies. They still do not close the 40+ defensive-TD bonus, but for a different reason than
  stated there: the yardage is a **weekly sum**, so a defender with two interception returns has no
  way to attribute distance to the one that scored. The obstacle is aggregation, not absence.
  #data-source
- [gotcha] `st_fum_rec` and `def_st_fum_rec` had **zero occurrences** in week 14. They are sparse
  keys that materialise only when the play happens, so the three-key turnover-qualified recovery
  sum from [[2026-08-09-sleeper-idp-ff-is-raw-idp-fum-rec-is-turnover-qualified]] cannot be
  exercised by a fixture built from an arbitrary week. Pick the week for the branch, or hand-build
  the fixture. #validation

## Relations

- extends [[2026-08-08-espn-vs-sleeper-stat-source-probe]]
- extends [[2026-08-15-sleeper-weekly-stats-absence-is-routine-ambiguous-and-sometimes-stale]]
- corrects [[0003-sleeper-as-initial-stat-provider]]
- relates_to [[2026-08-12-sleeper-graphql-is-the-fresh-surface-and-has-play-by-play]]
- relates_to [[2026-08-09-sleeper-idp-ff-is-raw-idp-fum-rec-is-turnover-qualified]]
- validates_against [[scoring]]

## What this changes

The unqualified-TD rule is now specifiable as an explicit allowlist of five keys —
`pass_td`, `rush_td`, `rec_td`, `idp_def_td`, `st_td` — with `pass_int_td`, `kr_td`, `pr_td`,
`def_td`, `td`, and `anytime_tds` named as deliberate exclusions. An allowlist matters more than
usual here because every excluded key is individually plausible; a future reader adding "the
obvious missing TD stat" reintroduces the quarterback bug.

It also means the existing offensive scoring path is **already incomplete**, independently of
defense: a rostered wide receiver who returns a punt scores 0 for it today, because `st_td` is not
mapped. "Score defensive points" and "finish the TD rule" are the same piece of work.

Promoted to [[scoring]] on 2026-08-20 as an "Attribution hazards" section, and the same edit
rescoped that document's implementation deviations from standing positions to
aggregate-stage-only ones. Forced fumbles moved from "pay it, flag it provisional" to **excluded
entirely** — the provisional-flag plan predated the finding that turnover qualification is a
property of the play and unreachable from any aggregate key, and a visible omission is preferred
to a knowingly wrong award, matching how safety is already handled.
