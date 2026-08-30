# player-week-score

## Purpose

Score NFL players' single-week production under the HMFFL rules, from a provider stat feed through
to an HTTP response. Covers the passing, rushing, receiving, two-point-conversion, kicking, and
defensive rules in `docs/scoring.md`.

A score is therefore meaningful for every rostered player, kickers and defenders included. The rules
that need play-by-play data — forced fumbles, safeties, and the 40+ yard bonus on defensive and
return touchdowns — are absent from the calculation rather than rejected by it, so a defender's
score is a number that may be low rather than an error.

## Requirements

### Requirement: Weekly stat line domain object

The system SHALL represent a single player's single-week NFL production in a provider-neutral
domain object holding raw stat counts. Vendor stat keys (`rec_yd`, `rush_td_40p`, …) SHALL NOT
appear outside the Sleeper transform.

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

### Requirement: Sleeper transform

The system SHALL fetch the Sleeper weekly stats aggregate for a given season and week, and SHALL map
any requested player's entry onto the domain stat line. One fetch of the weekly aggregate SHALL serve any
number of requested players. Only the stats needed by the calculator are mapped: rushing yards, receiving
yards, rushing touchdowns, receiving touchdowns, 40+ yard rushing and receiving touchdowns, fumbles lost,
passing yards, passing touchdowns, 40+ yard passing touchdowns, interceptions thrown, and two-point
conversions passed, rushed, and received. Kicking and defensive stats are out of scope.

Interceptions thrown SHALL be mapped from the passer's interception stat, not from any interception
stat credited to a defender.

Passing yards SHALL be mapped from the provider's passing-yards stat, not from its combined
passing-plus-rushing aggregate. The provider carries both, and the combined one appears on
non-quarterbacks.

The three two-point conversion stats MAY be summed into a single stat-line count, since the rules pay
the same 2 points for each.

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

Kicking and defensive scoring are out of scope. Passing and two-point conversions are scored
separately, under their own requirements.

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

Interceptions thrown SHALL be the only interceptions that affect a player's score. Interceptions caught
are defensive scoring and remain out of scope.

Kicking and defensive scoring remain out of scope. Two-point conversions are scored under their own
requirement, since they are not exclusively a passing play.

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

### Requirement: Two-point conversion fantasy points

The system SHALL award 2 points per two-point conversion a player is credited with, whether the
player threw it, ran it in, or caught it. A single conversion play credits both the passer and the
scorer, and SHALL pay both.

The award SHALL NOT depend on the flavor of conversion, so the stat line need not distinguish
between them.

#### Scenario: Two-point conversion pass

- **WHEN** a player throws a two-point conversion and has no other production
- **THEN** the score is 2

#### Scenario: Two-point conversion run

- **WHEN** a player runs in a two-point conversion and has no other production
- **THEN** the score is 2

#### Scenario: Two-point conversion reception

- **WHEN** a player catches a two-point conversion and has no other production
- **THEN** the score is 2

#### Scenario: Both players on one conversion play are paid

- **WHEN** one player throws a two-point conversion to another who catches it
- **THEN** each of the two players scores 2 from that play

#### Scenario: Multiple conversions in a week

- **WHEN** a player is credited with two conversions in the same week
- **THEN** those plays contribute 4 points

### Requirement: Score endpoint

The system SHALL expose an HTTP endpoint that, when hit, fetches and scores the hardcoded target
player and week (Puka Nacua, 2025 regular season, week 14), prints the resulting score to standard
output, and returns the stat line and score as JSON.

This endpoint SHALL treat an absent player as a failure, even though the transform now reports
absence as an ordinary result. The endpoint performs that conversion itself. Its target is a settled
historical week in which the player is known to be present, so absence there indicates the fetch or
the transform is broken rather than that the player has no stats.

#### Scenario: Successful scoring request

- **WHEN** the endpoint is requested and Sleeper returns the weekly payload
- **THEN** the server responds 200 with JSON containing the player's stats and fantasy points, and
  writes the score to standard output

#### Scenario: Upstream failure

- **WHEN** the Sleeper request fails or returns a non-200 status
- **THEN** the server responds with a 5xx status and an error message rather than a zero score

#### Scenario: Target player absent from a settled week

- **WHEN** the target player is absent from the weekly payload
- **THEN** the endpoint responds 502 rather than reporting the player as unscored

### Requirement: Multi-player score endpoint

The system SHALL expose an HTTP endpoint that accepts a season, a week, and a list of player IDs in
the request body, and returns the stats and fantasy points for each of those players from a single
fetch of the weekly stats aggregate.

The season and week SHALL be interpreted as the regular season. The request carries no season type,
and the endpoint SHALL NOT fetch preseason or postseason stats. Preseason is useful only for
exercising the server against a live in-progress week during local manual testing; the league itself
scores regular-season play, so selecting a season type is deferred rather than specified here.

The response SHALL separate scored players from absent ones. Fantasy points SHALL appear only
alongside the stat line they were computed from, so that no absent player can carry a point total.
Absent players SHALL be reported by player ID.

A request in which every player is absent SHALL succeed. An unplayed week is a legitimate answer,
not a failure.

#### Scenario: All requested players have stats

- **WHEN** the endpoint is requested with player IDs that are all present in the weekly payload
- **THEN** the server responds 200 with a scored entry for each requested player and an empty absent
  list

#### Scenario: Some requested players are absent

- **WHEN** the endpoint is requested with a mix of players present in and absent from the weekly
  payload
- **THEN** the server responds 200 with the present players scored and the absent players named by
  ID in the absent list

#### Scenario: A scored player is distinguishable from an absent one

- **WHEN** a requested player is present in the payload and their stats produce zero fantasy points
- **THEN** the response reports them as scored with a point total of zero, not as absent

#### Scenario: Every requested player is absent

- **WHEN** the endpoint is requested for a week that has not been played
- **THEN** the server responds 200 with no scored entries and every requested player ID in the
  absent list

#### Scenario: Every requested player is accounted for

- **WHEN** the endpoint responds successfully
- **THEN** the number of scored entries plus the number of absent player IDs equals the number of
  player IDs requested

#### Scenario: Repeated player IDs

- **WHEN** the same player ID appears more than once in the request
- **THEN** it appears once per occurrence in the response

#### Scenario: Upstream failure

- **WHEN** the Sleeper request fails, returns a non-200 status, or returns an undecodable body
- **THEN** the server responds with a 5xx status and an error message rather than a partial or
  zeroed result

### Requirement: Multi-player request validation

The system SHALL reject a malformed multi-player scoring request with a 4xx status and an error
message, without contacting the stat provider. A request is malformed when its body is not valid
JSON, when the season or week is missing or outside the range of a plausible NFL season and week,
when the player ID list is empty, or when the player ID list holds more than 26 entries.

The cap of 26 is the league's maximum roster size, which comfortably exceeds two full starting
lineups.

#### Scenario: Body is not valid JSON

- **WHEN** the request body cannot be decoded as JSON
- **THEN** the server responds 4xx and does not contact the stat provider

#### Scenario: Season or week missing or out of range

- **WHEN** the request omits the season or the week, or supplies a value outside the plausible range
- **THEN** the server responds 4xx and does not contact the stat provider

#### Scenario: Empty player list

- **WHEN** the request supplies no player IDs
- **THEN** the server responds 4xx and does not contact the stat provider

#### Scenario: Too many player IDs

- **WHEN** the request supplies more than 26 player IDs
- **THEN** the server responds 4xx and does not contact the stat provider

#### Scenario: Player ID count at the limit

- **WHEN** the request supplies exactly 26 player IDs
- **THEN** the request is accepted and scored
