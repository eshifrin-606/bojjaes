## MODIFIED Requirements

### Requirement: A week directory holds exactly one matchup

A matchup SHALL be resolvable from a season and a week alone, without a team argument, by reading
the week's directory within the lineup tree. Resolution SHALL yield the team names of that week's
matchup and nothing else: both teams when the directory holds two lineups, or one team and an empty
name for the side to be decided when it holds one. Reading and validating those rosters remains a
separate step.

Only the directory's own entries SHALL be considered, and only those that are regular files whose
name ends in `.csv`. Subdirectories, files with any other extension, and files whose name begins
with a dot SHALL be ignored rather than counted, so an editor backup, a `.DS_Store`, or a save's
temporary file cannot turn a well-formed week into an error. A team name SHALL be the file's name
with `.csv` removed.

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

- **WHEN** a week directory holds `bojjaes.csv`, `wood.csv`, a `.DS_Store`, a `notes.md`, a
  `.wood.csv.tmp`, and a subdirectory
- **THEN** resolution yields `bojjaes` and `wood` rather than failing on the extra entries

### Requirement: A week directory that is not a matchup is refused

Resolution SHALL fail, rather than choose, when the week directory does not hold a matchup. Each
failure SHALL name the directory it read, and SHALL name what it found there.

Three or more roster files SHALL be an error. There is no rule by which two of three files are the
matchup, and a guess would render a full, plausible two-column page for a matchup nobody plays.

A week directory that does not exist, or holds no roster files, SHALL be an error. Two roster files
with no `bojjaes.csv` among them SHALL be an error: resolution answers who the Bojjaes are playing,
and a directory of two other teams does not answer it.

One roster file SHALL NOT be an error: it is a matchup with one side to be decided. When the file is
`bojjaes.csv`, the Bojjaes are returned and the opponent is empty; otherwise that file's team is the
opponent and the Bojjaes' side is empty.

#### Scenario: Three rosters are refused rather than narrowed

- **WHEN** a week directory holds `bojjaes.csv`, `wood.csv`, and `aroma.csv`
- **THEN** resolution fails with an error naming the directory and the three files it found, and no
  pair is chosen

#### Scenario: A lone Bojjaes roster has its opponent to be decided

- **WHEN** a week directory holds only `bojjaes.csv`
- **THEN** resolution yields `bojjaes` and an empty opponent

#### Scenario: A lone opponent roster has the Bojjaes to be decided

- **WHEN** a week directory holds only `wood.csv`
- **THEN** resolution yields an empty Bojjaes side and `wood` as the opponent

#### Scenario: An empty or missing week is refused

- **WHEN** a week directory holds no roster files, or does not exist
- **THEN** resolution fails with an error naming the directory it looked in
