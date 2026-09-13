## MODIFIED Requirements

### Requirement: A roster file is an ordered list of records

A roster file SHALL be read as a header row followed by records, one record per line, in file order.
The records SHALL be returned in that order; reading a roster SHALL NOT sort, group, or otherwise
rearrange them, because file order is the lineup card.

A line SHALL be split on every comma, and each field SHALL be taken by the column its position in
the header names. Surrounding spaces SHALL be trimmed from every field and every header name, so
alignment spacing in a hand-edited file does not become part of a value.

Blank lines and lines whose first non-space character is `#` SHALL NOT be records and SHALL NOT be
the header. A final line with no trailing newline SHALL still yield its record.

#### Scenario: Records are returned in file order

- **WHEN** a roster file holds a header and three records
- **THEN** reading it yields those three records in the order they appear in the file

#### Scenario: Comments and blank lines are not records

- **WHEN** a roster file opens with a `#` comment line, then the header, and holds a blank line
  between two records
- **THEN** reading it yields only the records, and the comment and blank line occupy no position

#### Scenario: An indented comment is still a comment

- **WHEN** a line's first non-space character is `#`
- **THEN** that line is neither the header nor a record

#### Scenario: Surrounding spaces are not part of a field

- **WHEN** under the header `id, name, position, team` a record is written as
  `4984 , Josh Allen , QB , BUF`
- **THEN** its id is `4984`, its name `Josh Allen`, its position `QB`, and its team `BUF`

#### Scenario: A file with no final newline yields its last record

- **WHEN** a roster file's last line has no trailing newline
- **THEN** that line is read as a record

### Requirement: The id is authoritative and the name is a label

The id SHALL be the only field used to identify a player to anything outside the roster. The name,
the position, and the team SHALL be carried as display text only, and SHALL NOT be used to look up,
match, or validate a player. The short name derived from the name SHALL be display text on the
same terms, and nothing SHALL identify a player by it.

A record MAY have an empty name. Reading SHALL NOT reject it, invent a placeholder, or fall back to
the id, so that a roster written without labels stays readable as written. A record with an empty
name SHALL have an empty short name.

#### Scenario: A record with no name is read

- **WHEN** a record is written as `4984,,QB,BUF`
- **THEN** the record is read with id `4984`, an empty name, and an empty short name

#### Scenario: The labels are not checked against the player

- **WHEN** a record pairs a valid id with a name, position, and team that belong to a different
  player, and the position and team are each in their closed sets
- **THEN** reading succeeds and the record carries the labels as written

### Requirement: A line that is neither a comment nor a record is an error

A non-blank, non-comment line after the header SHALL make reading the roster fail with an error
naming the file and the line number when it yields no id, or when its number of fields differs
from the number of columns in the header. Reading SHALL NOT skip such a line.

A silently skipped line is worse than a refused file: every record after it moves up one position,
which changes which players are starters without changing anything visible in the report. A line
with the wrong number of fields is refused rather than padded or truncated, because a missing or
extra comma moves every later value into the wrong column.

#### Scenario: A line with no id is refused

- **WHEN** a roster file holds the line `,Josh Allen,QB,BUF`
- **THEN** reading fails with an error naming the file and that line's number

#### Scenario: A line that is only spaces and commas is refused

- **WHEN** a roster file holds the line `  , , ,  `
- **THEN** reading fails rather than treating the line as blank

#### Scenario: A line with too few fields is refused

- **WHEN** under a four-column header a roster file holds the line `4984,Josh Allen`
- **THEN** reading fails with an error naming the file and that line's number

#### Scenario: A name containing a comma is refused

- **WHEN** under a four-column header a roster file holds the line `1234,Smith, Jr.,WR,BUF`
- **THEN** reading fails, because the line has five fields

### Requirement: The first nine records are the starting lineup

A roster SHALL be split positionally: its first nine records are the starting lineup and every
record after the ninth is bench. Both groups SHALL preserve file order.

The split SHALL be positional only. A record's position is the player's listed position, not the
lineup slot he fills, so the split SHALL NOT use it, SHALL NOT validate that the starters form a
legal lineup, and SHALL NOT reorder records to produce one. Reordering two lines of the file
therefore changes who starts, which is the intended behaviour of a hand-maintained lineup card.

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

