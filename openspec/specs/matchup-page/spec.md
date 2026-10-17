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

The rendered markup, its styling, and any script it carries SHALL NOT contain a margin, a
difference between the two totals, a win probability, a progress bar, a leader highlight, or any
styling or behaviour that distinguishes the leading column from the trailing one.

A starter whose game has not kicked off is indistinguishable on this page from one who was
inactive, so any winner-implying element would present an unsettled reading as a result.

The refresh SHALL NOT create one by other means: it SHALL NOT compare a newly fetched total against
the one previously displayed, and SHALL NOT animate, highlight, or otherwise mark a total that has
changed since the last load.

#### Scenario: No margin appears

- **WHEN** the two columns total 56 and 42
- **THEN** the page shows both totals and nowhere shows their difference

#### Scenario: The leading column is styled like the trailing one

- **WHEN** one column's total is higher than the other's
- **THEN** the two columns carry the same markup and classes, with nothing marking either as ahead

#### Scenario: A refresh marks no change

- **WHEN** a refresh renders a total different from the one the previous load showed
- **THEN** the new page is rendered exactly as a first load of it would be, with nothing marking
  which values moved

### Requirement: The page refreshes itself while its tab is visible

The served page SHALL re-fetch itself on a recurring interval of approximately five minutes, so a
page left open during a game does not go silently stale.

The refresh SHALL be conditioned on the tab being visible at the moment it would fire. While the
tab is hidden the page SHALL make no refresh request, so upstream load is bounded by who is
actually looking rather than by how many tabs are open.

The refresh SHALL be a reload of the same URL. The page SHALL NOT fetch a fragment, swap markup in
place, or hold a persistent connection.

The version check defined below is the one other request the page may make. It is a short request
that returns no markup, and its only effect on the page is to cause the same reload of the same URL.

#### Scenario: A visible tab refreshes on the interval

- **WHEN** the page has been open and visible for the refresh interval
- **THEN** the page re-requests its own URL and renders the newly fetched scores

#### Scenario: A hidden tab makes no request

- **WHEN** the page's tab is hidden and the refresh interval elapses
- **THEN** no request is made for the page and no stat fetch is triggered on its behalf

#### Scenario: An hour hidden is not an hour of requests

- **WHEN** the page's tab is hidden for an hour spanning several refresh intervals
- **THEN** the number of refresh requests made while it was hidden is zero, not one per interval

#### Scenario: A version check swaps no markup

- **WHEN** a version check finds the server's version differs from the page's
- **THEN** the page reloads its own URL, and nothing from the check's response is written into the
  document

### Requirement: Returning to a stale tab refreshes immediately

When the tab becomes visible again and at least the refresh interval has elapsed since the page
was last loaded, the page SHALL refresh immediately rather than waiting out the remainder of a
timer.

A reader returning to the tab is the moment the page is most likely to be stale and most likely to
be believed, so what they see on returning SHALL NOT be older than the refresh interval by more
than the time the reload itself takes.

#### Scenario: A tab hidden past the interval refreshes on return

- **WHEN** the tab has been hidden for longer than the refresh interval and becomes visible
- **THEN** the page refreshes at once

#### Scenario: A brief switch away does not refresh

- **WHEN** the tab is hidden and made visible again well within the refresh interval
- **THEN** the page does not refresh on return and the existing interval continues

### Requirement: The refresh script carries no page data

The refresh script SHALL contain no interpolated value from the rendered page. No team name, player
name, point value, season, or week SHALL be written into the script element.

Team and player names reach the page from file names and hand-edited CSV files. Keeping them out of
the script keeps every one of them in the HTML text contexts the existing escaping requirement
already covers, rather than in a JavaScript context.

#### Scenario: A roster name never reaches the script

- **WHEN** a page is rendered for a week whose rosters contain any team and player names
- **THEN** the script element in the response contains none of those names

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

### Requirement: The page states the version of the stats it was rendered from

The page SHALL carry a stats version: a token that names the fetch the served stats came from. Two
pages rendered from the same fetch SHALL carry the same token, and pages rendered from different
fetches SHALL carry different tokens, including two fetches that complete within the same second.

The token SHALL be rendered once, as an attribute of an element outside the script. It is opaque to
the page: the script SHALL compare it for equality only and SHALL NOT parse it.

#### Scenario: The version names the fetch

- **WHEN** a page is served whose stats were fetched at a known instant
- **THEN** the page carries exactly one stats version, and it is the token for that instant

