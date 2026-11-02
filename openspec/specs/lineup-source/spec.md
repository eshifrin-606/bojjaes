# lineup-source

## Purpose

Define what a roster file is, where it lives, and how it becomes a lineup: an ordered record list,
under a header naming its `id`, `name`, `position`, and `team` columns, read from `<season>/<week>/<team>.csv` within a caller-supplied lineup tree, whose nine records fill the
lineup's slots by position.

The roster file is a hand-maintained lineup card, so this capability governs reading it as written
— file order only breaks ties between records for a slot, the id identifies the player and the name is only a label, and a
file that cannot serve as a lineup card is refused rather than silently repaired. Scoring belongs
to `player-week-score`; presenting a scored roster belongs to `matchup-page`.
## Requirements
### Requirement: A roster file is an ordered list of records

A roster file SHALL be read as a header row followed by records, one record per line, in file order.
Parsing SHALL yield the records in that order and SHALL NOT sort, group, or otherwise rearrange
them. File order does not decide a record's slot; it only breaks ties when the records are assigned
to slots, per *Records fill the slots by position, not by file order*.

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

The id SHALL be the only field used to identify a player to anything outside the roster. The name
and the position SHALL be carried as display text only, and SHALL NOT be used to look up, match, or
validate a player. The short name derived from the name SHALL be display text on the same terms,
and nothing SHALL identify a player by it.

The team SHALL NOT be used to identify, match, or validate a player either. It MAY be used to find
the NFL game that team plays in the record's week. Reading SHALL NOT check the team against the
player.

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

#### Scenario: The team names a game, not a player

- **WHEN** a record's team is `BUF`
- **THEN** the record's stats are still found by its id alone, and `BUF` is what a caller may use to
  find that week's `BUF` game

### Requirement: A line that is neither a comment nor a record is an error

A non-blank, non-comment line after the header SHALL make reading the roster fail with an error
naming the file and the line number when it yields no id, or when its number of fields differs
from the number of columns in the header. Reading SHALL NOT skip such a line.

A silently skipped line is worse than a refused file: it drops a player from the lineup, or changes
which player fills a FLEX or D slot, without anything visible in the report saying so. A line
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

The server binary SHALL carry git's lineup tree compiled into it, and the server SHALL NOT read any
lineup from the working directory. With no volume configured, the embedded tree SHALL be the only
tree read. With a volume configured, the embedded tree SHALL NOT be read to serve a request; it is
the source boot seeding copies from, per "Git seeds the store at boot, one week at a time".

Carrying the tree in the binary means the server is correct regardless of where it is started from,
which a container image cannot otherwise guarantee.

The tree SHALL be embedded by directory rather than by enumerating seasons or weeks, so adding a
season is adding a directory and never an edit to a directive. A tree that fails to root itself at
startup SHALL stop the process rather than serve a server with no lineups.

#### Scenario: A season directory is embedded without being named

- **WHEN** a new `<season>/<week>/` directory of lineups is added to the tree and the binary is
  rebuilt
- **THEN** that week is served, with no change to any embed directive or list of seasons

#### Scenario: The working directory does not decide what is served

- **WHEN** the server is started, with no volume configured, from a directory that holds no lineup
  files at all
- **THEN** every week in the embedded tree is still served

### Requirement: The server reads the in-flight week from a volume

The server SHALL take the path of a lineup volume from the `LINEUP_VOLUME` environment variable.
When it holds a value, the server's lineup tree SHALL be the directory `<LINEUP_VOLUME>/lineups`
alone, laid out exactly as the embedded tree (`<season>/<week>/<team>.csv`). There SHALL be no
layering and no per-week choice between sources: what that directory holds is what is served.
Nothing outside `lineups/` SHALL be read as a lineup, so the mount root's own contents (such as
`lost+found` or the seed manifest) are never mistaken for a season.

When `LINEUP_VOLUME` is unset or empty, the server SHALL read the embedded tree alone.

The volume SHALL be supplied to the lineup tree as a filesystem and consulted when read, not only at
startup, so path resolution, matchup resolution, week existence, and the latest week are answered
by the same rules as over the embedded tree, and a week that changes on the volume while the server
runs is served without a restart.

#### Scenario: No volume configured

