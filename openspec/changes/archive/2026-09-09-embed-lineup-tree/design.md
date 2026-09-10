## Context

`internal/roster` reads the lineup tree through `os`: `Tree` holds a directory string, `Read` calls
`os.Open`, `Matchup` calls `os.ReadDir`. `cmd/server/main.go` supplies that string as
`const lineupRoot = "scripts/lineups"`, with a comment already admitting it is interim. Templates
and CSS are embedded from `internal/web`, so the binary is half self-contained today: it renders
without the checkout and cannot find a single lineup without it.

`//go:embed` constrains where the tree can live. A directive names paths relative to the package
directory, cannot cross `..`, and does not follow symlinks — so a package must sit at or above the
tree. It also silently drops files whose names begin with `.` or `_` unless the pattern carries the
`all:` prefix.

Two questions were settled before this design, both by the user:

- The files are weekly lineup cards, not a standing roster. "Lineup" is the file, "starters" is its
  first nine, "bench" is the rest — so `internal/roster` is misnamed and becomes `internal/lineup`.
- The reader package embeds its own tree rather than importing a second data-only package, which
  would have been one letter away from it in every import block.

## Goals / Non-Goals

**Goals:**

- The server reads no file off the working directory, so a scratch-image container can serve it.
- The tree's location is a fact the composition root supplies, not one the reading package assumes.
- Adding a season is adding a directory: no directive edit, no list of seasons anywhere.
- `scripts/scores.sh` and `scripts/fantasycast.sh` keep working, per ADR 0004 decision 2.
- One vocabulary across the package, the tree, the specs, the docs, and the scripts.

**Non-Goals:**

- The Dockerfile, `fly.toml`, `PORT`, timeouts, and graceful shutdown. This change is their
  prerequisite, not their delivery.
- A dev flag that reads live lineups off disk. `os.DirFS` makes it a two-line addition later; until
  someone wants it, an unused flag is a second code path nobody tests.
- Any change to the CSV format, the parse rules, the starters/bench split, or the page.

## Decisions

### The tree is `internal/lineup/data/**`, embedded by its own reader package

`internal/lineup` gains `//go:embed data` and roots it with `fs.Sub(embedded, "data")`, exposing an
`fs.FS` that starts at the season directories. `data` is embedded as a directory, so `2027/` needs
no directive edit — and, unlike naming each season, a forgotten one cannot be a silent omission at
runtime.

*Alternatives:* a separate `internal/lineups` data package keeps CSVs out of a code package, but
puts `lineup` and `lineups` in the same import block. A top-level `lineups/` shortens the path the
CSVs are hand-edited at, at the cost of a publicly importable package in a repo that otherwise
keeps all Go under `cmd/` and `internal/`. Naming seasons directly (`//go:embed 2025 2026`) drops
the `data/` level entirely, and was rejected for the silent-omission failure mode.

### The tree is constructed from an `fs.FS`, supplied by `main`

`New` takes an `fs.FS` already rooted at the tree; the package exports the embedded one but never
defaults to it. `main` stays the only place that says which tree is served, exactly as it stays the
only place that names a stats provider.

Tests build trees with `fstest.MapFS`, which is why this is not merely a swap of one root for
another: today's tests either name the real `scripts/lineups` or write fixture directories to disk,
and both become an in-memory map with no I/O.

*Alternative:* `New(fs.FS, root string)` keeps a prefix inside the filesystem. Rejected — `fs.Sub`
already hands out a pre-rooted filesystem, so the second argument would be `"."` at every call
site.

### Resolution yields an `io/fs` name, not a path

`filepath.Join` becomes `path.Join`: `io/fs` names are always slash-separated, relative, and free
of `..`, on every platform. The existing team-name guard (`team != filepath.Base(team)`, no `..`)
already refuses everything `fs.ValidPath` would, and it keeps its own error message because it
knows what a team name is; `fs.ValidPath` on the joined name is the cheap backstop.

This is the visible break: `Path(2025, 14, "wood")` returns `2025/14/wood.csv`, and error text that
used to carry `scripts/lineups/...` now carries the name within the tree. Nothing outside the
package renders these strings — they reach the log, never the response body.

### Rename in the same commit sequence, not a follow-up

`roster` → `lineup`, `Roster` → `Lineup`, `lineupSize` → the starter count, and the two matchup
errors that say "rosters". Every caller of this package is already being touched by the `fs.FS`
change; renaming afterwards would touch them all a second time for no behavioural reason.

The rename goes *first*, while the tests are green, and not because a mechanical diff is nicer to
review before a behavioural one — though it is. `//go:embed` names paths relative to its package
directory, so where the tree lives is decided by what the package is called. `internal/lineup/data`
is only the right home once the package is `internal/lineup`.

The capability id `roster-source` stays put for this change: a delta archives into the folder it
came from. The spec-store rename (`roster-source` → `lineup-source`, `roster-score-report` →
`lineup-score-report`) is the last task, done after archive.

### Scripts follow the tree

`scores.sh` and `fantasycast.sh` build a lineup path from `$(dirname "$0")`. They point at
`../internal/lineup/data/<season>/<week>/` instead. Per the standing exemption, the scripts are
interim UI and are verified by running them, not by tests.

## Risks / Trade-offs

- **A deploy is the only way to fix a wrong lineup.** → Accepted, and the point: ADR 0004 decision
  3 chose it while lineups are hand-edited files in git. A wrong lineup is already a commit; this
  makes it a commit plus a build.
- **`//go:embed data` silently omits dotfiles**, so a `.DS_Store` in a week directory is embedded
  as nothing. → Harmless, and better than the disk behaviour: the "dotfiles are not counted"
  matchup scenario stays a real requirement for the `os.DirFS` and `fstest` paths, and the embedded
  tree simply never sees one. Not worth `all:`, which would embed the junk in order to then ignore
  it.
- **The tests stop reading the real tree.** `fstest.MapFS` means nothing exercises
  `internal/lineup/data/**` itself — a malformed CSV or a week with three files would build clean
  and fail at request time. → One test over the embedded tree, walking every week and reading both
  files, keeps that honest. It is a test of the data, not of the reader.
- **The CSVs now live under a path nobody would guess** (`internal/lineup/data/2025/15/`) for a
  file edited by hand every week. → `README.md` states the path in the one place a human looks for
  it, and the scripts take a team name rather than a path.
- **A big rename lands with a behavioural change**, which makes the diff harder to read. → The
  tasks order it: the rename lands first as a pure `git mv` and symbol-rename step that changes no
  assertion, so the behavioural diff that follows is small enough to read.

## Migration Plan

1. Rename the package, the type, and the errors, with behaviour unchanged and the suite green.
2. `git mv scripts/lineups internal/lineup/data` — one move, contents untouched, history intact —
   and point the still-working-directory-relative root at it, so the scripts and the server keep
   running while the reading changes underneath them.
3. Switch the package to `fs.FS` and its tests to `fstest.MapFS`.
4. Add the embed, wire `main` to it, and delete the directory-string constructor.
5. Fix the scripts and the docs.

No rollback plan beyond `git revert`: there is no deployed instance yet, and nothing persists.

## Open Questions

- Does `README.md` want the tree path stated once, or a `make lineup` helper that opens the right
  file? Path first; the helper is speculative until the path proves annoying.
