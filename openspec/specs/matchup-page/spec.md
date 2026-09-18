# matchup-page

## Purpose

Serve the week's matchup as an HTML page at `GET /{season}/{week}`: two equal columns of scored
starters, the Bojjaes on the left and their opponent on the right, each column a team name, its
nine starters with points, and that lineup's total.

This capability governs the served page — its URL, how a season and week resolve to two columns,
what each column contains, and how a week that is not a matchup is refused. The two teams are
resolved as `lineup-source` resolves them and the points are `player-week-score`'s; nothing here
parses a roster file or scores a stat line.

Nothing here compares the two columns. There is deliberately no margin, win probability, progress
bar, or leader highlight, because a starter whose game has not kicked off is indistinguishable on
this page from one who was inactive, and any winner-implying element would present an unsettled
reading as a result.

## Requirements

### Requirement: The page is addressed by season and week

The server SHALL serve the matchup page at `GET /{season}/{week}`, where both segments are decimal
integers, e.g. `/2025/15`. No team appears in the URL: the week directory names both teams.

A segment that is not an integer, or a season or week outside the plausible range `internal/score`
validates, SHALL be refused with `400 Bad Request` and SHALL NOT reach the lineup tree or the
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

The team name and the total SHALL share one heading at the top of the column, above the first
starter: the name at the left and the total at the right, on the same line. Nothing in the column
SHALL be rendered above the heading, and the total SHALL NOT be repeated anywhere else in the
column. A team name too long for the space beside the total SHALL wrap under itself, and SHALL NOT
push the total out of line or out of the column.

The heading SHALL be set apart from the starters by a tinted band across the full width of the
column and a dividing line below it. The team name SHALL be set in capitals and the total in bold
tabular figures. The band, the line, and the type SHALL be identical in both columns and SHALL NOT
depend on points or on which total is larger. No line SHALL be drawn above the total.

Each starter line SHALL show the player's name from the roster file and that player's points for
the requested season and week, computed by the `player-week-score` rules.

The name SHALL be rendered in both of its forms: the long form, which is the record's name, and
the short form, which is the record's short name as `lineup-source` provides it. Exactly one form
SHALL be visible. The short form is visible when the column is too narrow to set the league's
longest long names beside their points, and the long form otherwise. The choice SHALL depend only
on the width available to the column. It SHALL NOT depend on the device, the orientation, or
either column's points. Both columns SHALL use the same markup for both forms.

Each starter line SHALL also show the record's position and team, in that order, separated by a
middle dot (`WR · CIN`), on a line of its own below the name. The position and team SHALL be shown
for every starter, including starters the provider has no stats for, at every column width, and
beside either name form. They SHALL be set smaller than the name and in a muted colour, identically
in both columns.

The two columns SHALL be rendered side by side and at equal width at every viewport width,
including a phone held in portrait, and neither column's length or content SHALL change the other's
width.

#### Scenario: A column totals its starters

- **WHEN** a team's nine starters score 12, 0, 6, 3, 9, 0, 15, 4, and 7 points
- **THEN** that column lists all nine players with those points and shows a total of 56

#### Scenario: The total sits beside the team name

- **WHEN** a page is rendered for a week whose two teams are `bojjaes` and `wood`
- **THEN** each column's total is in the same heading element as its team name, on the same line
  as the name, and above the column's first starter

#### Scenario: Nothing sits above the heading

- **WHEN** the page is viewed in a browser
- **THEN** in each column the heading's top is above every other element of the column, and no
  total or dividing line is rendered above the team name

#### Scenario: A phone fits the heading on one line

- **WHEN** the page is viewed on a viewport 390 CSS px wide with `bojjaes` totalling `112.5`
- **THEN** the team name and total are on one line, the total is inside the column, and the two do
  not overlap

#### Scenario: A long team name wraps under itself

- **WHEN** a team name does not fit on one line beside its total
- **THEN** the name wraps onto further lines at the left, and the total stays on the name's first
  line inside the column

#### Scenario: Bench players are not rendered

- **WHEN** a roster file holds twelve records
- **THEN** the column shows the first nine and the page contains no mention of the last three

#### Scenario: Every starter line carries both name forms

- **WHEN** a page is rendered for a week whose starters are read from the lineup tree
- **THEN** each starter line contains that record's name and that record's short name, in both
  columns, including starters the provider has no stats for

#### Scenario: Every starter line carries its position and team