- **WHEN** `LINEUP_VOLUME` is not present in the environment, or is set to the empty string
- **THEN** the server serves the embedded tree alone, and nothing is written anywhere

#### Scenario: A volume is configured

- **WHEN** `LINEUP_VOLUME` holds `/data`
- **THEN** every lineup the server serves is read from `/data/lineups`

#### Scenario: A volume week the embedded tree lacks is served

- **WHEN** the volume holds `2026/9/` and the embedded tree's latest week is 2026/8
- **THEN** season 2026's latest week is 9, read from the volume

#### Scenario: An older volume week is served as the volume holds it

- **WHEN** the volume's `2026/2/` holds a different opponent than the embedded `2026/2/`, and the
  seed manifest records the embedded week's current hash
- **THEN** week 2 is served with the volume's opponent

#### Scenario: A week changed on the volume while running is served

- **WHEN** a roster file in a volume week is changed after the server has started
- **THEN** the next read of that week returns the changed roster

### Requirement: Git seeds the store at boot, one week at a time

When a volume is configured, the server SHALL seed the store from the embedded tree once at startup.
Every embedded `<season>/<week>` directory (numeric season and week names, as `LatestWeek` counts
them) SHALL be considered on its own, by a content hash of git's version of that week.

The hash SHALL be determined by the week's file names and file contents alone — not by modification
times, the build, or the order a directory is listed in — so rebuilding an unchanged tree yields the
same hash.

A manifest at `<LINEUP_VOLUME>/seed-manifest.json`, outside `lineups/`, SHALL record for each seeded
week the hash of git's version at its last seed. For each embedded week:

- when the manifest has no entry for the week, or its entry differs from the week's hash, the
  store's week SHALL be replaced with git's and the manifest entry then set to the week's hash;
- when the entry equals the week's hash, the store's week SHALL NOT be read for replacement,
  altered, or replaced, whatever it holds.

A volume with no `lineups/` directory or no manifest SHALL be seeded as one whose manifest is empty,
creating what it needs.

#### Scenario: A fresh volume receives every embedded week

- **WHEN** the volume is empty and the embedded tree holds 2025/14–16 and 2026/1–8
- **THEN** after seeding, the store holds each of those weeks with git's files, and the manifest
  holds an entry for each

#### Scenario: A week git changed is copied

- **WHEN** the manifest's entry for 2026/8 is the hash of an earlier version of that week, and the
  embedded 2026/8 has since changed
- **THEN** the store's 2026/8 holds the embedded files and the manifest's entry is the new hash

#### Scenario: A week git did not change is left alone

- **WHEN** the store's 2026/7 holds a phone-edited roster and the manifest's entry for 2026/7 equals
  the embedded week's hash
- **THEN** after seeding, the store's 2026/7 is byte-for-byte as before

#### Scenario: A week with no manifest entry is copied

- **WHEN** the store holds 2026/8 but the manifest has no entry for it
- **THEN** the store's 2026/8 is replaced with the embedded week and its entry is recorded

#### Scenario: The hash ignores everything but names and contents

- **WHEN** two trees hold the same week with identical file names and contents, built at different
  times and listed in different orders
- **THEN** their hashes for that week are equal

#### Scenario: Renaming a file changes the hash

- **WHEN** two trees hold a week with the same contents, but one names the opponent `wood.csv` and
  the other `aroma.csv`
- **THEN** their hashes for that week differ

#### Scenario: Non-week entries are not seeded

- **WHEN** the embedded tree holds a non-numeric directory or a stray file at the root or season
  level
- **THEN** seeding neither copies it nor records a manifest entry for it

### Requirement: Seeding replaces a store week whole

When seeding replaces a store week, the store's `<season>/<week>` directory SHALL afterwards hold
exactly git's files for that week, with git's contents: any file the store's week held that git's
does not SHALL be gone. A week SHALL never be merged file by file, since a changed opponent merged
into the old week would leave three rosters and break it.

Seeding SHALL NOT touch any other week while replacing one.

#### Scenario: A changed opponent replaces the old one

- **WHEN** the store's 2026/8 holds `bojjaes.csv` and `wood.csv`, and git's changed 2026/8 holds
  `bojjaes.csv` and `aroma.csv`
- **THEN** after seeding the store's 2026/8 holds only `bojjaes.csv` and `aroma.csv`, and its
  matchup resolves to `bojjaes` against `aroma`

