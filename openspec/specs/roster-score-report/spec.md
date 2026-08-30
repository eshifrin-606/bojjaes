# roster-score-report

## Purpose

Turn a saved roster file into a readable weekly report: which players are the starting lineup and
which are bench, what the lineup scored in total, and how a player the server has no stats for is
reported.

The report is a reading of a scored roster, not a scoring rule. Points come from the server's
`player-week-score` capability; this capability governs only how those points are grouped,
labelled, and summed for a manager reading the terminal. The players file is the lineup card:
maintained by hand, ordered by hand, and trusted as written.

## Requirements

### Requirement: Roster rows are split into starters and bench

The report SHALL treat the first nine records of the players file as the starting lineup and every
remaining record as bench. Record order SHALL be file order; comment and blank lines are not
records and SHALL NOT count toward the nine.

The split is positional only. The players file carries no position column, so the report SHALL NOT
attempt to validate that the starters form a legal lineup, and SHALL NOT reorder records to produce
one.

#### Scenario: A roster longer than a lineup is split

- **WHEN** a players file holds twelve records
- **THEN** the first nine appear under the starters section and the last three under the bench
  section, each in file order

#### Scenario: Comments and blank lines do not consume a starter slot

- **WHEN** a players file begins with a comment line and holds nine records
- **THEN** all nine records are starters and none is pushed onto the bench

#### Scenario: Record order is not rearranged

- **WHEN** the records of a players file are reordered
- **THEN** the report reflects the new order, and which players are starters changes accordingly

### Requirement: Both sections are always labelled

The report SHALL print a starters heading and a bench heading, in that order, on every run. The
bench heading SHALL be printed even when no records follow the ninth, so that the sectioned shape
of the report does not depend on roster length.

#### Scenario: A roster of exactly a lineup still shows both headings

- **WHEN** a players file holds exactly nine records
- **THEN** the report shows nine starters and a bench heading with no players under it

#### Scenario: A roster shorter than a lineup is all starters

- **WHEN** a players file holds five records
- **THEN** all five appear under the starters heading and the bench heading has no players under it

### Requirement: Starters are totalled

The report SHALL print a single aggregated point total for the starters, after the last starter and
before the bench heading. The total SHALL be the sum of the points of the starters the server
returned a score for.

The total SHALL NOT include bench players. The report SHALL NOT print a bench total, because bench
points count for nothing in the league.

#### Scenario: The starter total sums the starting lineup only

- **WHEN** starters score 3.5, 19, 0, 6, 6, 31, 4, and 3, and a bench player scores 12
- **THEN** the total reads 72.5

#### Scenario: Bench players are scored but not totalled

- **WHEN** the bench holds scored players
- **THEN** each bench player's points are printed and no bench total appears

#### Scenario: Fractional awards total exactly

- **WHEN** starters' points include half-point values
- **THEN** the total is their exact sum, with no rounding or representation drift

### Requirement: A starter without stats contributes nothing to the total

A player the server reports no stats for SHALL be printed as having no stats rather than as zero,
in both sections, preserving the distinction the server draws between an absent stat line and a
real score of zero.

When such a player is a starter, the total SHALL be the sum of the starters that do have stats, and
SHALL NOT be annotated as incomplete. The player's own line already reports the absence directly
above the total.

#### Scenario: A missing starter is reported and skipped

- **WHEN** one of nine starters is in the server's no-stats list
- **THEN** that starter's line reports no stats, and the total is the sum of the other eight

#### Scenario: The total is not marked when a starter is missing

- **WHEN** a starter has no stats
- **THEN** the total is printed as a plain number, carrying no marker, count, or warning of its own

#### Scenario: No starter has stats

- **WHEN** the server returns no stats for any starter
- **THEN** every starter line reports no stats and the total is zero
