## MODIFIED Requirements

### Requirement: The id is authoritative and the name is a label

The id SHALL be the only field used to identify a player to anything outside the roster. The name
and the position SHALL be carried as display text only, and SHALL NOT be used to look up, match, or
validate a player. The short name derived from the name SHALL be display text on the same terms,
and nothing SHALL identify a player by it.

The team SHALL NOT be used to identify, match, or validate a player either. It MAY be used to find
the NFL game that team plays in the record's week. Reading SHALL NOT check the team against the
player.

A record MAY have an empty name. Reading SHALL NOT reject it, invent a placeholder, or fall back to
the id, so that a roster written without labels stays readable as written. A record with an empty
name SHALL have an empty short name.

#### Scenario: A record with no name is read

- **WHEN** a record is written as `4984,,QB,BUF`
- **THEN** the record is read with id `4984`, an empty name, and an empty short name

#### Scenario: The labels are not checked against the player

- **WHEN** a record pairs a valid id with a name, position, and team that belong to a different
  player, and the position and team are each in their closed sets
- **THEN** reading succeeds and the record carries the labels as written

#### Scenario: The team names a game, not a player

- **WHEN** a record's team is `BUF`
- **THEN** the record's stats are still found by its id alone, and `BUF` is what a caller may use to
  find that week's `BUF` game
