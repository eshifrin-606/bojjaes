## Why

The rosters the tool already scores contain kickers and defenders — `scripts/bojjaes.csv` lists
Harrison Butker, Odafe Oweh, and Derrick Harmon today — and every one of them comes back as a
well-formed `0`. A silent zero is worse than an error here, because it reads as a real result and
quietly disagrees with the league spreadsheet.

The same gap reaches the offensive players already believed to be scored correctly: the league's
"TD scored" rule is unqualified, so a rostered receiver who returns a punt should be paid 6 and is
currently paid nothing.

## What Changes

- Score **kicking**: 3 points per field goal made, 1 per extra point, and the 1-point bonus for a
  field goal of 50+ yards.
- Score the **defensive rules reachable from the provider's weekly aggregate**: 6 per interception
  caught, 3 per sack (half-sacks included, paying 1.5), and 2 per fumble recovery that results in a
  turnover.
- **Finish the unqualified touchdown rule.** Defensive touchdowns and kick/punt-return touchdowns
  join passing, rushing, and receiving touchdowns in the 6-point term. This corrects an existing
  underpayment on offensive players, not only on defenders.
- **Model sacks as a fractional stat.** Every stat in the domain object is currently a whole count;
  sacks are recorded in half-sack granularity and must not be truncated.
- **Deliberately leave three rules unscored**, per the aggregate-stage deviations recorded in
  `docs/scoring.md`: forced fumbles, safeties, and the 40+ yard bonus on defensive and return
  touchdowns. Each depends on play-by-play data this change does not introduce.
- **Name the provider stats that must not be scored.** Several plausible-looking stats are credited
  to the wrong player — most damagingly, the quarterback who *threw* a pick-six carries an
  interception-returned-for-touchdown stat. Scoring it would pay him 6 for a play the rules dock him
  3 for. A second one is quieter and just as wrong: the provider's generic fumble-recovery stat holds
  *own-team* recoveries, which are not turnovers, and excluding it is exactly what makes the
  recovery keys this change does score turnover-qualified.

No breaking changes, but the response does change: it gains stat fields, gains its first
non-integer value (sacks), and players who previously scored 0 may now score more. Nothing
downstream reads the new keys — `scripts/scores.sh` reads only `points` — so existing consumers are
unaffected, but "unchanged" would be too strong.

## Capabilities

### New Capabilities

None. This extends the existing scoring capability rather than introducing a new one.

### Modified Capabilities

- `player-week-score`: The capability's stated scope excludes kicking and defensive scoring
  ("kicking and defensive scoring are not yet specified"). That exclusion is lifted in part. New
  requirements cover kicking points, the aggregate-reachable defensive points, and the completed
  touchdown rule; the stat-line and transform requirements change to carry the new stats, including
  the first fractional one.

## Impact

- `internal/score/stats.go` — the domain stat line gains kicking, defensive, and remaining
  touchdown stats, and its first non-integer field.
- `internal/score/calc.go` — `Points` gains terms. It reads stats rather than positions, so no
  branching on player position is introduced.
- `internal/score/sleeper.go` — the transform maps the new provider keys, and needs a fractional
  reader alongside the existing integer one.
- `internal/score/testdata/week14.json` — the existing fixture holds six offensive players and
  exercises none of the new rules.
- `README.md` — the documented example response gains fields.
- No new dependencies, no new provider surface, and no additional network calls: the existing single
  weekly fetch already carries every stat this change scores.
