---
title: 2026-08-09-sleeper-idp-ff-is-raw-idp-fum-rec-is-turnover-qualified
type: note
permalink: bojjaes-memory/basic-memory/2026-08-09-sleeper-idp-ff-is-raw-idp-fum-rec-is-turnover-qualified
tags:
- sleeper
- idp
- fumbles
- scoring
- validation
- nflverse
- data-source
---

# 2026-08-09 Sleeper `idp_ff` is raw, `idp_fum_rec` is turnover-qualified

Resolves open question 1 of [[probe-espn-sleeper]], which was flagged there as the top blocker
on the source decision in [[0002-live-scoreboard-backend]]. The probe suspected **both**
`idp_ff` and `idp_fum_rec` were unqualified raw counts. Half right.

## Method

Validated against nflverse play-by-play for the **entire 2025 regular season**, not a single
week. Turnover qualification read from `fumble_lost` (FF) and
`fumble_recovery_N_team != fumbled_1_team` (FR).

The join is the part that needed care. Two traps, both consistent with the name-matching hazards
already recorded in [[probe-espn-sleeper]] Tier 4:

- Sleeper's `team` field is the player's **current** roster team, not the team they played for in
  the probed season. Joining on name+team silently dropped 5 of 20 week-1 FF players.
- 83 exact full-name collisions among active players. The fix was to restrict the comparison to
  names mapping to exactly one Sleeper entry — this discards some volume but makes every
  surviving row trustworthy.

## Observations

- [finding] `idp_ff` matches **all** FF events 201/203 (99%) and turnover-qualified FF only
  104/203 (51%). It is a raw count. #sleeper #idp
- [magnitude] 2025 season: 366 forced fumbles, 204 turnover-qualified. Paying HMFFL's 4-point FF
  rule off `idp_ff` overpays **162 forced fumbles a season, 44% of them**. Too big to accept
  silently. #scoring
- [finding] `idp_fum_rec` counts only recoveries by the non-fumbling team — effectively
  turnover-qualified, contradicting the probe's suspicion. #sleeper #idp
- [finding] `idp_fum_rec` + `st_fum_rec` + `def_st_fum_rec` reproduces the turnover-qualified
  recovery set **268/269 (99.6%)**. All 19 misses against the IDP key alone were *undercounts* —
  special-teams recoveries booked to the ST keys. #sleeper
- [finding] Own-team recoveries land in the non-IDP `fum_rec` key, which is what keeps the IDP
  key clean. #sleeper
- [consequence] Sleeper now clears **4.5 of the 5 hard rules** from aggregates, not 4. The only
  rule it cannot express at any accuracy is the 4-point FF turnover rule — turnover qualification
  is a property of the play, not of a player's stat line, so no aggregate key can carry it.
  #data-source
- [open-rules-question] wk 14 Troy Dye (LAC) recovered his own team's fumble after an
  interception return; Sleeper credited `idp_fum_rec=1` though nflverse says no fumble-turnover
  occurred (the INT was the turnover). Needs a league ruling, not more data. Bundle with the
  `idp_safe` solo-credit question. #scoring
- [method-note] A one-week sample could not have distinguished "turnover-qualified" from merely
  "position-gated to defenders" for `idp_fum_rec` — in week 1 no defensive player recovered an
  own-team fumble on a scrimmage play. The full-season pass is what separated the two
  hypotheses. #validation

## Relations

- resolves_question_in [[probe-espn-sleeper]]
- informs [[0002-live-scoreboard-backend]]
- validates_against [[scoring]]

## What this changes about the decision

The remaining FF gap is now priced rather than unknown, which shifts the choice. "Accept a
known-wrong edge case on two low-frequency rules" was a reasonable-sounding fallback when the
gap was unmeasured; at 162 overpaid events a season on one rule it is much less attractive. A
narrow ESPN play-by-play supplement scoped **only to fumble plays** looks better than it did —
roughly 20 such plays a week, small enough that the name+team+position join stays tractable even
without the `espn_id` bridge that Tier 4 showed is unusable at scale (37% coverage).

Still blocking the source decision: Tier 3 freshness (needs live games), `idp_safe` semantics,
and the `*_td_40p` inclusive-bound check.

## Outcome (2026-08-09)

Superseded as a blocker. [[0003-sleeper-as-initial-stat-provider]] chose Sleeper alone and took
neither of the two options weighed above: instead of accepting the overpay silently or building
the ESPN supplement up front, it **awards the 4 points but flags them as provisional** in the
served output. The reasoning is that the error's danger is its systematic, one-directional
disagreement with the league spreadsheet — making it visible defuses that at near-zero build cost.
The ESPN fumble-play supplement stays on the roadmap as verification, not as a second provider.
