## ADDED Requirements

### Requirement: Weekly stat line domain object

The system SHALL represent a single player's single-week NFL production in a provider-neutral
domain object holding raw stat counts and the fantasy points computed from them. Vendor stat keys
(`rec_yd`, `rush_td_40p`, …) SHALL NOT appear outside the Sleeper transform.

#### Scenario: Stat line carries stats and points together

- **WHEN** a stat line has been scored
- **THEN** it exposes both the underlying NFL stats and the resulting fantasy point total

#### Scenario: Absent stats read as zero

- **WHEN** a stat is not present in the provider payload
- **THEN** the domain object reports that stat as zero rather than as unknown or an error

### Requirement: Sleeper transform

The system SHALL fetch the Sleeper weekly stats aggregate for a given season and week and map one
player's entry onto the domain stat line. Only the rushing/receiving stats needed by the calculator
are mapped: rushing yards, receiving yards, rushing touchdowns, receiving touchdowns, 40+ yard
rushing and receiving touchdowns, and fumbles lost. These are the stats a wide receiver can
plausibly produce; everything else is out of scope.

#### Scenario: Player present in the weekly payload

- **WHEN** the payload contains an entry for the requested player ID
- **THEN** the transform returns a domain stat line populated from that entry

#### Scenario: Player absent from the weekly payload

- **WHEN** the payload contains no entry for the requested player ID
- **THEN** the transform returns an error naming the player ID, season, and week

### Requirement: Rushing and receiving fantasy points

The system SHALL compute HMFFL fantasy points from a domain stat line using the rushing/receiving
subset of the league rules in `docs/scoring.md`:

- **Yardage bonus, awarded at most once.** A player qualifies with 80+ rushing yards, 80+ receiving
  yards, or 100+ combined rushing and receiving yards. Qualifying awards 3 points, plus 0.5 points
  for each full additional 10 yards beyond the qualifying threshold. Where more than one clause
  qualifies, the one yielding the most points is used.
- **6 points** per rushing or receiving touchdown.
- **1 bonus point** per touchdown of 40+ yards.
- **-3 points** per fumble lost.

Passing, kicking, defensive, and two-point-conversion scoring are out of scope for this change.

#### Scenario: Below every yardage threshold

- **WHEN** a player has 60 receiving yards, 20 rushing yards, and no touchdowns
- **THEN** the score is 0 — 80 combined yards clears no threshold

#### Scenario: Yardage bonus is awarded once

- **WHEN** a player has 80 rushing yards and 80 receiving yards
- **THEN** the yardage award is a single 3 points plus the increments earned past 100 combined,
  not two separate 3-point awards

#### Scenario: Increments are floored

- **WHEN** a player has 99 receiving yards
- **THEN** the score is 3.5 — the 3-point award plus one full 10-yard increment, with the
  remaining 9 yards paying nothing

#### Scenario: Best qualifying clause wins

- **WHEN** a player has 105 receiving yards and 0 rushing yards
- **THEN** the score is 4.0 — the receiving clause pays 3 plus two 10-yard increments for the 25
  yards past 80, beating the combined clause's 3 for 5 yards past 100

#### Scenario: Touchdowns and the long-touchdown bonus

- **WHEN** a player has 2 receiving touchdowns, one of which covered 40+ yards
- **THEN** those plays contribute 13 points — 6 each plus 1 for the long touchdown

#### Scenario: Fumble lost

- **WHEN** a player has 85 receiving yards and 1 fumble lost
- **THEN** the score is 0 — a 3-point yardage award, no increment, minus 3

### Requirement: Score endpoint

The system SHALL expose an HTTP endpoint that, when hit, fetches and scores the hardcoded target
player and week (Puka Nacua, 2025 regular season, week 14), prints the resulting score to standard
output, and returns the stat line and score as JSON.

#### Scenario: Successful scoring request

- **WHEN** the endpoint is requested and Sleeper returns the weekly payload
- **THEN** the server responds 200 with JSON containing the player's stats and fantasy points, and
  writes the score to standard output

#### Scenario: Upstream failure

- **WHEN** the Sleeper request fails or the player is absent from the payload
- **THEN** the server responds with a 5xx status and an error message rather than a zero score
