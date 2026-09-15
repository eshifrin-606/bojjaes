# lineup-source

## Purpose

Define what a roster file is, where it lives, and how it becomes a lineup: an ordered record list,
under a header naming its `id`, `name`, `position`, and `team` columns, read from `<season>/<week>/<team>.csv` within a caller-supplied lineup tree, whose first nine records are the
starting lineup.

The roster file is a hand-maintained lineup card, so this capability governs reading it as written
— file order is the lineup order, the id identifies the player and the name is only a label, and a
file that cannot serve as a lineup card is refused rather than silently repaired. Scoring belongs
to `player-week-score`; presenting a scored roster belongs to `matchup-page`.
## Requirements
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

A lineup SHALL be located within a lineup tree at `<season>/<week>/<team>.csv`, relative to a
filesystem supplied by the caller rather than assumed. The filesystem SHALL be supplied as an
`io/fs` filesystem already rooted at the tree, so the tree's own location is the caller's fact and
not this capability's. Callers SHALL NOT construct this path themselves; the tree layout SHALL be
stated in exactly one place.

Resolution SHALL yield a slash-separated name valid within that filesystem — `2025/14/wood.csv` —
and SHALL NOT yield an operating-system path, a leading separator, or a parent reference. Reading a
lineup SHALL go through the supplied filesystem and SHALL NOT reach the operating system directly,
so what the caller supplies is the only tree that can be read.

The season and week used to locate a lineup SHALL be the same season and week the lineup is scored
for, so neither a default nor a team shorthand can resolve onto another week.

A team name SHALL be a single path segment. A name containing a path separator or a parent
reference SHALL be refused rather than resolved, so a team argument cannot reach outside the
lineup tree.

#### Scenario: A team name resolves within its week

- **WHEN** a lineup is requested for season 2025, week 14, team `wood`
- **THEN** the resolved name is `2025/14/wood.csv`, relative to the supplied filesystem

#### Scenario: A missing lineup names the path it looked for

- **WHEN** the resolved name does not exist in the supplied filesystem
- **THEN** the error names that name

#### Scenario: A team name cannot escape the tree

- **WHEN** a team name contains `/` or `..`
- **THEN** resolution fails and no file is read

#### Scenario: Two filesystems, the same season and week, different lineups

- **WHEN** the same season, week, and team are resolved against two different supplied filesystems
- **THEN** each read yields that filesystem's own lineup, and neither reads the other's

### Requirement: The deployed binary carries its lineup tree

The server SHALL read lineups from a tree compiled into its own binary, and SHALL NOT read any
lineup from the working directory or from any other path on the host.

A deploy is therefore also the lineup update: the tree ships with the binary that serves it, which
is what we want while lineups remain hand-edited files in git. It also means the binary is correct
regardless of where it is started from, which a container image cannot otherwise guarantee.

The tree SHALL be embedded by directory rather than by enumerating seasons or weeks, so adding a
season is adding a directory and never an edit to a directive. A tree that fails to root itself at
startup SHALL stop the process rather than serve a server with no lineups.

#### Scenario: A season directory is embedded without being named

- **WHEN** a new `<season>/<week>/` directory of lineups is added to the tree and the binary is
  rebuilt
- **THEN** that week is served, with no change to any embed directive or list of seasons

#### Scenario: The working directory does not decide what is served

- **WHEN** the server is started from a directory that holds no lineup files at all
- **THEN** every week in the embedded tree is still served

### Requirement: A week directory holds exactly one matchup

A matchup SHALL be resolvable from a season and a week alone, without a team argument, by reading
the week's directory within the lineup tree. Resolution SHALL yield the two team names of that
week's matchup and nothing else; reading and validating those rosters remains a separate step.

Only the directory's own entries SHALL be considered, and only those that are regular files whose
name ends in `.csv`. Subdirectories, files with any other extension, and files whose name begins
with a dot SHALL be ignored rather than counted, so an editor backup or a `.DS_Store` cannot turn a
well-formed week into an error. A team name SHALL be the file's name with `.csv` removed.

