## ADDED Requirements

### Requirement: Kicking fantasy points

The system SHALL compute HMFFL fantasy points for kicking production from a domain stat line, using
the kicking rules in `docs/scoring.md`:

- **3 points** per field goal made, regardless of distance. The league pays a flat rate; distance
  bands do not change the award.
- **1 point** per extra point made.
- **1 bonus point** per field goal made from 50+ yards, under the same Misc bonus clause that pays
  the 40+ yard touchdown bonus.

Missed field goals and missed extra points SHALL NOT affect a player's score. The league's tables
carry no penalty for them, so a miss pays nothing rather than costing anything.

#### Scenario: Field goals pay a flat rate

- **WHEN** a player makes 3 field goals of differing distances, none of them 50+ yards
- **THEN** the score is 9 — 3 points each, with distance affecting nothing

#### Scenario: Extra points

- **WHEN** a player makes 4 extra points and no field goals
- **THEN** the score is 4

#### Scenario: Long field goal bonus

- **WHEN** a player makes 2 field goals, one of which was from 50+ yards
- **THEN** those kicks contribute 7 points — 3 each plus 1 for the long field goal

#### Scenario: Misses are not penalised

- **WHEN** a player makes 1 field goal and misses 2, and misses an extra point
- **THEN** the score is 3 — the misses neither add nor subtract

#### Scenario: A player with no kicking production is unaffected

- **WHEN** a stat line carries no kicking stats
- **THEN** the score is exactly what the other rules alone produce

### Requirement: Defensive fantasy points

The system SHALL compute HMFFL fantasy points for individual defensive production from a domain stat
line, using the subset of the defensive rules in `docs/scoring.md` that the provider's weekly
aggregate can express:

- **6 points** per interception caught.
- **3 points** per sack. Sacks are credited in half-sack granularity, and a half sack SHALL pay 1.5
  points. The award is proportional rather than tabulated, so any fractional credit the provider
  reports pays its proportional share.
- **2 points** per fumble recovery that results in a turnover. The provider has no single
  turnover-qualified recovery stat; the sum of its individual-defensive and special-teams recovery
  keys reproduces the qualified set 268 of 269 times across a validated full season. The known miss
  is a recovery credited on an interception return, where the interception was already the turnover
  — it pays 2 that the rules may not owe. The term is accepted as inexact at this stage rather than
  approximated further; see the open question on own-team recovery after an interception return.

Interceptions caught SHALL be scored only for the defender who caught them. Interceptions thrown
remain the passer's -3 penalty and SHALL NOT pay anyone 6.

Sacks recorded by a defender SHALL be the only sacks that pay. Sacks taken by a quarterback are a
separate stat and SHALL NOT be scored.

Forced fumbles and safeties are excluded from this requirement and are specified separately.

#### Scenario: Interception caught

- **WHEN** a defender catches 2 interceptions and has no other production
- **THEN** the score is 12

#### Scenario: Whole sacks

- **WHEN** a defender records 2 sacks and has no other production
- **THEN** the score is 6

#### Scenario: Half sack pays half

- **WHEN** a defender is credited with half a sack and has no other production
- **THEN** the score is 1.5, not 0 and not 3

#### Scenario: Mixed fractional sack credit

- **WHEN** a defender is credited with 1.5 sacks and has no other production
- **THEN** the score is 4.5

#### Scenario: Fumble recovery resulting in a turnover

- **WHEN** a defender recovers a fumble that results in a turnover and has no other production
- **THEN** the score is 2

#### Scenario: A quarterback sacked is not paid for it

- **WHEN** a quarterback is sacked 4 times in a week
- **THEN** those sacks contribute nothing to that quarterback's score

### Requirement: Touchdowns scored on defense and special teams

The system SHALL pay 6 points for a touchdown scored on defense or on a kick or punt return, on the
same terms as a passing, rushing, or receiving touchdown. The league's "TD scored" rule is
unqualified, so the flavor of the touchdown does not change its value.

A defender who returns an interception for a touchdown SHALL therefore score 12 for the play — 6 for
the interception and 6 for the touchdown.

