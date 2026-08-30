# matchup-report

## Purpose

Answer "how am I doing against them" on one screen: two rosters for the same season and week, set
side by side, each column a complete single-roster report.

This capability governs only selection and arrangement — which two teams are shown, which holds the
left column, and how the two reports are aligned. The content of each column is the
`roster-score-report` capability's, unchanged; the points within it are the server's
`player-week-score`. Nothing here scores, groups, or judges: there is deliberately no margin and no
winner, because a starter whose game has not kicked off is indistinguishable from one who was
inactive.

## Requirements

### Requirement: A matchup names one or two teams

The matchup report SHALL accept a season, a week, and one or two team arguments.

Given one team argument, the report SHALL treat that team as the Bojjaes' opponent and show the
Bojjaes against them. Given two team arguments, the report SHALL show those two teams and SHALL NOT
add the Bojjaes. The report SHALL reject zero team arguments and three or more.

Team arguments SHALL be resolved to roster files exactly as the single-roster report resolves them,
so a name means the same file in both tools.

#### Scenario: One team is read as the Bojjaes' opponent

- **WHEN** the report is asked for one season, one week, and the single team `wood`
- **THEN** it shows the Bojjaes' roster and Wood's roster for that season and week

#### Scenario: Two teams are shown as given

- **WHEN** the report is asked for two teams, neither of which is the Bojjaes
- **THEN** it shows those two rosters and the Bojjaes do not appear

#### Scenario: No team argument is refused

- **WHEN** the report is asked for a season and week with no team argument
- **THEN** it prints usage and exits non-zero without contacting the server

### Requirement: The named team order fixes the column order

The team shown in the left column SHALL be the Bojjaes in the one-argument form, and the
first-named team in the two-argument form. The remaining team SHALL occupy the right column.

#### Scenario: The Bojjaes hold the left column

- **WHEN** the report is asked for a single opponent
- **THEN** the Bojjaes are the left column and the opponent the right

#### Scenario: Argument order is column order

- **WHEN** the report is asked for `aroma` then `fuego`
- **THEN** Aroma is the left column and Fuego the right

### Requirement: Each column is a full roster report

Each column SHALL contain that team's complete single-roster report for the season and week —
starters, starter total, and bench — with the same content, ordering, and no-stats reporting the
single-roster report produces on its own.

The two columns SHALL be produced independently, one server request per team, so that neither
roster's length or contents can change how the other is scored or grouped.

#### Scenario: A column carries the whole report

- **WHEN** a matchup is shown
- **THEN** each column carries that team's starters heading, its players, its starter total, and its
  bench heading

#### Scenario: Columns are scored independently

- **WHEN** the two rosters together hold more players than one server request accepts
- **THEN** the report still succeeds, because each roster is scored in its own request

### Requirement: The season and week are stated once for the matchup

The report SHALL print the season and week once, above both columns, and SHALL NOT repeat them
inside either column.

#### Scenario: The heading is not duplicated per team

- **WHEN** a matchup is shown
- **THEN** the season and week appear exactly once in the output

### Requirement: Columns are aligned regardless of line content

Every line of the right column SHALL begin at the same character offset, whatever the left column's
line holds — a player line, a heading, a total, or a blank separator.

#### Scenario: A blank line does not collapse the right column

- **WHEN** the left column has a blank separator line
- **THEN** the right column's text on that line still begins at the same offset as every other line

#### Scenario: Short and long names do not shift the right column

- **WHEN** the left column holds player names of differing lengths
- **THEN** the right column begins at the same offset on each of those lines

### Requirement: Rosters of different lengths do not distort either column

Each column SHALL be laid out on its own. When one roster holds more players than the other, the
report SHALL NOT pad, truncate, or reorder either roster's players to make the columns's sections
line up horizontally, and SHALL NOT drop lines from the longer column.

#### Scenario: The longer roster keeps all of its lines

- **WHEN** the left roster holds twelve players and the right holds nine
- **THEN** every one of the left roster's twelve players is printed

#### Scenario: The shorter column simply ends

- **WHEN** the right roster's report is shorter than the left's
- **THEN** the remaining lines show only the left column, with nothing to their right

### Requirement: The matchup carries no scoreboard line

The report SHALL NOT print a margin, a difference, a projected result, or a winner. The two starter
totals sit on the same line as each other and are the comparison.

The reason is the server's no-stats contract: a starter whose game has not kicked off is
indistinguishable from one who was inactive, so a computed margin would read as a settled result
while it is nothing of the kind.

#### Scenario: Totals face each other with no verdict

- **WHEN** both columns' starter totals are printed
- **THEN** they appear on the same output line and no margin, difference, or winner is printed

#### Scenario: A missing starter produces no warning of its own

- **WHEN** a starter in either column has no stats
- **THEN** that player's line reports no stats and the matchup adds no annotation beyond it