Resolution SHALL NOT open, read, or parse either roster file. A matchup is a fact about the
directory; whether a roster inside it is usable is a fact about the file, reported when it is read.

#### Scenario: A two-roster week resolves to its two teams

- **WHEN** a week directory holds `bojjaes.csv` and `wood.csv`
- **THEN** resolving that season and week yields the team names `bojjaes` and `wood`

#### Scenario: A malformed roster does not prevent resolution

- **WHEN** a week directory holds `bojjaes.csv` and a `wood.csv` that would fail to parse
- **THEN** resolution succeeds and yields both team names, and the parse failure is reported only
  when that roster is read

#### Scenario: Files that are not rosters are not counted

- **WHEN** a week directory holds `bojjaes.csv`, `wood.csv`, a `.DS_Store`, a `notes.md`, and a
  subdirectory
- **THEN** resolution yields `bojjaes` and `wood` rather than failing on the extra entries

### Requirement: The Bojjaes' opponent is the roster that is not theirs

Of a week's two rosters, `bojjaes.csv` SHALL identify the Bojjaes and the other file SHALL identify
their opponent. Resolution SHALL return the Bojjaes first and the opponent second, so a caller that
uses the order gets the Bojjaes on the left without naming them.

The opponent SHALL be determined by elimination — it is the file that is not `bojjaes.csv` — and
SHALL NOT be inferred from any other property of the file, such as its name, its size, or its
position in the directory listing.

#### Scenario: The opponent is the other file

- **WHEN** a week directory holds `bojjaes.csv` and `wood.csv`
- **THEN** the Bojjaes are returned first and `wood` second

#### Scenario: Directory order does not decide the order

- **WHEN** a week directory holds `bojjaes.csv` and an opponent whose name sorts before it, such as
  `aroma.csv`
- **THEN** the Bojjaes are still returned first and `aroma` second

### Requirement: A week directory that is not a matchup is refused

Resolution SHALL fail, rather than choose, when the week directory does not hold exactly one
matchup. Each failure SHALL name the directory it read, and SHALL name what it found there.

Three or more roster files SHALL be an error. There is no rule by which two of three files are the
matchup, and a guess would render a full, plausible two-column page for a matchup nobody plays.

Fewer than two roster files SHALL be an error, as SHALL a week directory that does not exist. Two
roster files with no `bojjaes.csv` among them SHALL be an error: resolution answers who the Bojjaes
are playing, and a directory of two other teams does not answer it.

#### Scenario: Three rosters are refused rather than narrowed

- **WHEN** a week directory holds `bojjaes.csv`, `wood.csv`, and `aroma.csv`
- **THEN** resolution fails with an error naming the directory and the three files it found, and no
  pair is chosen

#### Scenario: A lone roster is refused

- **WHEN** a week directory holds only `bojjaes.csv`
- **THEN** resolution fails rather than returning a matchup with one side missing

#### Scenario: An empty or missing week is refused

- **WHEN** a week directory holds no roster files, or does not exist
- **THEN** resolution fails with an error naming the directory it looked in

#### Scenario: Two teams that are not ours are refused

- **WHEN** a week directory holds `aroma.csv` and `fuego.csv`
- **THEN** resolution fails, because neither file is the Bojjaes'

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

### Requirement: Every record read from a roster carries a short name

Every record in a lineup read from a roster SHALL carry a short name, starters and bench alike. The
short name SHALL be derived from the record's name when the roster is read. It SHALL NOT be read
from the file, and the roster file SHALL NOT carry a column for it. The short name SHALL be
non-empty whenever the name is non-empty.

The name SHALL be taken as words separated by whitespace. The short name SHALL be the first word's
lead, a single space, and the remaining words joined by single spaces. The lead SHALL be the first
character of the first word, with no period after it, unless the first word is written as initials,
in which case the lead SHALL be the first word as written.

