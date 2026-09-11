## MODIFIED Requirements

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

## ADDED Requirements

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
