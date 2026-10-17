## ADDED Requirements

### Requirement: Each starter line states its game

Each starter line SHALL state where that starter's NFL game stands in the page's season and week.
The game SHALL be the one the starter's roster team plays that week, as `game-schedule` finds it by
team code. The roster team is the team on the starter's record; no other source SHALL be consulted
for it.

What is stated SHALL be exactly one of:

- **Upcoming:** the kickoff instant as a weekday and a time of day, in a `<time>` element whose
  `datetime` attribute carries the instant as an RFC 3339 string.
- **In progress:** the provider's quarter text. Text made only of digits SHALL be prefixed with `Q`
  (`3` is shown as `Q3`). Any other text SHALL be shown as sent (`HALF`). If the provider gives no
  quarter text, `Live` SHALL be shown.
- **Complete:** the provider's quarter text with its leading `F` spelled out as `Final`, so `F` is
  shown as `Final` and `F/OT` as `Final/OT`. If the provider gives no quarter text, `Final` SHALL be
  shown. Text that does not begin with `F` SHALL be shown as sent.
- **Unrecognised state:** the provider's status text, as sent.
- **Bye:** `Bye`, when the week has games and the roster team is in none of them.

No game clock, score, or opponent SHALL be shown.

The game state SHALL follow the team on the starter's position-and-team line, separated from it by a
middle dot (`WR · CIN · Q3`), in an element of its own. It SHALL be shown for starters the provider
has no stats for. Both columns SHALL use the same markup for it, and it SHALL NOT depend on either
column's points.

A starter whose roster team is `FA` SHALL show no game state and no trailing separator.

The position-and-team line MAY wrap when the column is narrow. It SHALL NOT extend outside its
column at any viewport width, and a wrapped line SHALL NOT break the alignment of starter rows
across the two columns.

#### Scenario: An upcoming game shows its kickoff

- **WHEN** a starter's roster team is `BUF` and the `BUF` game is upcoming with kickoff
  2026-10-13T00:15:00Z
- **THEN** the starter's line contains a `<time>` element whose `datetime` is that instant, whose
  text names a weekday and a time of day

#### Scenario: A numbered quarter

- **WHEN** a starter's game is in progress with quarter text `3`
- **THEN** the starter's line shows `Q3`

#### Scenario: Halftime is shown as sent

- **WHEN** a starter's game is in progress with quarter text `HALF`
- **THEN** the starter's line shows `HALF`

#### Scenario: In progress with no quarter text

- **WHEN** a starter's game is in progress and the provider gives no quarter text
- **THEN** the starter's line shows `Live`

#### Scenario: A finished game

- **WHEN** a starter's game is complete with quarter text `F`
- **THEN** the starter's line shows `Final`

#### Scenario: A game finished in overtime

- **WHEN** a starter's game is complete with quarter text `F/OT`
- **THEN** the starter's line shows `Final/OT`

#### Scenario: An unrecognised status is shown as sent

- **WHEN** a starter's game has the provider status `postponed`
- **THEN** the starter's line shows `postponed`

#### Scenario: A bye

- **WHEN** the week has fifteen games, none involving `KC`, and a starter's roster team is `KC`
- **THEN** the starter's line shows `Bye`

#### Scenario: A free agent shows no game state

- **WHEN** a starter's roster team is `FA`
- **THEN** the starter's line shows the position and `FA` and nothing after them

#### Scenario: The game state sits after the team

- **WHEN** a starter is `Ja'Marr Chase,WR,CIN` and the `CIN` game is complete
- **THEN** the starter's position-and-team line reads `WR · CIN · Final`, with `Final` in its own
  element

#### Scenario: A starter without stats still shows its game

- **WHEN** a starter has no entry in the provider's stats and their game is upcoming
- **THEN** the starter's line shows `--` for points and the kickoff

#### Scenario: No clock is shown

- **WHEN** a starter's game is in the third quarter with 4:01 remaining
- **THEN** the starter's line shows `Q3` and the page does not contain `4:01`

#### Scenario: A phone keeps the line inside its column

- **WHEN** the page is viewed on a viewport 390 CSS px wide with a starter whose game is upcoming
- **THEN** that starter's position-and-team line lies inside its column, and every starter row
  still lines up with its counterpart in the other column

### Requirement: Both columns state their games from one fetch

The page SHALL read the week's games once per request and state every starter's game, in both
columns, from that one reading. Two starters on the same NFL team, or on the two teams of one game,
SHALL NOT be shown states from different readings.

#### Scenario: Teammates agree

- **WHEN** both columns hold a starter whose roster team is `CIN`
- **THEN** both lines show the same game state

#### Scenario: Opponents in one game agree

- **WHEN** one column holds a `DET` starter, the other an `ARI` starter, and `DET` plays at `ARI`
  and is at halftime
- **THEN** both lines show `HALF`

### Requirement: The page is served without game states when the games are unavailable

When the week's games cannot be fetched, the page SHALL still be served, with the same scores,
totals, and as-of line it would otherwise have, and with no game state on any starter line. The
response SHALL be the same status it would be with the games available.

When the week's games are fetched and the week holds no games at all, the page SHALL likewise show
no game state on any starter line. It SHALL NOT show `Bye` for every starter.

A starter line without a game state SHALL be exactly the line the page rendered before game states
existed: the position, a middle dot, and the team, with nothing after.

