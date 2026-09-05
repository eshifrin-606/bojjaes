## ADDED Requirements

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
