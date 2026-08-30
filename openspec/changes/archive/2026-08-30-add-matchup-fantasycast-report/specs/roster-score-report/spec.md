## ADDED Requirements

### Requirement: The report carries no season or week heading of its own

The report SHALL begin at the starters heading and SHALL NOT print the season and week it was asked
for. The season and week are the caller's own arguments, and stating them is the caller's job — a
report composed into a wider layout would otherwise repeat them once per roster.

#### Scenario: The report opens on the starters heading

- **WHEN** a roster is scored for a season and week
- **THEN** the first line of output is the starters heading

#### Scenario: The season and week are absent from the report

- **WHEN** a roster is scored
- **THEN** neither the season nor the week appears anywhere in the report

### Requirement: Player lines use a fixed-width points column

Every player line SHALL place its points in a field of fixed width, so that a player line's total
width does not vary with the magnitude of the score or with a no-stats marker. This lets the report
be set beside another one without the second column ragging.

#### Scenario: Scores of different widths occupy the same field

- **WHEN** one player scores 4 and another scores 31.5
- **THEN** both points values occupy a field of the same width on their lines

#### Scenario: A no-stats marker keeps the field width

- **WHEN** a player has no stats
- **THEN** the no-stats marker occupies the same fixed-width field a score would