- **WHEN** a page is rendered for a week whose starters include `Ja'Marr Chase,WR,CIN`
- **THEN** that starter's line contains `WR · CIN` below the name, in its own element inside the
  starter line

#### Scenario: A starter without stats still shows position and team

- **WHEN** a starter has no entry in the provider's stats
- **THEN** that starter's line shows `--` for points and still shows its position and team

#### Scenario: A portrait phone shows short names side by side

- **WHEN** the page is viewed on a viewport 390 CSS px wide
- **THEN** the two columns are side by side at equal width and every starter line shows the short
  form of the name, with the position and team below it

#### Scenario: A wide column shows long names

- **WHEN** the page is viewed on a viewport 768 CSS px wide, or on a phone held in landscape
- **THEN** every starter line shows the long form of the name, with the position and team below it

### Requirement: A starter's points hold their place in the row

A starter's point value, including the `--` placeholder, SHALL be rendered on one line and
right-aligned. It SHALL sit in a box that has the same minimum width on every row, wide enough for a
five-character value such as `112.5`, so that the points form one aligned column down each card.
The box SHALL NOT shrink to make room for a name.

A name too long for the space beside the points box SHALL wrap onto further lines under itself.
The name SHALL NOT be truncated or elided, and it SHALL NOT push the points box out of line or out
of the column.

The two column totals SHALL sit at the same height, whichever column has more wrapped lines in its
team name or its starters.

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

- **WHEN** one column's team name wraps onto two lines and the other's does not, or one column has a
  wrapped starter name and the other has none
- **THEN** the two column totals are rendered at the same height

### Requirement: Starters are separate rows that line up across the columns

Each starter's name, points, position, and team SHALL read as one row, visibly separated from the
starters above and below it by a dividing line between adjacent rows. The points SHALL sit on the
same line as the first line of the name. The separation SHALL NOT use a per-row background, a
per-row box, or any styling that depends on points or on which column is ahead, and both columns
SHALL be styled identically.

The two columns SHALL share their row heights: the headings, which hold the team names and totals,
SHALL start at the same height and occupy the same height, and the Nth starter in each column SHALL
start at the same height and occupy the same height. This SHALL hold whichever column has more
wrapped lines, and when a column has fewer than nine starters.

Separating the rows SHALL NOT add horizontal padding or borders inside a row, so the width left for
a name beside the points box is unchanged.

#### Scenario: A wrapped name on one side does not stagger the rows

- **WHEN** the third starter's name wraps onto two lines in the left column and no name wraps in the
  right column
- **THEN** the fourth starter in each column starts at the same height, and the dividers between
  rows sit at the same heights in both columns

#### Scenario: A wrapped team name on one side does not stagger the rows

- **WHEN** the left column's team name wraps onto two lines and the right column's does not
- **THEN** both headings occupy the same height and the first starter in each column starts at the
  same height

#### Scenario: Adjacent starters are divided

- **WHEN** a column shows nine starters
- **THEN** a dividing line separates each pair of adjacent starters, and nothing separates a name
  from its own position and team line

#### Scenario: Points sit on the name's line

- **WHEN** a starter's row has a name line and a position-and-team line
- **THEN** the points are on the name's first line, right-aligned in the points box

#### Scenario: A short column still lines up

- **WHEN** one roster has nine starters and the other has eight
- **THEN** the first eight rows line up across the columns and the two totals are at the same height

#### Scenario: Row styling does not depend on the score

- **WHEN** one column's total is higher than the other's, or one starter outscores the rest
- **THEN** every starter row and heading in both columns carries the same classes and styling

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


### Requirement: The page names its week and links to the adjacent weeks of its season

The page SHALL show a week bar above both columns and outside either one. The bar SHALL hold a
label naming the season and week the page shows, in the form `2026 · Week 2`.

The bar SHALL hold a link to the previous week at its left when that week exists, and a link to the
next week at its right when that week exists. The previous week is the same season, week − 1. The
next week is the same season, week + 1. A link SHALL NOT cross into another season: week 1 has no
previous link, and a season's last week has no next link, even if the other season has lineups.
Whether a week exists SHALL be decided as `lineup-source` decides it, so a week whose directory is
not a well-formed matchup still gets a link.

Each link SHALL be an ordinary hyperlink to that week's page (`/{season}/{week}`). The previous link
SHALL carry `rel="prev"` and the next link `rel="next"`. Each link's text SHALL name the week it
leads to.

Deciding which links to show SHALL NOT call the stats provider, and the page SHALL still fetch only
the week it shows.