This rule applies to any rostered player, not only to defenders. A player whose primary production is
offensive SHALL be paid for a return touchdown on the same terms.

A touchdown SHALL be paid only to the player credited with scoring it. A quarterback whose
interception was returned for a touchdown SHALL NOT be paid for that touchdown; the play costs the
quarterback 3 under the interception rule and pays the returning defender.

#### Scenario: Defensive touchdown

- **WHEN** a defender scores a defensive touchdown and has no other production
- **THEN** the score is 6

#### Scenario: Pick-six pays interception and touchdown together

- **WHEN** a defender catches an interception and returns it for a touchdown
- **THEN** the score is 12

#### Scenario: The intercepted quarterback is penalised, not paid

- **WHEN** a quarterback throws an interception that is returned for a touchdown, and has no other
  production
- **THEN** the score is -3, not +3 and not +6

#### Scenario: Return touchdown by an offensive player

- **WHEN** a receiver with 40 receiving yards returns a punt for a touchdown
- **THEN** the touchdown contributes 6 points

#### Scenario: Return touchdown pays the returner only

- **WHEN** a kick or punt is returned for a touchdown
- **THEN** the 6 points are credited to the returning player, and no other player on that team is
  credited for the same touchdown

### Requirement: Scoring rules excluded at the aggregate stage

The system SHALL NOT score forced fumbles, safeties, or the 40+ yard bonus on defensive and return
touchdowns. Each of these is a real league rule, recorded in `docs/scoring.md`, that the provider's
weekly aggregate cannot express correctly:

- **Forced fumbles** pay only when the fumble results in a turnover. Turnover qualification is a
  property of the play rather than of any player's aggregate stat line, so no aggregate stat can
  carry it. Paying the unqualified count would overpay roughly 44% of forced fumbles.
- **Safeties** pay only on solo credit. The aggregate carries a per-player safety stat but nothing
  that distinguishes solo credit from shared.
- **The 40+ yard bonus on defensive and return touchdowns** requires the distance of the scoring
  play. The aggregate carries defensive return yardage only as a weekly sum, so a player with more
  than one return has no attributable distance for the one that scored.

These omissions SHALL be visible rather than silent: a known gap is preferred to a knowingly wrong
award, and each is documented as a stage-scoped deviation rather than a rules change.

#### Scenario: Forced fumble pays nothing

- **WHEN** a defender is credited with a forced fumble and has no other production
- **THEN** the score is 0

#### Scenario: Safety pays nothing

- **WHEN** a defender is credited with a safety and has no other production
- **THEN** the score is 0

#### Scenario: A long defensive touchdown earns no distance bonus

- **WHEN** a defender returns an interception 63 yards for a touchdown
- **THEN** the score is 12 — the interception and the touchdown, with no 40+ yard bonus

#### Scenario: The 40+ bonus still applies to offensive touchdowns

- **WHEN** a player scores a receiving touchdown of 40+ yards
- **THEN** the 40+ yard bonus is awarded, unaffected by its exclusion on defensive and return
  touchdowns

### Requirement: Provider stats that must not be scored

The transform SHALL NOT map provider stats that credit production to a player other than the one who
earned it. The provider records several stats whose names resemble stats the rules pay for, but which
sit on the opposing player or on a team rather than a player. Mapping any of them produces a large,
confident, wrong score on a play the league notices.

The following SHALL be excluded, and the exclusion SHALL be recorded in the transform so that a later
reader does not mistake any of them for an oversight:

- The passer's **interception-returned-for-touchdown** stat, which marks a quarterback who threw a
  pick-six.
- The quarterback's **sacks taken** stat, as distinct from a defender's sacks recorded.
- The provider's **generic fumble-recovery** stat, as distinct from its individual-defensive and
  special-teams recovery stats. The generic key holds recoveries by the fumbling team, which are not
  turnovers. Its exclusion is what makes the scored recovery keys turnover-qualified, so mapping it
  would not add coverage — it would silently break the property the recovery rule depends on. It is
  also the most plausibly-named key in the payload for the rule it must not be used for.
- **Team-level** touchdown, return-touchdown, and safety stats, which the provider mixes into the
  same payload as player entries under non-player identifiers.