#### Scenario: Other weeks are untouched by a replacement

- **WHEN** seeding replaces 2026/8
- **THEN** every file in the store's other weeks is byte-for-byte as before

### Requirement: A week git deleted stays in the store

Seeding SHALL NOT delete or alter a store week that the embedded tree does not hold, and SHALL NOT
remove that week's manifest entry. A week only the store holds may be a phone-set lineup git has
never seen.

#### Scenario: A store-only week survives seeding

- **WHEN** the store holds 2026/9 and the embedded tree has no 2026/9
- **THEN** after seeding the store's 2026/9 is byte-for-byte as before

#### Scenario: A store-only week survives seeding with no manifest

- **WHEN** the store holds 2026/9, the embedded tree has no 2026/9, and the manifest is absent or
  unparsable
- **THEN** after seeding the store's 2026/9 is byte-for-byte as before

#### Scenario: A week removed from git keeps its manifest entry

- **WHEN** the manifest has an entry for 2026/3 and the embedded tree no longer holds 2026/3
- **THEN** after seeding the entry for 2026/3 is unchanged

### Requirement: Seeding logs a phone edit it overwrites

When seeding replaces a store week that already exists and whose own hash equals neither the
manifest's entry for it nor git's new hash — content git never seeded, such as a phone edit — the
server SHALL log, before replacing it, the week's season and week and every file it held, by name
and full contents, so the overwritten lineup can be recovered from the log.

Replacing a week that is absent from the store, or that still holds exactly what git last seeded,
or that already holds git's new content, SHALL NOT log an overwrite.

#### Scenario: A phone edit overwritten by a git change is logged

- **WHEN** the store's 2026/8 holds a phone-edited `aroma.csv`, and git's 2026/8 changed since its
  last seed
- **THEN** a log records 2026/8 as overwritten, with `aroma.csv`'s name and the phone-edited
  contents, and the store's 2026/8 then holds git's files

#### Scenario: Replacing what git last seeded is not logged as an overwrite

- **WHEN** the store's 2026/8 holds exactly git's previously seeded version, and git's 2026/8 has
  changed
- **THEN** the week is replaced and no overwrite is logged

#### Scenario: Seeding a fresh week is not logged as an overwrite

- **WHEN** the store has no 2026/8 and the embedded tree does
- **THEN** the week is copied and no overwrite is logged

#### Scenario: A store-only week git now adds is logged if it differs

- **WHEN** the store holds a phone-set 2026/9, the manifest has no entry for it, and a new binary's
  embedded tree holds a different 2026/9
- **THEN** the phone-set 2026/9 is logged as overwritten and then replaced with git's

### Requirement: Seeding is idempotent

The manifest entry for a week SHALL be written only after that week's replacement has completed,
and the manifest SHALL be written so that a reader never sees it partially written. Seeding twice
from the same embedded tree SHALL leave the store and manifest as the first seeding left them, and
the second seeding SHALL replace no week.

#### Scenario: A second boot is a no-op

- **WHEN** seeding has completed and the server restarts with the same binary
- **THEN** no week is replaced, nothing is logged as overwritten, and the store and manifest are
  unchanged

#### Scenario: A crash before the manifest write re-copies on the next boot

- **WHEN** a week was replaced but the manifest entry was not written
- **THEN** the next seeding replaces that week again with identical content, logs no overwrite,
  and records the entry

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

### Requirement: A lineup is nine slots in a fixed order

A lineup SHALL be exactly nine slots, in this canonical order: QB, RB, WR, TE, FLEX, FLEX, K, D, D.
Each slot SHALL hold exactly one record, and the starters SHALL be those nine records in slot order.
There SHALL be no bench: every record in a roster file fills a slot.

A slot SHALL accept a record by its position:

- QB accepts `QB`; K accepts `K`.
- RB accepts `RB` or `FB`; WR accepts `WR`; TE accepts `TE`.
- FLEX accepts any position RB, WR, or TE accepts.
- D accepts `DL DE DT NT LB OLB ILB DB CB S SS FS`.

A record's position SHALL remain the player's listed position as written in the file. Filling a
slot SHALL NOT rewrite it, so a fullback in the RB slot is still shown and written as `FB`.

#### Scenario: A file already in slot order keeps its order

