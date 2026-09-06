## ADDED Requirements

### Requirement: The page is addressed by season and week

The server SHALL serve the matchup page at `GET /{season}/{week}`, where both segments are decimal
integers, e.g. `/2025/15`. No team appears in the URL: the week directory names both teams.

A segment that is not an integer, or a season or week outside the range the scoring endpoints
already accept, SHALL be refused with `400 Bad Request` and SHALL NOT reach the lineup tree or the
stat provider.

#### Scenario: A season and week resolve to a page

- **WHEN** `GET /2025/15` is requested and that week directory holds the Bojjaes and one opponent
- **THEN** the response is `200 OK` with an HTML content type

#### Scenario: A non-numeric segment is refused

- **WHEN** `GET /2025/fifteen` is requested
- **THEN** the response is `400 Bad Request` and no roster file is opened and no upstream fetch is made

#### Scenario: An out-of-range week is refused

- **WHEN** `GET /2025/23` is requested
- **THEN** the response is `400 Bad Request`

### Requirement: The page shows the week's matchup, the Bojjaes on the left

The page SHALL show exactly two columns: the two teams of that week directory, resolved as the
`roster-source` capability resolves them. The Bojjaes SHALL hold the left column and their opponent
the right, regardless of file name order in the directory.

#### Scenario: The Bojjaes hold the left column

- **WHEN** a week directory holds `bojjaes.csv` and `wood.csv`
- **THEN** the left column is the Bojjaes and the right column is `wood`

#### Scenario: An alphabetically earlier opponent still holds the right column

- **WHEN** a week directory holds `bojjaes.csv` and `aardvarks.csv`
- **THEN** the left column is still the Bojjaes and `aardvarks` is on the right

### Requirement: Each column is a team, its starters, and one total

Each column SHALL show the team name, that team's starting lineup in file order, and the total of
the starters' points. Starters are the roster's first nine records per `roster-source`; bench
players SHALL NOT appear on the page.

Each starter line SHALL show the player's name from the roster file and that player's points for
the requested season and week, computed by the `player-week-score` rules.

The two columns SHALL be rendered at equal width, and neither column's length or content SHALL
change the other's width.

#### Scenario: A column totals its starters

- **WHEN** a team's nine starters score 12, 0, 6, 3, 9, 0, 15, 4, and 7 points
- **THEN** that column lists all nine players with those points and shows a total of 56

#### Scenario: Bench players are not rendered

- **WHEN** a roster file holds twelve records
- **THEN** the column shows the first nine and the page contains no mention of the last three

### Requirement: A starter the provider has no stats for shows no number

A starter absent from the provider's weekly payload SHALL be rendered with a placeholder in place
of a point value, not with `0`, and SHALL contribute nothing to the column total. A starter present
in the payload with no scoring production SHALL be rendered as `0` and is a real scoreless line.

Absence and a scoreless week are distinguishable on the page because they are different facts: the
payload cannot say whether a missing player has not kicked off, is inactive, or does not exist.

#### Scenario: A missing starter is not zero

- **WHEN** one of nine starters has no entry in the weekly payload and the other eight total 40
- **THEN** that starter's line shows the placeholder rather than `0` and the column total is 40

#### Scenario: A scoreless starter is zero

- **WHEN** a starter has an entry in the weekly payload but scores nothing under the rules
- **THEN** that starter's line shows `0`

### Requirement: Both columns are scored from one fetch, or the page is not served

The handler SHALL fetch the weekly stat aggregate once per request and score both columns from it,
so the two columns cannot be read from different snapshots of the week.

If that fetch fails, the server SHALL respond `502 Bad Gateway` and SHALL NOT render the page. A
partially scored or zeroed page is forbidden: it reads as "these players scored nothing" rather
than "we do not know".

#### Scenario: One fetch serves both columns

- **WHEN** a page request is served
- **THEN** the stat provider is called once for that season and week

#### Scenario: A failed fetch serves no page

- **WHEN** the stat provider returns an error or a non-200 status
- **THEN** the response is `502 Bad Gateway` and the body is not a scoreboard

### Requirement: A week that is not a matchup is refused, not guessed

A week directory the lineup tree cannot resolve to exactly two rosters, one of them the Bojjaes',
SHALL be refused with an error response naming the problem. The page SHALL NOT be rendered from one
roster, from two of three rosters, or from a matchup the Bojjaes are not in.

A week directory that does not exist SHALL be refused with `404 Not Found`. A week directory that
exists but does not hold exactly one Bojjaes matchup SHALL be refused with `500 Internal Server
Error`, because the request was well formed and the served lineup tree is wrong.

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

- **WHEN** one of the week's two roster files has a line with no id
- **THEN** the response is `500 Internal Server Error` and no page is rendered

### Requirement: The page implies no winner

The rendered markup and its styling SHALL NOT contain a margin, a difference between the two
totals, a win probability, a progress bar, a leader highlight, or any styling that distinguishes
the leading column from the trailing one.

A starter whose game has not kicked off is indistinguishable on this page from one who was
inactive, so any winner-implying element would present an unsettled reading as a result.

#### Scenario: No margin appears

- **WHEN** the two columns total 56 and 42
- **THEN** the page shows both totals and nowhere shows their difference

#### Scenario: The leading column is styled like the trailing one

- **WHEN** one column's total is higher than the other's
- **THEN** the two columns carry the same markup and classes, with nothing marking either as ahead

### Requirement: Roster text is escaped as HTML

Team names and player names reach the page from file names and hand-edited CSV files and SHALL be
rendered through contextual escaping, so a name containing markup is displayed as text rather than
interpreted.

#### Scenario: A name containing markup is displayed as text

- **WHEN** a roster record's name contains `<b>`
- **THEN** the page renders those characters as text and the response contains no unescaped `<b>` tag from that name