- Any **aggregate touchdown total** that does not distinguish the flavor of touchdown, since it
  cannot be reconciled against the flavors scored individually.

#### Scenario: A quarterback who threw a pick-six

- **WHEN** the payload entry for a quarterback carries an interception-returned-for-touchdown stat
- **THEN** it does not contribute to that player's touchdown count, and the player is penalised for
  the interception rather than paid for the touchdown

#### Scenario: Sacks taken are not sacks recorded

- **WHEN** the payload entry for a quarterback carries a sacks-taken stat
- **THEN** it does not contribute to the stat line's sack count

#### Scenario: An own-team fumble recovery is not a turnover

- **WHEN** a payload entry carries the provider's generic fumble-recovery stat and none of the
  turnover-qualified recovery stats
- **THEN** it does not contribute to the stat line's recovery count, and the player is paid nothing
  for it

#### Scenario: Team rows are not scored as players

- **WHEN** the weekly payload contains entries under team identifiers carrying team touchdown,
  return touchdown, or safety stats
- **THEN** those entries are not treated as players and their stats do not reach any player's score

## MODIFIED Requirements

### Requirement: Weekly stat line domain object

The system SHALL represent a single player's single-week NFL production in a provider-neutral
domain object holding raw stat counts. Vendor stat keys (`rec_yd`, `rush_td_40p`, …) SHALL NOT
appear outside the Sleeper transform.

A stat SHALL be represented at the granularity the league scores it. Most stats are whole counts, but
sacks are credited in halves and SHALL be carried as a fractional value. A stat line SHALL NOT round
or truncate a fractional stat, since doing so discards points the rules award.

Fantasy points SHALL NOT be a field of the domain object. Points are computed from a stat line on
demand, so a stat line cannot carry a total that has drifted from the stats it was derived from.
Points appear in responses, paired with the stat line they were computed from.

#### Scenario: Stat line carries stats only

- **WHEN** a stat line is inspected
- **THEN** it exposes the underlying NFL stats and carries no fantasy point total

#### Scenario: Points are paired with the stats they came from

- **WHEN** a fantasy point total is served
- **THEN** it appears alongside the stat line it was computed from

#### Scenario: Absent stats read as zero

- **WHEN** a stat is not present in the provider payload
- **THEN** the domain object reports that stat as zero rather than as unknown or an error

#### Scenario: A fractional stat survives the domain object

- **WHEN** the provider credits a player with half a sack
- **THEN** the stat line reports half a sack rather than zero or one

### Requirement: Sleeper transform

The system SHALL fetch the Sleeper weekly stats aggregate for a given season and week, and SHALL map
any requested player's entry onto the domain stat line. One fetch of the weekly aggregate SHALL serve any
number of requested players. Only the stats needed by the calculator are mapped: rushing yards, receiving
yards, rushing touchdowns, receiving touchdowns, 40+ yard rushing and receiving touchdowns, fumbles lost,
passing yards, passing touchdowns, 40+ yard passing touchdowns, interceptions thrown, two-point
conversions passed, rushed, and received, field goals made, field goals made from 50+ yards, extra
points made, interceptions caught, sacks recorded, fumble recoveries resulting in a turnover,
defensive touchdowns, and special-teams return touchdowns.

Interceptions thrown SHALL be mapped from the passer's interception stat, not from any interception
stat credited to a defender. Interceptions caught SHALL be mapped from the defender's stat, not from
the passer's.

Sacks SHALL be mapped from the defender's sacks-recorded stat and SHALL preserve half-sack
granularity. The quarterback's sacks-taken stat SHALL NOT be mapped.

Passing yards SHALL be mapped from the provider's passing-yards stat, not from its combined
passing-plus-rushing aggregate. The provider carries both, and the combined one appears on
non-quarterbacks.

Fumble recoveries resulting in a turnover SHALL be mapped from the defensive and special-teams
recovery stats together, since the provider books special-teams recoveries under separate keys and
the defensive key alone undercounts. These keys are sparse: they are absent from the payload except
in the weeks a qualifying play occurred, and absence SHALL read as zero rather than as an error.