- **WHEN** a roster file lists a QB, RB, WR, TE, WR, RB, K, LB, and CB in that order
- **THEN** the starters are those nine records in file order

#### Scenario: A fullback fills the RB slot

- **WHEN** a roster file holds a QB, an `FB`, two WRs, a TE, another WR, a K, and two defenders,
  and no `RB`
- **THEN** the `FB` record fills the RB slot and its position is still `FB`

### Requirement: Records fill the slots by position, not by file order

Reading a roster SHALL assign its records to slots deterministically, whatever order the file lists
them in:

1. QB takes the `QB` record; K takes the `K` record.
2. RB, WR, and TE each take the first record in file order that it accepts.
3. The two FLEX slots take the remaining RB, WR, and TE records, in file order.
4. The two D slots take the defensive records, in file order.

Reordering lines of a roster file SHALL change a slot's occupant only where that order breaks a
tie: which of several same-position records fills the RB, WR, or TE slot and which a FLEX, the
order of the two FLEX slots, and the order of the two D slots.

#### Scenario: A file in another order is read into slot order

- **WHEN** a roster file lists a K, LB, WR, WR, TE, RB, RB, QB, and DE in that order
- **THEN** the starters are the QB, the first RB, the first WR, the TE, the second WR, the second
  RB, the K, the LB, and the DE

#### Scenario: Extra players at one position fill the FLEX slots in file order

- **WHEN** a roster file lists three RBs before its only WR and its only TE
- **THEN** the first RB fills the RB slot, and the second and third fill the two FLEX slots in
  file order

#### Scenario: A FLEX may hold a tight end

- **WHEN** a roster file holds one RB, one WR, and three TEs
- **THEN** the first TE fills the TE slot and the second and third fill the FLEX slots

#### Scenario: Defenders keep file order

- **WHEN** a roster file lists a CB above a DE
- **THEN** the CB fills the first D slot and the DE the second

#### Scenario: Swapping two RB lines swaps the RB and the FLEX

- **WHEN** a roster file's two RBs, filling the RB slot and a FLEX slot, swap lines
- **THEN** each now fills the slot the other filled, and no other slot changes

### Requirement: A roster that cannot fill the slots is refused

Reading SHALL fail with an error naming the file when the roster does not hold exactly nine
records, or when its records cannot fill every slot by the rules in *Records fill the slots by
position, not by file order*. A roster is never partly slotted, and no record is dropped or moved
into a slot that does not accept it.

When the count is wrong the error SHALL state the count. When nine records cannot fill the slots
the error SHALL name a slot left empty.

#### Scenario: A tenth player row is refused

- **WHEN** a roster file holds ten records
- **THEN** reading fails with an error naming the file and stating ten records

#### Scenario: Eight player rows are refused

- **WHEN** a roster file holds eight records
- **THEN** reading fails with an error naming the file and stating eight records

#### Scenario: Comments do not count as players

- **WHEN** a roster file holds nine records and comment and blank lines between them
- **THEN** reading succeeds

#### Scenario: A roster with no tight end is refused

- **WHEN** a roster file holds nine records: a QB, two RBs, three WRs, a K, and two defenders
- **THEN** reading fails with an error naming the TE slot

#### Scenario: A second quarterback is refused

- **WHEN** a roster file holds two QBs and otherwise one each of RB, WR, TE, a fourth skill player,
  K, and two defenders
- **THEN** reading fails, because the second QB fits no slot and a FLEX slot is left empty

#### Scenario: A third defender is refused

- **WHEN** a roster file holds three defenders, one QB, one K, and four RB, WR, and TE records
- **THEN** reading fails with an error naming a slot left empty

### Requirement: A lineup renders back to roster records in slot order

A lineup SHALL render to the text of a roster file: the header `id,name,position,team`, then one
record per slot, in canonical slot order, each carrying its id, name, position, and team as read.
The rendering SHALL NOT include a short name, comments, or blank lines.

Reading the rendered text SHALL yield a lineup equal to the one rendered.

#### Scenario: A lineup read from any order renders in slot order

- **WHEN** a roster file listing its K first and its QB last is read and rendered
- **THEN** the rendered text is the header followed by the QB, RB, WR, TE, FLEX, FLEX, K, D, D
  records, in that order

#### Scenario: Rendering keeps a fullback's position