A failure to fetch the week's stats SHALL still fail the page as before, whether or not the games
were fetched.

#### Scenario: A failed game fetch does not fail the page

- **WHEN** the week's stats are fetched and the week's games fail to fetch
- **THEN** the response is 200, every starter shows its points, and no starter line shows a game
  state

#### Scenario: A week with no games shows no byes

- **WHEN** the week's games are fetched and the list is empty
- **THEN** no starter line shows `Bye` or any other game state

#### Scenario: A line without a game state is unchanged

- **WHEN** the games are unavailable and a starter is `Ja'Marr Chase,WR,CIN`
- **THEN** that starter's position-and-team line reads `WR · CIN` with no trailing middle dot

#### Scenario: A failed stats fetch still fails the page

- **WHEN** the week's stats fail to fetch and the week's games are available
- **THEN** the response is 502 and no page is rendered

### Requirement: Instants are shown in the reader's timezone

Every instant the page shows SHALL be shown in the timezone of the reader's browser when the page
script runs, and in `America/Chicago` otherwise. This covers each kickoff and the as-of line.

The server SHALL render each instant as an `America/Chicago` wall-clock time, as the text of a
`<time>` element whose `datetime` attribute carries the instant as an RFC 3339 string. A served
kickoff SHALL show a weekday and a time of day with no zone label, so the meta line fits a phone. A
served as-of instant SHALL carry the zone abbreviation.

The page script SHALL replace the text of each such element with the same instant in the browser's
timezone, taking the instant from the element's `datetime` attribute. The weekday SHALL be computed
in the same timezone as the time of day, so a kickoff that falls on different days in different
zones is shown on the reader's day.

A rewritten kickoff SHALL show a weekday and a time of day with no zone label. A rewritten as-of
instant SHALL keep the form the server gave it: weekday, date, time of day, and the abbreviation of
the reader's zone.

If the script cannot rewrite an element, it SHALL leave that element's server-rendered text in
place. The rewrite SHALL write no value into the script element: it reads from the document, in
keeping with the requirement that the script carries no page data.

#### Scenario: The server writes Central

- **WHEN** a page is served with a kickoff at 2026-10-11T17:00:00Z
- **THEN** the served markup's `<time>` for that kickoff reads `Sun 12:00 PM`

#### Scenario: A reader in another zone sees their own time

- **WHEN** that page is viewed in a browser whose timezone is `America/New_York` and the script runs
- **THEN** the kickoff reads `Sun 1:00 PM`

#### Scenario: The weekday follows the reader's zone

- **WHEN** a kickoff is at 2026-10-13T00:15:00Z
- **THEN** the served markup reads `Mon 7:15 PM`, and a browser whose timezone is `Europe/London`
  shows `Tue 1:15 AM`

#### Scenario: A reader in Central sees the same clock time

- **WHEN** the page is viewed in a browser whose timezone is `America/Chicago` and the script runs
- **THEN** each kickoff shows the same weekday and time of day as the served markup

#### Scenario: The as-of line is rewritten too

- **WHEN** a page whose stats were fetched at 2026-10-11T22:07:00Z is viewed in a browser whose
  timezone is `America/New_York`
- **THEN** the as-of line reads `Sun, Oct 11 2026 6:07 PM EDT`

#### Scenario: Without the script, Central stays

- **WHEN** the page is viewed with scripts disabled
- **THEN** every instant on the page is the served `America/Chicago` text: kickoffs unlabelled, the
  as-of line with its zone abbreviation

#### Scenario: The rewrite puts no page data in the script

- **WHEN** a page is rendered for a week with known kickoffs and a known fetch instant
- **THEN** the script element contains none of those instants

## MODIFIED Requirements

### Requirement: The page states when its stats were fetched

The page SHALL show the instant its weekly stats were fetched from the provider, once, above or
below both columns and never inside either. The instant SHALL be the time the fetch that produced
the served stats completed — so a page served from cache states the original fetch, not the time of
the request it answered.

The instant SHALL be rendered both machine-readably and human-readably: a `<time>` element whose
`datetime` attribute carries the fetch instant as an RFC 3339 string, with served text showing that
instant as a wall-clock time in `America/Chicago`, the league's timezone. The as-of line SHALL
contain exactly one `<time>` element. Where the page script runs, that text is shown in the reader's
timezone, as the requirement on instants in the reader's timezone defines.

The visible text SHALL be labelled so a reader reads it as when the server fetched the data, not as
the current time and not as when the upstream stats last changed. The label SHALL NOT claim the
page is "live" or "current".

The rendering SHALL carry no relative phrasing ("4 minutes ago"); that is left to a later change,
which the machine-readable `datetime` attribute exists to enable.

The as-of line states the stats fetch only. It SHALL NOT state when the week's games were fetched.

#### Scenario: The fetch instant appears once, machine- and human-readable

- **WHEN** a page is served whose stats were fetched at a known instant
- **THEN** the as-of line contains exactly one `<time>` element whose `datetime` is that instant in
  RFC 3339 and whose served text is that instant as an `America/Chicago` wall-clock time

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

#### Scenario: The games' fetch instant is not stated

- **WHEN** the week's stats and games were fetched at different instants
- **THEN** the as-of line states the stats' instant and the page states no instant for the games
