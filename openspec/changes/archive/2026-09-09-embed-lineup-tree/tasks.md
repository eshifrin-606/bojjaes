Each behaviour below is one red-green pair. A RED task is not done until the test has been run and
has failed **for the expected behavioural reason** — a wrong value or a wrong error, not a build
error and not a missing symbol. Where a step would otherwise fail to compile, the preceding GREEN
scaffold deliberately ships a *wrong* value for the next RED to fail against.

The order is: rename while green, then change how files are read, then embed. The rename comes
first because `//go:embed` names paths relative to its package directory — where the tree lives is
decided by what the package is called, so the package has to be called the right thing before the
tree can move into it.

`scripts/*.sh` are interim UI and exempt from TDD; they are verified by running them.

## 1. Rename, with behaviour unchanged

- [x] 1.1 `git mv internal/roster internal/lineup`. Rename the package clause, `Roster` → `Lineup`,
      `lineupSize` → `starterCount`, `ErrTooFewRosters` → `ErrTooFewLineups`, `ErrTooManyRosters` →
      `ErrTooManyLineups`, and the "roster" wording in doc comments and error strings. `ErrNoWeek`,
      `ErrNotOurMatchup`, `Tree`, `Record`, `Starters`, and `Bench` keep their names.
- [x] 1.2 Update the importers — `internal/web/matchup.go`, its test, `cmd/server/main.go`. Run
      `go build ./... && go test ./...`; confirm green. No test assertions should have needed to
      change: this task changes no behaviour.
- [x] 1.3 `git mv scripts/lineups internal/lineup/data`. Point `lineupRoot` in `cmd/server/main.go`
      at `internal/lineup/data`, and update the `New("scripts/lineups")` calls and path assertions
      in the package's tests. Run the suite; confirm green. The root is still a working-directory
      path — that is what section 3 removes.
- [x] 1.4 Update `scripts/scores.sh` and `scripts/fantasycast.sh` to build their lineup path from
      `$(dirname "$0")/../internal/lineup/data`. Run each by hand for a week that exists and one
      that does not; confirm the output and the error message are what they were before the move.

## 2. Reading through a supplied filesystem

- [x] 2.1 GREEN scaffold: give `Tree` an `fs.FS` field and add `NewFS(fs.FS) *Tree` beside the
      existing `New(root string)`. Have `NewFS` set the root to the literal `internal/lineup/data`
      — deliberately wrong, and what 2.2 fails against. `Read` and `Matchup` still use `os`.
      Confirm it builds and the suite is green; `NewFS` has no callers yet.
- [x] 2.2 RED: test that `NewFS(...).Path(2025, 14, "wood")` is `2025/14/wood.csv`. Run; confirm it
      fails on the `internal/lineup/data/` prefix rather than on a missing symbol.
- [x] 2.3 GREEN: resolve paths relative to the supplied filesystem — no root field in the `NewFS`
      path, `filepath.Join` → `path.Join`. Keep `New(root string)` working by defining it as
      `NewFS(os.DirFS(root))`, and update the existing `Path` assertions, which now name a lineup
      within the tree rather than a path from the working directory. Re-run; confirm green.
- [x] 2.4 RED: test that `Read` returns the records of a lineup that exists only in an
      `fstest.MapFS`. Run; confirm it fails because `Read` went to the disk and found nothing —
      a "no such file" error, not a compile error.
- [x] 2.5 GREEN: `Read` opens through the tree's filesystem. Re-run; confirm pass, and confirm the
      error path still names the file it looked for.
- [x] 2.6 RED: test that `Matchup` resolves a week held only in an `fstest.MapFS` — two files,
      `bojjaes.csv` and `wood.csv` — to those two teams. Run; confirm it fails with `ErrNoWeek`
      from the `os.ReadDir` on a path that is not on disk.
- [x] 2.7 GREEN: `Matchup` lists through the tree's filesystem with `fs.ReadDir`. Re-run; confirm
      pass, and confirm the existing refusals — three lineups, one lineup, no `bojjaes.csv`,
      dotfiles and non-`.csv` entries ignored — still pass unchanged against a `MapFS` week.