- **WHEN** a lineup whose RB slot holds an `FB` is rendered
- **THEN** that record's line carries the position `FB`

#### Scenario: A rendered lineup reads back unchanged

- **WHEN** a lineup is rendered and the text is read as a roster file
- **THEN** the starters, in slot order, have the same ids, names, positions, and teams

### Requirement: Every record read from a roster carries a short name

Every record in a lineup read from a roster SHALL carry a short name. The
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

#### Scenario: A record in a late slot carries a short name

- **WHEN** a roster's second defender, filling the last D slot, is named `Fred Warner`
- **THEN** that record's short name is `F Warner`

### Requirement: A short name depends only on its own record's name

A record's short name SHALL be derived from that record's name alone, by the rules in *Every record
read from a roster carries a short name*. It SHALL NOT depend on any other record in the roster or in
any other roster, and it SHALL NOT depend on which slot the record fills.

Two or more records MAY carry the same short name. That SHALL NOT be an error, and neither record
SHALL fall back to its full name or to any longer form.

#### Scenario: Two starters who share an initial and surname keep the short form

- **WHEN** a roster's starters include `Jameson Williams`, `Javonte Williams`, and `Caleb Williams`
- **THEN** their short names are `J Williams`, `J Williams`, and `C Williams`

#### Scenario: A repeated name keeps the short form

- **WHEN** two starters carry different ids and the same name `Josh Allen`
- **THEN** reading succeeds and both records carry `J Allen` as their short name

#### Scenario: Moving a record between slots does not change another's short name

- **WHEN** a roster holds wide receivers `Josh Allen` and `Jaylen Allen`, and their two lines are
  swapped so each fills the other's slot
- **THEN** both records carry `J Allen` as their short name before and after the swap

### Requirement: A week's existence can be asked without resolving it

The lineup tree SHALL answer whether a season and week exist in it. A week exists when the tree
holds a directory at `<season>/<week>`, the same directory that matchup resolution reads.

The answer SHALL depend only on that directory being present. It SHALL NOT open, read, or parse any
roster file, and SHALL NOT decide whether the directory holds a matchup. A week that exists but
would fail resolution (too many, too few, or no Bojjaes roster) SHALL still be reported as
existing: it is a mistake in the tree, not a missing week.

A file at `<season>/<week>`, rather than a directory, SHALL NOT be reported as an existing week.

#### Scenario: A week with a directory exists

- **WHEN** the tree holds `2026/2/bojjaes.csv` and `2026/2/renegades.csv`
- **THEN** season 2026 week 2 exists

#### Scenario: A week with no directory does not exist

- **WHEN** the tree holds `2026/1/` and `2026/2/` and nothing else under 2026
- **THEN** season 2026 week 3 does not exist

#### Scenario: A week that is not a matchup still exists

- **WHEN** the tree's `2026/3/` directory holds `bojjaes.csv`, `wood.csv`, and `aroma.csv`
- **THEN** season 2026 week 3 exists

#### Scenario: A malformed roster does not affect existence

- **WHEN** the tree's `2026/3/` directory holds `bojjaes.csv` and a `wood.csv` that would fail to
  parse
- **THEN** season 2026 week 3 exists and neither file is read

#### Scenario: A file where a week should be is not a week

- **WHEN** the tree holds a regular file named `2026/3`
- **THEN** season 2026 week 3 does not exist

### Requirement: A season's latest existing week can be found without resolving it

The lineup tree SHALL answer, for a season, the greatest week number it holds a directory for. The
answer is the largest `<week>` such that `<season>/<week>` exists per the existence rule above; it
SHALL NOT depend on whether that week's directory is a well-formed matchup.

The answer SHALL depend only on which week directories are present under the season. It SHALL NOT
open, read, or parse any roster file. A gap in a season's weeks SHALL NOT change the answer: the
latest week is the greatest number present, not one more than the previous answer or the count of
directories.

A season that holds no week directories SHALL report that it has no latest week, distinguishably
from reporting week 0 or any other number.

#### Scenario: The latest week is the greatest present

- **WHEN** the tree holds `2026/1/` and `2026/2/` and nothing else under 2026
- **THEN** season 2026's latest week is 2

#### Scenario: A gap does not lower the answer

