## MODIFIED Requirements

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

## ADDED Requirements

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
