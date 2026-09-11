# matchup-page

## Purpose

Serve the week's matchup as an HTML page at `GET /{season}/{week}`: two equal columns of scored
starters, the Bojjaes on the left and their opponent on the right, each column a team name, its
nine starters with points, and that lineup's total.

This capability governs the served page — its URL, how a season and week resolve to two columns,
what each column contains, and how a week that is not a matchup is refused. The two teams are
resolved as `lineup-source` resolves them and the points are `player-week-score`'s; nothing here
parses a roster file or scores a stat line. `matchup-report` governs the terminal report that
answers the same question on one machine, and the two exist side by side until the page replaces
it.

Nothing here compares the two columns. There is deliberately no margin, win probability, progress
bar, or leader highlight, because a starter whose game has not kicked off is indistinguishable on
this page from one who was inactive, and any winner-implying element would present an unsettled
reading as a result.

## Requirements

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
`lineup-source` capability resolves them. The Bojjaes SHALL hold the left column and their opponent
the right, regardless of file name order in the directory.

#### Scenario: The Bojjaes hold the left column

- **WHEN** a week directory holds `bojjaes.csv` and `wood.csv`
- **THEN** the left column is the Bojjaes and the right column is `wood`

#### Scenario: An alphabetically earlier opponent still holds the right column

- **WHEN** a week directory holds `bojjaes.csv` and `aardvarks.csv`
- **THEN** the left column is still the Bojjaes and `aardvarks` is on the right

### Requirement: Each column is a team, its starters, and one total

Each column SHALL show the team name, that team's starting lineup in file order, and the total of
the starters' points. Starters are the roster's first nine records per `lineup-source`; bench
players SHALL NOT appear on the page.

Each starter line SHALL show the player's name from the roster file and that player's points for
the requested season and week, computed by the `player-week-score` rules.

The name SHALL be rendered in both of its forms: the long form, which is the record's name, and
the short form, which is the record's short name as `lineup-source` provides it. Exactly one form
SHALL be visible. The short form is visible when the column is too narrow to set the league's
longest long names beside their points, and the long form otherwise. The choice SHALL depend only
on the width available to the column. It SHALL NOT depend on the device, the orientation, or
either column's points. Both columns SHALL use the same markup for both forms.

The two columns SHALL be rendered side by side and at equal width at every viewport width,
including a phone held in portrait, and neither column's length or content SHALL change the other's
width.

#### Scenario: A column totals its starters

- **WHEN** a team's nine starters score 12, 0, 6, 3, 9, 0, 15, 4, and 7 points
- **THEN** that column lists all nine players with those points and shows a total of 56

#### Scenario: Bench players are not rendered

- **WHEN** a roster file holds twelve records
- **THEN** the column shows the first nine and the page contains no mention of the last three

#### Scenario: Every starter line carries both name forms

- **WHEN** a page is rendered for a week whose starters are read from the lineup tree
- **THEN** each starter line contains that record's name and that record's short name, in both
  columns, including starters the provider has no stats for

#### Scenario: A portrait phone shows short names side by side

- **WHEN** the page is viewed on a viewport 390 CSS px wide
- **THEN** the two columns are side by side at equal width and every starter line shows the short
  form of the name

#### Scenario: A wide column shows long names

- **WHEN** the page is viewed on a viewport 768 CSS px wide, or on a phone held in landscape
- **THEN** every starter line shows the long form of the name

### Requirement: A starter's points hold their place in the row

A starter's point value, including the `--` placeholder, SHALL be rendered on one line and
right-aligned. It SHALL sit in a box that has the same minimum width on every row, wide enough for a
five-character value such as `112.5`, so that the points form one aligned column down each card.
The box SHALL NOT shrink to make room for a name.

A name too long for the space beside the points box SHALL wrap onto further lines under itself.
The name SHALL NOT be truncated or elided, and it SHALL NOT push the points box out of line or out
of the column.

The two column totals SHALL sit at the same height, whichever column has more wrapped lines.

At narrow viewports the page SHALL use less padding and spacing than it does at wide viewports, so
the space goes to the names rather than to margins.

#### Scenario: The placeholder never breaks