#### Scenario: Position does not decide who starts

- **WHEN** a roster's first nine records are all quarterbacks and its tenth is a kicker
- **THEN** the nine quarterbacks are starters and the kicker is bench

## ADDED Requirements

### Requirement: A roster file opens with a header naming its columns

The first line of a roster file that is neither blank nor a comment SHALL be its header. The header
SHALL name exactly the columns `id`, `name`, `position`, and `team`, each once, in any order.
Column names SHALL be matched exactly, including case.

Reading SHALL fail with an error naming the file when the header is missing a column, names a
column twice, or names a column that is not one of the four. An unknown column SHALL be refused
rather than ignored, so a misspelled column name cannot leave every record without that field.

Because fields are taken by name, the order of columns in the header SHALL NOT change what a record
reads as.

#### Scenario: Columns are read by name, not order

- **WHEN** a roster file's header is `team,position,name,id` and a record is `BUF,QB,Josh Allen,4984`
- **THEN** the record's id is `4984`, its name `Josh Allen`, its position `QB`, and its team `BUF`

#### Scenario: A header missing a column is refused

- **WHEN** a roster file's header is `id,name,position`
- **THEN** reading fails with an error naming the file and the missing column `team`

#### Scenario: An unknown column is refused

- **WHEN** a roster file's header is `id,name,postion,team`
- **THEN** reading fails with an error naming the file and the column `postion`

#### Scenario: A repeated column is refused

- **WHEN** a roster file's header is `id,name,position,team,team`
- **THEN** reading fails with an error naming the file and the column `team`

#### Scenario: A file in the old headerless format is refused

- **WHEN** a roster file's first non-comment line is `4984,Josh Allen`
- **THEN** reading fails, because that line is read as a header naming unknown columns

#### Scenario: A header with no records is refused

- **WHEN** a roster file holds a header and comment lines but no records
- **THEN** reading fails with an error naming the file

### Requirement: A position is one of the league's scoring positions as Sleeper names them

A record's position SHALL be one of `QB`, `RB`, `FB`, `WR`, `TE`, `K`, `DL`, `DE`, `DT`, `NT`, `LB`,
`OLB`, `ILB`, `DB`, `CB`, `S`, `SS`, or `FS`, matched exactly, including case. Any other value,
including an empty one, SHALL make reading fail with an error naming the file, the line number, and
the value.

A position SHALL be written as Sleeper's `position` field has it for that player, not converted to a
league or fantasy grouping. Sleeper is inconsistent across defenders — one lineman is `DL` and
another `DE` — and the file carries that inconsistency rather than resolving it.

#### Scenario: A Sleeper defensive position is accepted as written

- **WHEN** one record's position is `DE` and another's is `DL`
- **THEN** reading succeeds and each record carries its position as written

#### Scenario: A misspelled position is refused

- **WHEN** a record's position is `W R`, `wr`, or empty
- **THEN** reading fails with an error naming the file, the line number, and the value

#### Scenario: A Sleeper position that cannot score is refused

- **WHEN** a record's position is `OL` or `P`
- **THEN** reading fails, because no player at that position scores in this league

### Requirement: A team is an NFL team code or FA

A record's team SHALL be one of `ARI`, `ATL`, `BAL`, `BUF`, `CAR`, `CHI`, `CIN`, `CLE`, `DAL`,
`DEN`, `DET`, `GB`, `HOU`, `IND`, `JAX`, `KC`, `LAC`, `LAR`, `LV`, `MIA`, `MIN`, `NE`, `NO`, `NYG`,
`NYJ`, `PHI`, `PIT`, `SEA`, `SF`, `TB`, `TEN`, `WAS`, or `FA`, matched exactly, including case. Any
other value, including an empty one, SHALL make reading fail with an error naming the file, the line
number, and the value.

`FA` SHALL mean the player had no NFL team when the file was written. A team SHALL be the player's
team at the time the file was written; reading SHALL NOT update it.

#### Scenario: A free agent is accepted

- **WHEN** a record's team is `FA`
- **THEN** reading succeeds and the record's team is `FA`

#### Scenario: A misspelled or retired team code is refused

- **WHEN** a record's team is `BUFF`, `buf`, `OAK`, or empty
- **THEN** reading fails with an error naming the file, the line number, and the value
