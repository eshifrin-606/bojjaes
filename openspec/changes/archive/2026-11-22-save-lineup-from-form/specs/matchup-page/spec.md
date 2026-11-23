## ADDED Requirements

### Requirement: A side not yet set is shown as TBD

When `lineup-source` resolves a week to one lineup, the page SHALL be served with that side's column
as usual and the other side's column showing the heading `TBD`, no total, and no starters. The TBD
column SHALL hold the side it stands for: the left when the Bojjaes are missing, the right when the
opponent is. The page SHALL still fetch the week's stats once and score the side it has, and SHALL
imply no winner.

#### Scenario: The opponent is to be decided

- **WHEN** `2026/12/` holds only `bojjaes.csv` and `GET /2026/12` is requested
- **THEN** the response is 200, the left column is the Bojjaes with their starters and total, and the
  right column's heading reads `TBD` with no total and no starters

#### Scenario: The Bojjaes are to be decided

- **WHEN** `2026/12/` holds only `wood.csv` and `GET /2026/12` is requested
- **THEN** the response is 200, the left column's heading reads `TBD`, and the right column is wood
  with its starters and total

#### Scenario: The season URL shows a one-lineup latest week

- **WHEN** 2026 week 12 is the season's latest week and holds only `bojjaes.csv`, and `GET /2026` is
  requested
- **THEN** the response is 200 with week 12's page, the opponent shown as TBD

## MODIFIED Requirements

### Requirement: A week that is not a matchup is refused, not guessed

A week directory the lineup tree cannot resolve to a matchup — two rosters, one of them the
Bojjaes', or one roster — SHALL be refused with an error response naming the problem. The page SHALL
NOT be rendered from two of three rosters, or from two rosters the Bojjaes are not in. A week of one
roster is rendered with the other side as TBD (see *A side not yet set is shown as TBD*).

A week directory that does not exist SHALL be refused with `404 Not Found`. A week directory that
exists but does not hold a matchup SHALL be refused with `500 Internal Server Error`, because the
request was well formed and the served lineup tree is wrong.

A roster file that cannot be parsed as a lineup card SHALL likewise be refused with `500 Internal
Server Error` rather than rendered around.

#### Scenario: A week with no directory is a 404

- **WHEN** `GET /2025/17` is requested and no such week directory exists
- **THEN** the response is `404 Not Found`

#### Scenario: A week with three rosters is a 500

- **WHEN** a week directory holds three roster files
- **THEN** the response is `500 Internal Server Error` and no page is rendered

#### Scenario: A week without the Bojjaes is a 500

- **WHEN** a week directory holds two rosters and neither is `bojjaes.csv`
- **THEN** the response is `500 Internal Server Error`

#### Scenario: An unparseable roster is a 500

- **WHEN** one of the week's roster files has a line with no id
- **THEN** the response is `500 Internal Server Error` and no page is rendered
