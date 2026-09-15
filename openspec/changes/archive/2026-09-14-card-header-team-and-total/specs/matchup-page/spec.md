## MODIFIED Requirements

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