- [x] 2.8 RED: test the two-filesystems scenario — the same season, week, and team resolved against
      two different `fstest.MapFS` trees yields each tree's own lineup. Run; confirm it fails only
      if some read still reaches past the supplied filesystem. If it passes first time, say so and
      keep it: it is the regression guard for exactly that.
- [x] 2.9 Convert the remaining tests in the package and in `internal/web` off `t.TempDir()` and
      off named directories, onto `fstest.MapFS`. Run the full suite; confirm green and confirm
      nothing writes to disk.
- [x] 2.10 Consider an `fs.ValidPath` backstop on the joined name. Add it only if a test can reach
      it past the existing team-name guard; if nothing can, note that in the code review of this
      task and skip it rather than adding an unreachable branch.

## 3. Carrying the tree in the binary

- [x] 3.1 GREEN scaffold: add the embed to `internal/lineup` — `//go:embed data`, and an exported
      `fs.FS` rooted with `fs.Sub`, deliberately rooted at `.` rather than `data`. Confirm it
      builds.
- [x] 3.2 RED: test that the exported filesystem holds `2025/15/bojjaes.csv`. Run; confirm it fails
      because the name in it is `data/2025/15/bojjaes.csv`.
- [x] 3.3 GREEN: root the sub-filesystem at `data`, panicking at package initialisation if it
      fails, the way `internal/web` already fails a broken template at startup. Re-run; confirm
      pass.
- [x] 3.4 RED: test that a tree over the embedded filesystem resolves 2025 week 15 to `bojjaes` and
      `aroma` and reads both lineups. Run; confirm it fails only if the move in 1.3 lost a file.
- [x] 3.5 Add a test that walks every `<season>/<week>` in the embedded tree, asserting each is a
      matchup and that both lineups parse. This is a test of the data, not of the reader: with the
      unit tests on `fstest.MapFS`, nothing else reads the real files. Run; confirm it passes over
      all four weeks.
- [x] 3.6 Delete `lineupRoot` from `cmd/server/main.go` and construct the tree from the embedded
      filesystem. Delete `New(root string)` and rename `NewFS` → `New`. Run `go build ./... &&
      go test ./...`; confirm green.
- [x] 3.7 Build the binary, copy it to an empty directory, run it from there, and `curl`
      `/2025/15`. Confirm the page renders. This is the check the whole change exists for — the
      unit tests cannot tell you the working directory stopped mattering.
- [x] 3.8 From that same empty directory, confirm a week that is not in the tree still 404s and
      that the log names what it looked for.

## 4. Docs and vocabulary

- [x] 4.1 `README.md`: the tree's path, the `scripts/scores.sh` examples, and "roster" → "lineup"
      wherever it means the file. Say once, where a reader will look for it, that lineups are
      hand-edited at `internal/lineup/data/<season>/<week>/<team>.csv` and that changing one means
      rebuilding.
- [x] 4.2 `docs/architecture.md` and `docs/package-dependencies.md`: the tree node in both
      diagrams, the package name, and the paragraph in `package-dependencies.md` that calls the
      root a working-directory constant — it is now an embedded filesystem supplied by `main`.
- [x] 4.3 Amend `docs/adr/0004-web-frontend-stack.md` in place — decisions 3 and 4 name
      `scripts/lineups/**`. The decision itself is unchanged and this is the change that carries it
      out, so this is a path and vocabulary edit, not a new ADR or a superseding note.
- [x] 4.4 `backlog.md`: drop the two lines this change completes — the embed line and the
      "keep the scripts working" line — leaving the Dockerfile and `fly.toml` lines, which this
      unblocks.
- [x] 4.5 `grep -ri roster` across the repo, excluding `openspec/changes/archive/`. Every survivor
      should be either an OpenSpec capability id (section 5) or a place where "roster" is the right
      word. Fix or justify each.

## 5. After archive

- [x] 5.1 Once this change is archived and its delta is synced into `openspec/specs/`, `git mv`
      `openspec/specs/roster-source` → `lineup-source` and `roster-score-report` →
      `lineup-score-report`, rename the `# roster-source` headings, and update the cross-references
      in the other specs and in `README.md`. A delta has to archive into the folder it came from,
      which is why this trails the change rather than living inside it.
