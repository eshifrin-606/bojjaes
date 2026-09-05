# roster-source

## Purpose

Define what a roster file is, where it lives, and how it becomes a lineup: an ordered `id,name`
record list read from `<lineups>/<season>/<week>/<team>.csv`, whose first nine records are the
starting lineup.

The roster file is a hand-maintained lineup card, so this capability governs reading it as written
— file order is the lineup order, the id identifies the player and the name is only a label, and a
file that cannot serve as a lineup card is refused rather than silently repaired. Scoring belongs
to `player-week-score`; presenting a scored roster belongs to `roster-score-report`.

## Requirements

### Requirement: A roster file is an ordered list of records

A roster file SHALL be read as lines of `id,name`, one record per line, in file order. The records
SHALL be returned in that order; reading a roster SHALL NOT sort, group, or otherwise rearrange
them, because file order is the lineup card.

A line SHALL be split on its first comma only: the text before it is the id, the text after it is
the name. Surrounding spaces SHALL be trimmed from both fields, so alignment spacing in a
hand-edited file does not become part of an id or a name.

Blank lines and lines whose first non-space character is `#` SHALL NOT be records. A final line
with no trailing newline SHALL still yield its record.

#### Scenario: Records are returned in file order

- **WHEN** a roster file holds three records
- **THEN** reading it yields those three records in the order they appear in the file

#### Scenario: Comments and blank lines are not records

- **WHEN** a roster file opens with a `#` comment line and holds a blank line between two records
- **THEN** reading it yields only the records, and the comment and blank line occupy no position

#### Scenario: An indented comment is still a comment

- **WHEN** a line's first non-space character is `#`
- **THEN** that line is not a record

#### Scenario: Surrounding spaces are not part of a field

- **WHEN** a record is written as `4984 , Josh Allen`
- **THEN** its id is `4984` and its name is `Josh Allen`

#### Scenario: A name containing a comma survives

- **WHEN** a record is written as `1234,Smith, Jr.`
- **THEN** its id is `1234` and its name is `Smith, Jr.`

#### Scenario: A file with no final newline yields its last record

- **WHEN** a roster file's last line has no trailing newline
- **THEN** that line is read as a record

### Requirement: The id is authoritative and the name is a label

The id SHALL be the only field used to identify a player to anything outside the roster. The name
SHALL be carried as display text only and SHALL NOT be used to look up, match, or validate a
player.

A record MAY have an empty name. Reading SHALL NOT reject it, invent a placeholder, or fall back to
the id, so that a roster written without labels stays readable as written.

#### Scenario: A record with no name is read

- **WHEN** a record is written as `4984,` or as `4984`
- **THEN** the record is read with id `4984` and an empty name

#### Scenario: The name is not checked against anything

- **WHEN** a record pairs a valid id with a name that belongs to a different player
- **THEN** reading succeeds and the record carries the name as written

### Requirement: A line that is neither a comment nor a record is an error

A non-blank, non-comment line that yields no id SHALL make reading the roster fail with an error
naming the file and the line number. Reading SHALL NOT skip such a line.

A silently skipped line is worse than a refused file: every record after it moves up one position,
which changes which players are starters without changing anything visible in the report.

#### Scenario: A line with no id is refused

- **WHEN** a roster file holds the line `,Josh Allen`
- **THEN** reading fails with an error naming the file and that line's number

#### Scenario: A line that is only spaces and a comma is refused

- **WHEN** a roster file holds the line `  ,  `
- **THEN** reading fails rather than treating the line as blank

### Requirement: A roster that cannot serve as a lineup card is refused

Reading SHALL fail when a roster file holds no records, and when two records carry the same id.

Duplicate ids SHALL be refused because the same player cannot occupy two lineup slots: his points
would be printed twice and counted twice in the starters' total, with nothing in the report
disclosing it.

#### Scenario: A roster with only comments is refused

- **WHEN** a roster file holds comment lines and blank lines but no records
- **THEN** reading fails with an error naming the file

#### Scenario: A repeated id is refused

- **WHEN** a roster file holds the same id on two records
- **THEN** reading fails with an error naming the repeated id

#### Scenario: A repeated name is not an error

- **WHEN** two records carry different ids and the same name
- **THEN** reading succeeds, because the name identifies nothing

### Requirement: Roster files are located by season, week, and team

A roster SHALL be located within a lineup tree at `<root>/<season>/<week>/<team>.csv`, where the
root is supplied by the caller rather than assumed. Callers SHALL NOT construct this path
themselves; the tree layout SHALL be stated in exactly one place.

The season and week used to locate a roster SHALL be the same season and week the roster is scored
for, so neither a default nor a team shorthand can resolve onto another week.

A team name SHALL be a single path segment. A name containing a path separator or a parent
reference SHALL be refused rather than resolved, so a team argument cannot reach outside the
lineup tree.

#### Scenario: A team name resolves within its week

- **WHEN** a roster is requested for season 2025, week 14, team `wood`, under root `scripts/lineups`
- **THEN** the resolved path is `scripts/lineups/2025/14/wood.csv`

#### Scenario: A missing roster names the path it looked for

- **WHEN** the resolved path does not exist
- **THEN** the error names that path

#### Scenario: A team name cannot escape the tree

- **WHEN** a team name contains `/` or `..`
- **THEN** resolution fails and no file is read

### Requirement: The first nine records are the starting lineup

A roster SHALL be split positionally: its first nine records are the starting lineup and every
record after the ninth is bench. Both groups SHALL preserve file order.

The split SHALL be positional only. A roster file carries no position column, so the split SHALL
NOT validate that the starters form a legal lineup and SHALL NOT reorder records to produce one.
Reordering two lines of the file therefore changes who starts, which is the intended behaviour of a
hand-maintained lineup card.

#### Scenario: A roster longer than a lineup splits at nine

- **WHEN** a roster holds twelve records
- **THEN** the first nine are starters and the last three are bench, each group in file order

#### Scenario: A roster shorter than a lineup is all starters

- **WHEN** a roster holds five records
- **THEN** all five are starters and the bench is empty

#### Scenario: A roster of exactly nine has an empty bench

- **WHEN** a roster holds exactly nine records
- **THEN** all nine are starters and the bench is empty, rather than the bench being absent

#### Scenario: Comments do not consume a starter slot

- **WHEN** a roster file holds comment and blank lines interleaved among nine records
- **THEN** all nine records are starters

#### Scenario: Reordering the file changes who starts

- **WHEN** the tenth record of a roster file is moved above the ninth
- **THEN** it becomes a starter and the record it displaced becomes bench
