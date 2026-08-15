## MODIFIED Requirements

### Requirement: Sleeper transform

The system SHALL fetch the Sleeper weekly stats aggregate for a given season and week, and SHALL map any
requested player's entry onto the domain stat line. One fetch of the weekly aggregate SHALL serve any number of
requested players. Only the rushing/receiving stats needed by the calculator are mapped: rushing yards,
receiving yards, rushing touchdowns, receiving touchdowns, 40+ yard rushing and receiving touchdowns, and
fumbles lost. These are the stats a wide receiver can plausibly produce; everything else is out of scope.

A player absent from the weekly payload SHALL be reported as absent rather than as an error. Absence does not
identify its own cause: a player whose game has not kicked off, a player who was inactive, and an unknown
player ID are indistinguishable in the payload. The transform SHALL NOT guess between them, and SHALL NOT
substitute a zero stat line for an absent player.

Transport, status, and decode failures against the Sleeper request remain errors.

#### Scenario: Player present in the weekly payload

- **WHEN** the payload contains an entry for the requested player ID
- **THEN** the transform returns a domain stat line populated from that entry

#### Scenario: Player present but recorded no stats

- **WHEN** the payload contains an entry for the requested player ID that carries none of the mapped stat keys
- **THEN** the transform returns a stat line reading zero for every mapped stat, which is a real result and not
  an absence

#### Scenario: Player absent from the weekly payload

- **WHEN** the payload contains no entry for the requested player ID
- **THEN** the transform reports that player as absent, without returning an error and without returning a
  zeroed stat line

#### Scenario: Weekly payload is empty

- **WHEN** the requested week has not been played and the payload contains no entries at all
- **THEN** every requested player is reported as absent

#### Scenario: Upstream request fails

- **WHEN** the Sleeper request fails, returns a non-200 status, or returns an undecodable body
- **THEN** the transform returns an error naming the season and week

## ADDED Requirements

### Requirement: Multi-player score endpoint

The system SHALL expose an HTTP endpoint that accepts a season, a week, and a list of player IDs in the request
body, and returns the stats and fantasy points for each of those players from a single fetch of the weekly
stats aggregate.

The response SHALL separate scored players from absent ones. Fantasy points SHALL appear only alongside the
stat line they were computed from, so that no absent player can carry a point total. Absent players SHALL be
reported by player ID.

A request in which every player is absent SHALL succeed. An unplayed week is a legitimate answer, not a
failure.

#### Scenario: All requested players have stats

- **WHEN** the endpoint is requested with player IDs that are all present in the weekly payload
- **THEN** the server responds 200 with a scored entry for each requested player and an empty absent list

#### Scenario: Some requested players are absent

- **WHEN** the endpoint is requested with a mix of players present in and absent from the weekly payload
- **THEN** the server responds 200 with the present players scored and the absent players named by ID in the
  absent list

#### Scenario: A scored player is distinguishable from an absent one

- **WHEN** a requested player is present in the payload and their stats produce zero fantasy points
- **THEN** the response reports them as scored with a point total of zero, not as absent

#### Scenario: Every requested player is absent

- **WHEN** the endpoint is requested for a week that has not been played
- **THEN** the server responds 200 with no scored entries and every requested player ID in the absent list

#### Scenario: Every requested player is accounted for

- **WHEN** the endpoint responds successfully
- **THEN** the number of scored entries plus the number of absent player IDs equals the number of player IDs
  requested

#### Scenario: Repeated player IDs

- **WHEN** the same player ID appears more than once in the request
- **THEN** it appears once per occurrence in the response

#### Scenario: Upstream failure

- **WHEN** the Sleeper request fails, returns a non-200 status, or returns an undecodable body
- **THEN** the server responds with a 5xx status and an error message rather than a partial or zeroed result

### Requirement: Multi-player request validation

The system SHALL reject a malformed multi-player scoring request with a 4xx status and an error message,
without contacting the stat provider. A request is malformed when its body is not valid JSON, when the season
or week is missing or outside the range of a plausible NFL season and week, when the player ID list is empty,
or when the player ID list holds more than 26 entries.

The cap of 26 is the league's maximum roster size, which comfortably exceeds two full starting lineups.

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