The three two-point conversion stats MAY be summed into a single stat-line count, since the rules pay
the same 2 points for each.

The touchdown stats mapped SHALL be exactly those credited to the scoring player. Team-level and
passer-side touchdown stats SHALL NOT be mapped, per the requirement on provider stats that must not
be scored.

Forced fumble, safety, and defensive return yardage stats SHALL NOT be mapped, since the rules that
would consume them are excluded at this stage.

A player absent from the weekly payload SHALL be reported as absent rather than as an error. Absence does
not identify its own cause: a player whose game has not kicked off, a player who was inactive, and an
unknown player ID are indistinguishable in the payload. The transform SHALL NOT guess between them, and
SHALL NOT substitute a zero stat line for an absent player.

Transport, status, and decode failures against the Sleeper request remain errors.

#### Scenario: Player present in the weekly payload

- **WHEN** the payload contains an entry for the requested player ID
- **THEN** the transform returns a domain stat line populated from that entry

#### Scenario: Player present but recorded no stats

- **WHEN** the payload contains an entry for the requested player ID that carries none of the mapped
  stat keys
- **THEN** the transform returns a stat line reading zero for every mapped stat, which is a real result
  and not an absence

#### Scenario: Passing stats are mapped

- **WHEN** the payload entry carries passing yards, passing touchdowns, 40+ yard passing touchdowns,
  and interceptions thrown
- **THEN** each is populated on the stat line, distinctly, so that a mistyped key cannot read as zero

#### Scenario: Combined passing-plus-rushing yardage is not mistaken for passing yardage

- **WHEN** the payload entry carries the provider's combined passing-plus-rushing yardage stat
- **THEN** it does not contribute to the stat line's passing yards

#### Scenario: Two-point conversions are mapped in every flavor

- **WHEN** the payload entry carries a two-point conversion passed, rushed, or received
- **THEN** each contributes to the stat line's two-point conversion count

#### Scenario: Kicking stats are mapped

- **WHEN** the payload entry carries field goals made, field goals made from 50+ yards, and extra
  points made
- **THEN** each is populated on the stat line, distinctly

#### Scenario: Defensive stats are mapped

- **WHEN** the payload entry carries interceptions caught, sacks recorded, and a defensive touchdown
- **THEN** each is populated on the stat line, distinctly

#### Scenario: Half-sack granularity survives the transform

- **WHEN** the payload entry credits a player with half a sack
- **THEN** the stat line reports half a sack rather than zero

#### Scenario: Special-teams fumble recovery keys are absent from most weeks

- **WHEN** the payload carries no special-teams fumble recovery keys for the requested week
- **THEN** the transform maps the defensive recovery stat alone and reports zero for the others,
  without erroring

#### Scenario: Excluded stats are not mapped

- **WHEN** the payload entry carries forced fumble, safety, or defensive return yardage stats
- **THEN** none of them appears on the stat line

#### Scenario: Player absent from the weekly payload

- **WHEN** the payload contains no entry for the requested player ID
- **THEN** the transform reports that player as absent, without returning an error and without returning
  a zeroed stat line

#### Scenario: Weekly payload is empty

- **WHEN** the requested week has not been played and the payload contains no entries at all
- **THEN** every requested player is reported as absent

#### Scenario: Upstream request fails

- **WHEN** the Sleeper request fails, returns a non-200 status, or returns an undecodable body
- **THEN** the transform returns an error naming the season and week

### Requirement: Rushing and receiving fantasy points

The system SHALL compute HMFFL fantasy points from a domain stat line using the rushing/receiving subset
of the league rules in `docs/scoring.md`:

- **Yardage bonus, awarded at most once.** A player qualifies with 80+ rushing yards, 80+ receiving
  yards, or 100+ combined rushing and receiving yards. Qualifying awards 3 points, plus 0.5 points for
  each full additional 10 yards beyond the qualifying threshold. Where more than one clause qualifies,
  the one yielding the most points is used.
- **6 points** per rushing or receiving touchdown.
- **1 bonus point** per rushing or receiving touchdown of 40+ yards.
- **-3 points** per fumble lost.