The first word SHALL count as written as initials when it contains a period, or when it is exactly
two letters and both are uppercase. No other first word SHALL count as initials. A first word with
lowercase letters, or of three or more letters with no period, or joined by a hyphen or apostrophe,
SHALL contribute only its first character.

When the name has three or more words and its last word is a generational suffix (`Jr.`, `Jr`,
`Sr.`, `Sr`, `II`, `III`, `IV`, or `V`), that word SHALL be left out of the short name, whether or
not the first word is initials. The name itself SHALL keep the suffix.

A name of one word SHALL be its own short name.

A short name is a narrower label, not a normalized one. Deriving it SHALL NOT change the record's
name, and SHALL NOT fail a read for any name the roster accepts.

#### Scenario: A two-word name becomes an initial and a surname

- **WHEN** a record's name is `Josh Allen`
- **THEN** its short name is `J Allen`

#### Scenario: A hyphenated first name contributes one letter

- **WHEN** a record's name is `Amon-Ra St. Brown`
- **THEN** its short name is `A St. Brown`

#### Scenario: A first name written with periods is kept as written

- **WHEN** a record's name is `A.J. Brown`
- **THEN** its short name is `A.J. Brown`

#### Scenario: A first name of two capital letters is kept as written

- **WHEN** a record's name is `KC Concepcion`, or `DJ Moore`
- **THEN** its short name is `KC Concepcion`, or `DJ Moore`

#### Scenario: A short or mixed-case first name is not initials

- **WHEN** a record's name is `Al Smith`, or `CeeDee Lamb`
- **THEN** its short name is `A Smith`, or `C Lamb`

#### Scenario: Three capitals with no period are not initials

- **WHEN** a record's name is `JOE Smith`
- **THEN** its short name is `J Smith`

#### Scenario: A generational suffix is dropped from the short name only

- **WHEN** a record's name is `Will McDonald IV`
- **THEN** its short name is `W McDonald` and its name is still `Will McDonald IV`

#### Scenario: A suffix written with or without a period is dropped

- **WHEN** a record's name is `Marvin Harrison Jr.`, or `Marvin Harrison Jr`
- **THEN** its short name is `M Harrison`

#### Scenario: A suffix is dropped after an initials first name

- **WHEN** a record's name is `A.J. Brown Jr.`
- **THEN** its short name is `A.J. Brown`

#### Scenario: A two-word name keeps a word that looks like a suffix

- **WHEN** a record's name is `Tony V`
- **THEN** its short name is `T V`

#### Scenario: A one-word name is unchanged

- **WHEN** a record's name is `Mahomes`
- **THEN** its short name is `Mahomes`

#### Scenario: Bench records carry a short name

- **WHEN** a roster holds ten records and the tenth is named `Dak Prescott`
- **THEN** the bench record's short name is `D Prescott`

### Requirement: A short name depends only on its own record's name

A record's short name SHALL be derived from that record's name alone, by the rules in *Every record
read from a roster carries a short name*. It SHALL NOT depend on any other record in the roster or in
any other roster, and it SHALL NOT depend on whether the record is a starter or on the bench.

Two or more records MAY carry the same short name. That SHALL NOT be an error, and neither record
SHALL fall back to its full name or to any longer form.

#### Scenario: Two starters who share an initial and surname keep the short form

- **WHEN** a roster's starters include `Jameson Williams`, `Javonte Williams`, and `Caleb Williams`
- **THEN** their short names are `J Williams`, `J Williams`, and `C Williams`

#### Scenario: A repeated name keeps the short form

- **WHEN** two starters carry different ids and the same name `Josh Allen`
- **THEN** reading succeeds and both records carry `J Allen` as their short name

#### Scenario: Moving a record across the starter split does not change another's short name

- **WHEN** a roster's second record is `Josh Allen`, and `Jaylen Allen` is moved from the tenth
  record to the ninth
- **THEN** both records carry `J Allen` as their short name before and after the move