- **WHEN** the tree holds `2026/1/` and `2026/3/` but not `2026/2/`
- **THEN** season 2026's latest week is 3

#### Scenario: A week that is not a matchup still counts

- **WHEN** the tree's `2026/3/` directory holds `bojjaes.csv`, `wood.csv`, and `aroma.csv`, and no
  higher week exists under 2026
- **THEN** season 2026's latest week is 3

#### Scenario: A season with no weeks has no latest week

- **WHEN** the tree holds no directories under `2027`
- **THEN** season 2027 has no latest week

#### Scenario: A malformed roster does not affect the answer

- **WHEN** the tree's `2026/2/` directory holds a `wood.csv` that would fail to parse, and no higher
  week exists under 2026
- **THEN** season 2026's latest week is 2 and no roster file is read

### Requirement: A slot states which positions it accepts

Each slot SHALL answer whether it accepts a position, by the same rule that fills slots on read:
RB accepts RB and FB; FLEX accepts anything RB, WR, or TE accepts; D accepts every defensive
position; QB, WR, TE, and K accept only themselves. The nine slots SHALL be available in slot
order.

#### Scenario: Slot acceptance

- **WHEN** RB is asked about FB, FLEX about TE, D about CB, and QB about WR
- **THEN** the answers are yes, yes, yes, and no

### Requirement: A team's roster is read by team name

Roster files SHALL be located as `<team>.csv` within a caller-supplied roster directory, and SHALL
be read with the same record rules as a lineup file (header, closed sets, unique IDs), in file
order, but with no limit on the number of records and no slot assignment. The team names SHALL be
listable, in name order, as the `.csv` files in that directory, ignoring dotfiles. A team name that
is not a single plain path segment SHALL be refused.

#### Scenario: Teams are listed

- **WHEN** the directory holds `wood.csv`, `aroma.csv`, `.DS_Store`, and `notes.txt`
- **THEN** the teams are `aroma`, `wood`

#### Scenario: A roster of more than nine

- **WHEN** `wood.csv` holds 25 records
- **THEN** reading wood's roster yields all 25 in file order

#### Scenario: A path as a team

- **WHEN** the roster for `../wood` is read
- **THEN** the read fails

#### Scenario: Every embedded roster reads

- **WHEN** each team listed in the embedded rosters is read
- **THEN** every read succeeds, and the teams are the league's ten

### Requirement: A lineup can be built from chosen slots

A lineup SHALL be buildable from nine records given in slot order, keeping each record in the slot
it was given. It SHALL be refused when a record has no ID, a slot does not accept its record's
position, or two records share an ID; the error SHALL name the slot. Rendered back to a roster file
and read again, it SHALL yield the same starters in the same slots.

#### Scenario: Chosen order is kept

- **WHEN** a lineup is built with an RB in RB and a second RB in the first FLEX
- **THEN** its starters are in the slots given

#### Scenario: Ineligible record

- **WHEN** a WR is given for the QB slot
- **THEN** building fails, naming QB

#### Scenario: Duplicate record

- **WHEN** the same ID is given for RB and FLEX
- **THEN** building fails

#### Scenario: Round trip

- **WHEN** a built lineup is written as a roster file and read back from the tree
- **THEN** the starters equal the ones it was built from

### Requirement: A team's latest lineup at or before a week can be found

The tree SHALL find the newest season and week, at or before a given season and week, that holds a
lineup file for a team, without opening it. Weeks after the bound, in its season or a later one,
SHALL NOT count. When there is none, it SHALL report that the team has no lineup. A team name that
is not a single plain path segment SHALL be refused.

#### Scenario: The bound week itself

- **WHEN** wood has lineups for 2026 weeks 8 and 9, and the bound is 2026 week 9
- **THEN** the answer is 2026 week 9

#### Scenario: A later week is skipped

- **WHEN** wood has lineups for 2026 weeks 7 and 10, and the bound is 2026 week 9
- **THEN** the answer is 2026 week 7

#### Scenario: Back into an earlier season

- **WHEN** wood has a lineup for 2025 week 16 and 2026 week 3, and the bound is 2026 week 2
- **THEN** the answer is 2025 week 16

#### Scenario: Nothing at or before

- **WHEN** wood's only lineup is 2026 week 10 and the bound is 2026 week 9
- **THEN** it reports no lineup