When a link is absent, its space SHALL stay empty rather than collapse: the label's horizontal
position SHALL NOT depend on which links are present. The label and both links SHALL fit on one line
on a phone held in portrait. The bar SHALL be styled identically whatever either column's points
are.

#### Scenario: A middle week links both ways

- **WHEN** the tree holds 2026 weeks 1, 2, and 3, and `GET /2026/2` is requested
- **THEN** the page's label reads `2026 · Week 2`, a `rel="prev"` link points to `/2026/1`, and a
  `rel="next"` link points to `/2026/3`

#### Scenario: The first week has no previous link

- **WHEN** the tree holds 2026 weeks 1 and 2, and `GET /2026/1` is requested
- **THEN** the page has a `rel="next"` link to `/2026/2` and no `rel="prev"` link

#### Scenario: The latest week has no next link

- **WHEN** the tree holds 2026 weeks 1 and 2, and `GET /2026/2` is requested
- **THEN** the page has a `rel="prev"` link to `/2026/1` and no `rel="next"` link

#### Scenario: Links do not cross a season boundary

- **WHEN** the tree holds 2025 week 16 and 2026 week 1, and no 2025 week 17 or 2026 week 0
- **THEN** the 2025 week 16 page has no next link and the 2026 week 1 page has no previous link

#### Scenario: A gap in a season is not skipped

- **WHEN** the tree holds 2026 weeks 1 and 3 but not week 2, and `GET /2026/3` is requested
- **THEN** the page has no `rel="prev"` link

#### Scenario: A broken neighbouring week still gets a link

- **WHEN** the tree's 2026 week 3 directory holds three rosters, and `GET /2026/2` is requested
- **THEN** the page has a `rel="next"` link to `/2026/3`

#### Scenario: Only the shown week is fetched

- **WHEN** `GET /2026/2` is served and weeks 1 and 3 both exist
- **THEN** the stats provider is asked for 2026 week 2 exactly once and for no other week

#### Scenario: The label does not move when a link is absent

- **WHEN** the page is viewed in a browser once with both links and once with only a next link
- **THEN** the label's horizontal position is the same in both

#### Scenario: A phone fits the week bar on one line

- **WHEN** the page is viewed on a viewport 390 CSS px wide with both links present
- **THEN** the previous link, the label, and the next link share one line and do not overlap

### Requirement: A season alone redirects to its latest week

The server SHALL serve `GET /{season}`, where the segment is a decimal integer, by redirecting to
`/{season}/{week}` for the greatest week the lineup tree holds a directory for in that season, per
`lineup-source`'s latest-week answer. The redirect SHALL NOT depend on that week being a
well-formed matchup: the same "exists, not resolved" rule the week bar's neighbour links use.

A segment that is not an integer, or a season outside the plausible range `internal/score`
validates, SHALL be refused with `400 Bad Request` and SHALL NOT reach the lineup tree.

A season with no week directories in the tree SHALL be refused with `404 Not Found`.

Resolving the redirect SHALL read only the lineup tree's directory listing. It SHALL NOT open a
roster file and SHALL NOT call the stats provider.

#### Scenario: A season redirects to its highest week

- **WHEN** the tree holds 2026 weeks 1 and 2, and `GET /2026` is requested
- **THEN** the response redirects to `/2026/2`

#### Scenario: A gap does not change the target

- **WHEN** the tree holds 2026 weeks 1 and 3 but not week 2, and `GET /2026` is requested
- **THEN** the response redirects to `/2026/3`

#### Scenario: A non-numeric segment is refused

- **WHEN** `GET /twentytwentysix` is requested
- **THEN** the response is `400 Bad Request` and no directory is read

#### Scenario: An out-of-range season is refused

- **WHEN** `GET /1899` is requested
- **THEN** the response is `400 Bad Request`

#### Scenario: A season with no weeks is a 404

- **WHEN** the tree holds no directories under 2027, and `GET /2027` is requested
- **THEN** the response is `404 Not Found`

#### Scenario: A season whose only week is not a matchup still redirects

- **WHEN** the tree's `2026/3/` directory holds three rosters and no higher week exists under 2026
- **THEN** `GET /2026` redirects to `/2026/3`, and `/2026/3` itself is served (or refused) by the
  existing season/week handler

#### Scenario: Resolving the redirect makes no upstream fetch

- **WHEN** `GET /2026` is requested
- **THEN** the stats provider is not called
