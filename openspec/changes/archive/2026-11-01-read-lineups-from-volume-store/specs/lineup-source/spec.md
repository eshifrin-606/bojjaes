## MODIFIED Requirements

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

## ADDED Requirements

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

## REMOVED Requirements

### Requirement: The volume's week is served when it is at or after the embedded latest week

**Reason**: ADR 0005 (amended 2026-11-01) decision 2 makes the volume the sole read source. There is
no per-week choice between layers; which version of a week is held is decided by the writers (boot
seeding here, the write handler later).
**Migration**: None for readers. A week formerly served from the embedded tree is served from the
volume after seeding copies it there; a volume week formerly served is served unchanged.

### Requirement: A volume week that is not served is ignored, not deleted

**Reason**: With one read source every volume week is served; "stale" and "superseded" volume weeks
no longer exist. Never deleting a week git lacks is kept, as "A week git deleted stays in the
store".
**Migration**: A volume week formerly ignored as stale is now served as the volume holds it, or
replaced by seeding (and logged) if git's copy of that week has no manifest entry yet.

### Requirement: An absent or empty volume reads as the embedded tree

**Reason**: Seeding runs before the server listens and fills an absent or empty `lineups/` from the
embedded tree, so the server never reads an empty store.
**Migration**: Covered by "Git seeds the store at boot, one week at a time" (a fresh volume receives
every embedded week).
