## Why

The binary carries its templates and CSS but not its lineups. `cmd/server/main.go` still points
`roster.New` at `scripts/lineups`, a path relative to the working directory, and the package reads
it with `os.Open` and `os.ReadDir`. That works when the server is started from a checkout and stops
working the moment it is a scratch-image container, which is the next thing on the backlog.

[ADR 0004](../../../docs/adr/0004-web-frontend-stack.md) decision 3 already settled the answer —
templates, CSS, and the lineup tree are pulled into the binary with `//go:embed`. The directive
will not cross `..` and will not follow symlinks, so the tree has to move inside a package before
it can be named.

## What Changes

- The lineup tree moves from `scripts/lineups/**` to `internal/lineup/data/**`, keeping its
  `<season>/<week>/<team>.csv` shape. `internal/lineup` embeds it with `//go:embed data` and
  exposes it as an `fs.FS` rooted at the tree, so a new season is a new directory and never an
  edit to a directive.
- **BREAKING (internal API):** the tree is constructed from an `fs.FS` instead of a root directory
  string, and reads through `io/fs` rather than `os`. Resolving a lineup file therefore yields a
  slash-separated name relative to the tree root — `2025/14/wood.csv`, not
  `scripts/lineups/2025/14/wood.csv`.
- `cmd/server/main.go` drops the `lineupRoot` constant and hands the tree the embedded filesystem.
  The server no longer reads any file off the working directory.
- `scripts/scores.sh` and `scripts/fantasycast.sh` keep working against the moved tree by path.
  Their promise in ADR 0004 decision 2 is that the interim UI survives the web page.
- Naming is settled in the same pass, on the reading that these files are weekly lineup cards
  rather than a team's standing roster: a **lineup** is the file, its first nine records are the
  **starters**, and the rest is the **bench**. `internal/roster` becomes `internal/lineup`, the
  `Roster` type becomes `Lineup`, `lineupSize` becomes the starter count, and "roster" leaves the
  specs, the docs, and the scripts. The starters/bench split itself is unchanged.

Out of scope, each its own backlog line: `PORT`, timeouts and graceful shutdown in `main`; the
Dockerfile and `fly.toml` this unblocks; what `/` and an unknown week render; and growing the CSV
to carry position and team.

## Capabilities

### New Capabilities

None. Embedding is where the bytes come from, not a new thing the system does.

### Modified Capabilities

- `roster-source`: the requirement "Roster files are located by season, week, and team" changes
  what a root is. Today it is a caller-supplied directory and resolution yields a filesystem path;
  after this change it is a caller-supplied `fs.FS` and resolution yields a slash-separated name
  within it. The capability also gains a requirement that the deployed binary carries its own
  lineup tree, so no lineup is read from the working directory. Record order, the id/name rule, the
  refusals, the matchup rules, and the starters/bench split are unchanged in behaviour; the
  vocabulary throughout moves from "roster" to "lineup".

The capability's own id stays `roster-source` for this change, because a delta has to archive into
the folder it came from. Renaming the folders — `roster-source` → `lineup-source`,
`roster-score-report` → `lineup-score-report` — is a `git mv` of the spec store, listed as the last
task and done after this change is archived.

## Impact

- **New:** the embed directive and its `fs.Sub` in `internal/lineup`, and `internal/lineup/data/**`
  — the eight CSVs moved from `scripts/lineups/**` by `git mv`, contents untouched.
- **Renamed:** `internal/roster` → `internal/lineup`, `Roster` → `Lineup`, and the two matchup
  errors that say "rosters".
- `path.go`: the tree holds an `fs.FS`; `filepath` becomes `path`. The team-name guard stays and
  gains what `io/fs` requires of a name.
- `read.go`, `matchup.go`: `os.Open` → `fs.FS.Open`, `os.ReadDir` → `fs.ReadDir`. `ErrNoWeek` and
  its siblings keep their meanings.
- The package's tests: trees are built from `fstest.MapFS` rather than named by directory string,
  and the path assertions lose the `scripts/lineups` prefix.
- `internal/web/matchup.go` and its test: the renamed package and type; the fixture week becomes an
  `fstest.MapFS`.
- `cmd/server/main.go`: one import, one constant deleted.
- `scripts/scores.sh`, `scripts/fantasycast.sh`: the path they build for a lineup file.
- `README.md`, `docs/architecture.md`, `docs/package-dependencies.md`, ADR 0004, `backlog.md`: the
  tree's path and the roster/lineup vocabulary.
- No change to the URL, the rendered page, `POST /scores`, the CSV format, the Sleeper client, or
  the cache.
