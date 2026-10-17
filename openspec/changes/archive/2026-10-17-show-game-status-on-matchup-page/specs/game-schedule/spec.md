## ADDED Requirements

### Requirement: A week's games are fetched from the provider in one call

The system SHALL fetch a season and week's NFL games from the provider with one request, and from
it SHALL read, for each game: the home team code, the away team code, the provider's status, the
kickoff instant, and the provider's quarter text.

The kickoff instant SHALL be the provider's start time, which is given in milliseconds since the
Unix epoch. A missing quarter text SHALL be read as empty.

A game that lacks either team code SHALL be left out. The other games of the week SHALL still be
read.

#### Scenario: A week's games are read

- **WHEN** the provider returns fifteen games for 2026 week 5
- **THEN** the week holds fifteen games, each with its two team codes, status, kickoff instant, and
  quarter text

#### Scenario: The kickoff instant is exact

- **WHEN** a game's start time is `1791750300000`
- **THEN** its kickoff instant is 2026-10-11T20:25:00Z

#### Scenario: A null quarter is empty

- **WHEN** a game's quarter text is null
- **THEN** the game is read with an empty quarter text

#### Scenario: A game without a team code is skipped

- **WHEN** one of three returned games has no home team code
- **THEN** the week holds the other two games

### Requirement: A game is upcoming, in progress, complete, or carried as the provider states it

Each game SHALL have one state. The provider's status `pre_game` SHALL be upcoming, `in_game` SHALL
be in progress, and `complete` SHALL be complete.

Any other status SHALL NOT be mapped to one of those three. The game SHALL be kept, marked as having
an unrecognised state, with the provider's status text carried unchanged.

The provider's quarter text SHALL be carried unchanged in every state.

#### Scenario: The three known statuses

- **WHEN** three games have status `pre_game`, `in_game`, and `complete`
- **THEN** they are upcoming, in progress, and complete

#### Scenario: Halftime is in progress

- **WHEN** a game has status `in_game` and quarter text `HALF`
- **THEN** the game is in progress and its quarter text is `HALF`

#### Scenario: An unrecognised status is kept as sent

- **WHEN** a game has status `postponed`
- **THEN** the game is in the week, its state is unrecognised, and its status text is `postponed`

### Requirement: A team's game in a week is found by team code

Given a team code, a week's games SHALL answer with the game in which that code is the home or the
away team, or SHALL answer that the team has no game that week.

The match SHALL be exact, including case.

#### Scenario: A home team finds its game

- **WHEN** the week holds `DET` at `ARI` and the game for `ARI` is asked for
- **THEN** that game is returned

#### Scenario: An away team finds its game

- **WHEN** the week holds `DET` at `ARI` and the game for `DET` is asked for
- **THEN** that game is returned

#### Scenario: A team with no game

- **WHEN** the week holds fifteen games, none involving `KC`, and the game for `KC` is asked for
- **THEN** the answer is that `KC` has no game that week

### Requirement: A week with no games is distinguishable from a week of byes

A week's games SHALL report whether it holds any game at all, so a caller can tell a week the
provider has nothing for from a week in which a particular team is idle.

A provider response with no games SHALL NOT be an error.

#### Scenario: An empty response is an empty week

- **WHEN** the provider answers 200 with an empty list for 2026 week 19
- **THEN** the fetch succeeds and the week reports that it holds no games

#### Scenario: A week with games is not empty

- **WHEN** the provider returns fifteen games
- **THEN** the week reports that it holds games

### Requirement: A failed game fetch is an error

When the provider does not answer 200, or its body cannot be read as a list of games, the fetch
SHALL fail with an error naming the season and week. It SHALL NOT return a partial or empty week in
place of the error.

#### Scenario: The provider answers with an error status

- **WHEN** the provider answers 503 for 2026 week 5
- **THEN** the fetch fails with an error naming season 2026 and week 5

#### Scenario: The body is not a list of games

- **WHEN** the provider answers 200 with a body that is not a JSON list
- **THEN** the fetch fails with an error naming the season and week