#### Scenario: A cache hit carries the original fetch's version

- **WHEN** two pages are served from the same cached fetch, minutes apart
- **THEN** both carry the same stats version

#### Scenario: Two fetches in one second have different versions

- **WHEN** two pages are served from two fetches that completed within the same second
- **THEN** their stats versions differ

### Requirement: The server reports a week's current stats version

`GET /{season}/{week}/version` SHALL report the stats version the server currently holds for that
season and week, in the same token form the page carries, so the two can be compared as strings.

When the server holds no completed fetch for that week, the endpoint SHALL report an empty version.
This covers a week never fetched, a week whose stats were invalidated, a week whose fetch is in
flight, and a server that has restarted.

Answering SHALL NOT call the stat provider, SHALL NOT start or wait on a fetch, and SHALL NOT read
the lineup tree. The response SHALL be marked as not to be stored by any cache.

The season and week SHALL be validated as the page's are: a segment that is not a number, or a
season or week out of range, SHALL be refused with status 400.

#### Scenario: The endpoint reports the version a fresh page carries

- **WHEN** a page for week 4 is served and the version endpoint for week 4 is then asked
- **THEN** the endpoint's version equals the page's stats version

#### Scenario: Invalidation empties the version

- **WHEN** week 4's stats are fetched and then invalidated, and the version endpoint is asked
- **THEN** the version reported is empty

#### Scenario: A refetch changes the version

- **WHEN** week 4's stats are fetched, invalidated, and fetched again, and the version endpoint is asked
- **THEN** the version reported is the second fetch's, and differs from the first page's

#### Scenario: Asking costs nothing upstream

- **WHEN** the version endpoint is asked for a week, whether or not that week has been fetched
- **THEN** no call is made to the stat provider

#### Scenario: A malformed week is refused

- **WHEN** `GET /2026/five/version` is requested
- **THEN** the response status is 400 and no version is reported

#### Scenario: The response is not cacheable

- **WHEN** the version endpoint answers
- **THEN** the response carries `Cache-Control: no-store`

### Requirement: The page reloads when the server's stats version differs from its own

While its tab is visible, the page SHALL check the version endpoint for its own season and week and
compare the reported version with the one it carries. When they differ, including when the reported
version is empty, the page SHALL reload its own URL.

The check SHALL be conditioned on the tab being visible at the moment it would fire. While the tab is
hidden the page SHALL make no check request.

A check that fails or cannot be read SHALL NOT reload the page. The page SHALL check again after the
delay it last used.

The reload SHALL be subject to every rule that binds the timed refresh: it compares no totals, marks
nothing as changed, and keeps no state across loads.

#### Scenario: A superseded page reloads

- **WHEN** a visible page carries one stats version and a check reports a different one
- **THEN** the page reloads its own URL

#### Scenario: An invalidated week reloads the page

- **WHEN** a visible page's week is invalidated on the server and the next check reports an empty
  version
- **THEN** the page reloads its own URL, and that reload's request fetches the week afresh

#### Scenario: A matching version does nothing

- **WHEN** a check reports the same stats version the page carries
- **THEN** the page does not reload

#### Scenario: A hidden tab makes no check

- **WHEN** the page's tab is hidden when a check would fire
- **THEN** no request is made to the version endpoint

#### Scenario: A failed check does not reload

- **WHEN** a check request fails
- **THEN** the page does not reload, and checks again after the same delay

### Requirement: The server directs how often the page checks

The delay before each check SHALL come from the server, not from a constant in the script. The served
page SHALL state the delay before the first check, as an attribute outside the script, and every
version endpoint response SHALL state the delay before the next one.

A delay of zero SHALL mean stop: the page SHALL make no further check until it is next loaded. The
timed refresh is unaffected by a stop.

The server SHALL give the same delay, approximately ten seconds, on every page and every response.
Varying it by week or by time is left to a later change, which this requirement exists to allow
without changing the script.

#### Scenario: The page states its first delay

- **WHEN** a page is served
- **THEN** it carries the delay before its first check, and that delay is the server's default

#### Scenario: The endpoint states the next delay

- **WHEN** the version endpoint answers
- **THEN** the response carries the delay before the next check, and it is the server's default

#### Scenario: The page follows a changed delay

- **WHEN** a check response states a delay different from the one the page last used
- **THEN** the page waits the newly stated delay before its next check

#### Scenario: A zero delay stops the checks

- **WHEN** a check response states a delay of zero
- **THEN** the page makes no further check, and still reloads on its five-minute interval

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