- **WHEN** a starter with no stats is shown on a viewport 390 CSS px wide
- **THEN** that starter's `--` is on a single line

#### Scenario: Points line up down a column

- **WHEN** a column's starters have names of different lengths, some of them wrapping
- **THEN** every starter's points have the same right edge and the name area beside them has the
  same width on every row

#### Scenario: A long name wraps under itself

- **WHEN** a name does not fit on one line beside the points box
- **THEN** the full name is visible over several lines beneath its own start, no part of it is
  replaced by an ellipsis, and the points box keeps its width and its alignment with the other rows

#### Scenario: Totals line up

- **WHEN** one column has a wrapped starter name and the other has none
- **THEN** the two column totals are rendered at the same height

### Requirement: A starter the provider has no stats for shows no number

A starter absent from the provider's weekly payload SHALL be rendered with the placeholder `--` in
place of a point value, not with `0`, and SHALL contribute nothing to the column total. A starter
present in the payload with no scoring production SHALL be rendered as `0` and is a real scoreless
line.

Absence and a scoreless week are distinguishable on the page because they are different facts: the
payload cannot say whether a missing player has not kicked off, is inactive, or does not exist.

The column total SHALL always be rendered as a number, never as the placeholder, including when
every starter in the column is absent.

#### Scenario: A missing starter is not zero

- **WHEN** one of nine starters has no entry in the weekly payload and the other eight total 40
- **THEN** that starter's line shows `--` rather than `0` and the column total is 40

#### Scenario: A scoreless starter is zero

- **WHEN** a starter has an entry in the weekly payload but scores nothing under the rules
- **THEN** that starter's line shows `0`

#### Scenario: A column with no stats at all totals zero

- **WHEN** none of a column's nine starters has an entry in the weekly payload
- **THEN** every starter line in that column shows `--` and the column total shows `0`

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

Team names and player names reach the page from file names and hand-edited CSV files. They SHALL be
rendered through contextual escaping, so a name containing markup is displayed as text rather than
interpreted. This applies to both the long and the short form of a player's name.

#### Scenario: A name containing markup is displayed as text

- **WHEN** a roster record's name contains `<b>`
- **THEN** the page renders those characters as text and the response contains no unescaped `<b>` tag from that name

#### Scenario: A short name containing markup is displayed as text

- **WHEN** a roster record's short name contains `<b>`
- **THEN** the short form is rendered as text equal to that short name and the response contains no
  unescaped `<b>` tag from it

### Requirement: The page states when its stats were fetched

The page SHALL show the instant its weekly stats were fetched from the provider, once, above or
below both columns and never inside either. The instant SHALL be the time the fetch that produced
the served stats completed — so a page served from cache states the original fetch, not the time of
the request it answered.

The instant SHALL be rendered both machine-readably and human-readably: a `<time>` element whose
`datetime` attribute carries the fetch instant as an RFC 3339 string, with visible text showing
that instant as a wall-clock time in `America/Chicago`, the league's timezone.

The visible text SHALL be labelled so a reader reads it as when the server fetched the data, not as
the current time and not as when the upstream stats last changed. The label SHALL NOT claim the
page is "live" or "current".

The rendering SHALL carry no relative phrasing ("4 minutes ago"); that is left to a later change,
which the machine-readable `datetime` attribute exists to enable.

#### Scenario: The fetch instant appears once, machine- and human-readable

- **WHEN** a page is served whose stats were fetched at a known instant
- **THEN** the page contains exactly one `<time>` element whose `datetime` is that instant in RFC
  3339 and whose visible text is that instant as an `America/Chicago` wall-clock time

#### Scenario: A cache hit reports the fetch, not the request

- **WHEN** the stats are served from a cache entry whose fetch completed three minutes before this
  request
- **THEN** the stated instant is that fetch's completion time, three minutes ago, not the current
  request time

#### Scenario: The label does not overclaim currency

- **WHEN** the page renders its as-of line
- **THEN** the surrounding text identifies the instant as when the data was fetched and nowhere
  describes the page as live or current

#### Scenario: The as-of line implies no winner

- **WHEN** the two columns total 56 and 42 and the as-of line is rendered
- **THEN** the as-of line references neither total, sits outside both columns, and adds no margin,
  difference, or winner to the page