Passing, two-point conversions, kicking, and defensive production are scored separately, under their
own requirements. Touchdowns scored on defense or special teams are likewise scored under their own
requirement, at the same 6 points.

#### Scenario: Below every yardage threshold

- **WHEN** a player has 60 receiving yards, 20 rushing yards, and no touchdowns
- **THEN** the score is 0 — 80 combined yards clears no threshold

#### Scenario: Yardage bonus is awarded once

- **WHEN** a player has 80 rushing yards and 80 receiving yards
- **THEN** the yardage award is a single 3 points plus the increments earned past 100 combined, not two
  separate 3-point awards

#### Scenario: Increments are floored

- **WHEN** a player has 99 receiving yards
- **THEN** the score is 3.5 — the 3-point award plus one full 10-yard increment, with the remaining 9
  yards paying nothing

#### Scenario: Best qualifying clause wins

- **WHEN** a player has 105 receiving yards and 0 rushing yards
- **THEN** the score is 4.0 — the receiving clause pays 3 plus two 10-yard increments for the 25 yards
  past 80, beating the combined clause's 3 for 5 yards past 100

#### Scenario: Touchdowns and the long-touchdown bonus

- **WHEN** a player has 2 receiving touchdowns, one of which covered 40+ yards
- **THEN** those plays contribute 13 points — 6 each plus 1 for the long touchdown

#### Scenario: Fumble lost

- **WHEN** a player has 85 receiving yards and 1 fumble lost
- **THEN** the score is 0 — a 3-point yardage award, no increment, minus 3

### Requirement: Passing fantasy points

The system SHALL compute HMFFL fantasy points for passing production from a domain stat line, using the
passing rules in `docs/scoring.md`:

- **Yardage bonus.** 200+ passing yards awards 3 points, plus 1 point for each full additional 50 yards
  beyond 200. Increments are floored, consistent with the rushing/receiving thresholds.
- **6 points** per passing touchdown.
- **1 bonus point** per passing touchdown of 40+ yards, per the rule that the long-touchdown bonus pays
  every player credited on the play.
- **-3 points** per interception thrown.

The passing yardage bonus SHALL be awarded independently of the rushing/receiving yardage bonus, and the
two SHALL both apply when both qualify. The "awarded at most once" rule governs only the three
rushing/receiving clauses it is stated for; passing yards are earned on different plays and are not in
competition with them.

Interceptions thrown SHALL be the only interceptions that penalise a player. An interception caught by
a defender pays that defender 6 under the defensive scoring requirement and SHALL NOT be confused with
the passer's penalty.

Kicking, defensive, and defensive-touchdown scoring are specified under their own requirements.
Two-point conversions are likewise scored under their own requirement, since they are not exclusively
a passing play.

#### Scenario: Below the passing yardage threshold

- **WHEN** a player has 199 passing yards and nothing else
- **THEN** the score is 0 — the 200-yard threshold is not cleared

#### Scenario: Passing increments are floored

- **WHEN** a player has 249 passing yards
- **THEN** the score is 3 — the award with no full 50-yard increment, the remaining 49 yards paying nothing

#### Scenario: Passing increment is earned exactly at the boundary

- **WHEN** a player has 250 passing yards
- **THEN** the score is 4 — the 3-point award plus one full 50-yard increment

#### Scenario: Passing touchdown

- **WHEN** a player throws 2 touchdown passes and has no other production
- **THEN** those plays contribute 12 points

#### Scenario: Long passing touchdown bonus

- **WHEN** a player throws 2 touchdown passes, one of which covered 40+ yards
- **THEN** those plays contribute 13 points — 6 each plus 1 for the long touchdown

#### Scenario: Interception thrown

- **WHEN** a player throws 3 interceptions and has no other production
- **THEN** the score is -9

#### Scenario: Passing and rushing awards both apply

- **WHEN** a player has 250 passing yards and 80 rushing yards
- **THEN** the score is 7 — the 4-point passing award plus the 3-point rushing award, not the better of
  the two

#### Scenario: A player with no passing production is unaffected

- **WHEN** a stat line carries no passing stats
- **THEN** the score is exactly what the rushing/receiving rules alone produce
