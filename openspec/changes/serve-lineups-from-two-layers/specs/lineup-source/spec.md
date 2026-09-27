## MODIFIED Requirements

### Requirement: The deployed binary carries its lineup tree

The server SHALL read lineups from a tree compiled into its own binary, and SHALL NOT read any
lineup from the working directory. The only other place the server MAY read lineups from is the
volume directory named by its configuration, per "The server reads the in-flight week from a
volume"; with no volume configured, the embedded tree SHALL be the only tree read.

The embedded tree remains the archive of finalized weeks and a read source for every week the
volume does not serve. It also means the binary is correct regardless of where it is started from,
which a container image cannot otherwise guarantee.

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

## ADDED Requirements

### Requirement: The server reads the in-flight week from a volume

The server SHALL take the path of a lineup volume from the `LINEUP_VOLUME` environment variable.
When it holds a value, the server's lineup tree SHALL be the embedded tree layered with the
directory `<LINEUP_VOLUME>/lineups`, laid out exactly as the embedded tree
(`<season>/<week>/<team>.csv`). Nothing outside that `lineups/` subdirectory SHALL be read, so the
mount root's own contents (such as `lost+found`) are never mistaken for lineups.

When `LINEUP_VOLUME` is unset or empty, the server SHALL read the embedded tree alone.

The composition SHALL be supplied to the lineup tree as a filesystem, so path resolution, matchup
resolution, week existence, and the latest week are answered by the same rules over the composed
tree as over the embedded one.

#### Scenario: No volume configured

- **WHEN** `LINEUP_VOLUME` is not present in the environment, or is set to the empty string
- **THEN** the server serves the embedded tree alone

#### Scenario: A volume is configured

- **WHEN** `LINEUP_VOLUME` holds `/data`
- **THEN** the server layers `/data/lineups` over the embedded tree

### Requirement: The volume's week is served when it is at or after the embedded latest week

The volume SHALL be read as holding at most one week. Weeks SHALL be ordered by season, then by
week within a season, so the first week of a season is after every week of the season before it.
The embedded latest week is the greatest week, in that order, the embedded tree holds a directory
for.

When the volume's week is at or after the embedded latest week, that week SHALL be served from the
volume, and every other week from the embedded tree. The latest week of any season SHALL therefore
be the greater of the two layers' answers.

A week served from the volume SHALL be served from the volume whole: its week directory's listing
and every file within it come from the volume, and no file from the embedded copy of the same week
SHALL be visible. A volume edit of an archived week can thereby change the opponent without the
week holding three rosters.

The volume's contents SHALL be consulted when read, not only at startup, so a week that appears on
the volume while the server runs is served without a restart.

#### Scenario: A later volume week becomes the latest week

- **WHEN** the embedded tree's latest week is 2026/3 and the volume holds `2026/4/`
- **THEN** season 2026's latest week is 4, and week 4's matchup and rosters are read from the volume

#### Scenario: A volume edit of the latest archived week shadows it

- **WHEN** the embedded tree's latest week is 2026/3, holding `bojjaes.csv` and `wood.csv`, and the
  volume holds `2026/3/` with `bojjaes.csv` and `aroma.csv`
- **THEN** week 3 resolves to `bojjaes` against `aroma`, both rosters are read from the volume, and
  the embedded `wood.csv` is not visible

#### Scenario: The volume's week can open a new season

- **WHEN** the embedded tree's latest week is 2025/16 and the volume holds `2026/1/`
- **THEN** season 2026's latest week is 1, served from the volume, and every 2025 week is still
  served from the embedded tree

#### Scenario: Weeks the volume does not hold are served from the embedded tree

- **WHEN** the volume holds `2026/4/` and a request is for 2026/2
- **THEN** week 2 is read from the embedded tree

### Requirement: A volume week older than the embedded latest week is ignored, not deleted

When the volume holds a week before the embedded latest week, the tree SHALL read as the embedded
tree alone: that week SHALL NOT shadow the embedded copy of it, nor be served in its absence. The
stale week SHALL be logged, naming its season and week, and SHALL NOT be deleted or altered, since
it may hold edits that never reached git.

#### Scenario: A stale volume week is not served

- **WHEN** the embedded tree's latest week is 2026/3 and the volume holds `2026/2/` with a
  different opponent than the embedded `2026/2/`
- **THEN** week 2 is read from the embedded tree, and the volume's `2026/2/` is untouched

#### Scenario: A stale volume week is logged

- **WHEN** the volume holds a week before the embedded latest week at startup
- **THEN** a log line names that season and week as ignored

### Requirement: An absent or empty volume reads as the embedded tree

When the configured `lineups/` directory does not exist, or holds no week directory, the composed
tree SHALL answer every question — lineup reads, matchup resolution, week existence, latest week —
exactly as the embedded tree alone does. Nothing SHALL be written to the volume to make this so.

#### Scenario: The lineups directory does not exist

- **WHEN** `LINEUP_VOLUME` names a directory with no `lineups/` subdirectory
- **THEN** every week in the embedded tree is served as without a volume, and no directory is
  created

#### Scenario: The lineups directory is empty

- **WHEN** `<LINEUP_VOLUME>/lineups` exists and holds nothing
- **THEN** every week in the embedded tree is served as without a volume
