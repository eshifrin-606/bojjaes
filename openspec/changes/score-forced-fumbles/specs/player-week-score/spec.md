## MODIFIED Requirements

### Requirement: Defensive fantasy points

The system SHALL compute HMFFL fantasy points for individual defensive production from a domain stat
line, using these defensive rules from `docs/scoring.md`:

- **6 points** per interception caught.
- **3 points** per sack. Sacks are credited in half-sack granularity, and a half sack SHALL pay 1.5
  points. The award is proportional rather than tabulated, so any fractional credit the provider
  reports pays its proportional share.
- **2 points** per fumble recovery that results in a turnover. The provider has no single
  turnover-qualified recovery stat. The sum of its individual-defensive and special-teams recovery
  keys reproduces the qualified set 268 of 269 times across a validated full season. The known miss
  is a recovery credited on an interception return, where the interception was already the turnover.
  It pays 2 that the rules may not owe. The term is accepted as inexact rather than approximated
  further; see the open question on own-team recovery after an interception return.
- **4 points** per forced fumble that results in a turnover. The stat line carries a
  provider-neutral count of turnover-qualified forced fumbles. A forced fumble that was not a turnover
  never reaches that count, so the rule is a flat term over it. How the count is decided is specified
  in `forced-fumble-attribution`.

Interceptions caught SHALL be scored only for the defender who caught them. Interceptions thrown
remain the passer's -3 penalty and SHALL NOT pay anyone 6.

Sacks recorded by a defender SHALL be the only sacks that pay. Sacks taken by a quarterback are a
separate stat and SHALL NOT be scored.

The provider's aggregate forced-fumble stat SHALL NOT feed the forced-fumble count. It is not
turnover-qualified.

Safeties are excluded from this requirement and are specified separately.

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

#### Scenario: Forced fumble resulting in a turnover

- **WHEN** a defender is credited with one turnover-qualified forced fumble and has no other
  production
- **THEN** the score is 4

#### Scenario: Forced fumbles pay per fumble

- **WHEN** a defender is credited with two turnover-qualified forced fumbles and has no other
  production
- **THEN** the score is 8

#### Scenario: Official week 3 total for Will Anderson

- **WHEN** a defender records 2.5 sacks, one turnover-qualified fumble recovery, and one
  turnover-qualified forced fumble
- **THEN** the score is 13.5

### Requirement: Scoring rules excluded at the aggregate stage

The system SHALL NOT score safeties. Safety is a real league rule, recorded in `docs/scoring.md`, that
the provider's weekly aggregate cannot express correctly. It pays only on solo credit. The aggregate
carries a per-player safety stat but nothing that distinguishes solo credit from shared.

This omission SHALL be visible rather than silent: a known gap is preferred to a knowingly wrong
award, and it is documented as a stage-scoped deviation rather than a rules change.

The 40+ yard bonus is not an exclusion. It applies to offensive touchdowns only, and defensive and
return touchdowns never earn it at any distance.

#### Scenario: Safety pays nothing

- **WHEN** a defender is credited with a safety and has no other production
- **THEN** the score is 0

#### Scenario: A long defensive touchdown earns no distance bonus

- **WHEN** a defender returns an interception 63 yards for a touchdown
- **THEN** the score is 12: the interception and the touchdown, with no 40+ yard bonus, because the
  rule does not pay one

#### Scenario: The 40+ bonus still applies to offensive touchdowns

- **WHEN** a player scores a receiving touchdown of 40+ yards
- **THEN** the 40+ yard bonus is awarded

## ADDED Requirements

### Requirement: The weekly snapshot carries forced fumbles from play-by-play

A week's snapshot SHALL take every stat except forced fumbles from the provider's weekly aggregate.
It SHALL take the turnover-qualified forced-fumble count from play-by-play attribution. The two
sources SHALL be merged before the snapshot is returned, so the snapshot stays complete on return.

A player credited with a forced fumble but absent from the aggregate SHALL appear in the snapshot with
that count and otherwise zero stats.

When play-by-play is unavailable, the snapshot SHALL still be returned, built from the aggregate and
whatever forced fumbles are already held. An aggregate failure SHALL still fail the fetch as before.

#### Scenario: A forced fumble reaches the stat line

- **WHEN** play-by-play attribution credits a player with one turnover-qualified forced fumble in a
  week
- **THEN** that player's stat line read from the week's snapshot carries a count of 1

#### Scenario: A forcer missing from the aggregate is still paid

- **WHEN** a player is credited with a forced fumble but has no entry in the weekly aggregate
- **THEN** the snapshot holds a stat line for that player carrying only the forced fumble

#### Scenario: Play-by-play failure leaves the aggregate intact

- **WHEN** play-by-play cannot be read and nothing is held for the week
- **THEN** the snapshot is returned with every aggregate stat and no forced fumbles
