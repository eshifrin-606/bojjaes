Each behaviour below is one red-green pair. RED tasks are not done until the test has been run and
has failed **for the expected behavioural reason** — a wrong value, not a build error. So every RED
task assumes the symbols it calls already exist, stubbed to a wrong-but-compiling answer by the
preceding task. GREEN tasks make the smallest change that turns that failure into a pass, then
re-run the test.

## 1. Package skeleton

- [x] 1.1 Create `internal/roster/roster.go` with `package roster`, a `Record` struct holding `ID`
      and `Name` (both `string`, with a comment at `Name` recording that it is display-only and
      never leaves the roster), and an unexported `parse(io.Reader) ([]Record, error)` stubbed to
      return `nil, nil`. Confirm `go build ./...` passes — this task exists so the first RED can
      fail on a value.
- [x] 1.2 Create `internal/roster/roster_test.go` with an empty test file that compiles, and
      confirm `go test ./internal/roster` passes with no tests.

## 2. Reading records

- [x] 2.1 RED: test that parsing `4984,Josh Allen` yields one record with ID `4984` and Name
      `Josh Allen`. Run it; confirm it fails on the count/values returned by the stub, not on a
      build error.
- [x] 2.2 GREEN: read lines and split each on its first comma. Re-run; confirm pass.
- [x] 2.3 RED: test that a three-record input yields those three records **in file order**. Run;
      confirm it fails on order or count.
- [x] 2.4 GREEN: append records in read order, no sorting. Re-run; confirm pass.
- [x] 2.5 RED: test that a leading `#` comment line and a blank line between records are not
      records — three records in, three records out, positions unshifted. Run; confirm the extra
      lines appear as records.
- [x] 2.6 GREEN: skip blank lines and lines whose first non-space character is `#`. Re-run; confirm
      pass. Include an indented `#` line in the test input so the "first non-space" rule is what is
      being exercised, not "starts with `#`".
- [x] 2.7 RED: test that `4984 , Josh Allen` yields ID `4984` and Name `Josh Allen`. Run; confirm it
      fails on the untrimmed spaces.
- [x] 2.8 GREEN: trim surrounding spaces from both fields. Re-run; confirm pass.
- [x] 2.9 RED: test that `1234,Smith, Jr.` yields Name `Smith, Jr.`. Run; confirm it fails with the
      name truncated at the second comma (or passes trivially — if so, add a case that would fail
      under an all-commas split, so the first-comma rule is actually pinned).
- [x] 2.10 GREEN: split on the first comma only. Re-run; confirm pass.
- [x] 2.11 RED: test that a final line with **no trailing newline** still yields its record. Run;
      confirm the last record is missing.
- [x] 2.12 GREEN: handle the unterminated final line. Re-run; confirm pass.
- [x] 2.13 RED: test that `4984,` and a bare `4984` each yield a record with an empty Name rather
      than an error or a placeholder. Run; confirm it fails.
- [x] 2.14 GREEN: accept a missing name as an empty label. Re-run; confirm pass.

## 3. Refusing an unusable roster

- [x] 3.1 RED: test that a line yielding no id — `,Josh Allen` — returns an error, and that the
      error text names the **line number**. Run; confirm the line is silently skipped instead.
- [x] 3.2 GREEN: return an error naming the line number for any non-blank, non-comment line with an
      empty id. Re-run; confirm pass. Comment the *why* at the check: a skipped line shifts every
      record after it up one position and silently changes who starts.
- [x] 3.3 RED: test that a line of only spaces and a comma (`  ,  `) is refused rather than treated
      as blank. Run; confirm it is currently skipped as blank.
- [x] 3.4 GREEN: distinguish "blank line" from "line with fields that are all blank". Re-run;
      confirm pass.
- [x] 3.5 RED: test that input holding only comments and blank lines returns an error. Run; confirm
      it currently returns an empty roster and no error.
- [x] 3.6 GREEN: refuse a roster with no records. Re-run; confirm pass.
- [x] 3.7 RED: test that the same id on two records returns an error naming the repeated id. Run;
      confirm both records are currently accepted.
- [x] 3.8 GREEN: reject duplicate ids after the whole file is read. Re-run; confirm pass.
- [x] 3.9 RED: test that two records with **different ids and the same name** are accepted. Run;
      confirm — this guards the duplicate check against being written against the wrong field. If
      it passes immediately, keep it as a regression test and note that in the commit.

## 4. Starters and bench

- [x] 4.1 Add `Roster` (holding the ordered records) and a `Starters()`/`Bench()` split — or a
      single `Split()` — stubbed to return everything as starters and an empty bench, so the next
      task fails on a value. Confirm it builds.
- [x] 4.2 RED: test that a twelve-record roster yields nine starters and three bench, each in file
      order. Run; confirm it fails on the counts.
- [x] 4.3 GREEN: split at nine, using a named constant whose comment states the league rule. Re-run;
      confirm pass.
- [x] 4.4 RED: test that a five-record roster yields five starters and an **empty, non-nil** bench.
      Run; confirm the current behaviour differs (nil bench, or a panic on the short slice).
- [x] 4.5 GREEN: handle rosters shorter than the lineup. Re-run; confirm pass.
- [x] 4.6 RED: test that an exactly-nine roster yields nine starters and an empty bench. Run.
- [x] 4.7 GREEN: make it pass if it does not already; if it passes, keep it as the boundary case and
      say so.
- [x] 4.8 RED: test that comment and blank lines interleaved among nine records leave all nine as
      starters — no line pushed onto the bench. Run.
- [x] 4.9 GREEN: confirm pass; this pins parsing and splitting together, which is where a positional
      rule breaks.
- [x] 4.10 RED: test that moving the tenth record above the ninth makes it a starter and demotes the
      record it displaced. Run.
- [x] 4.11 GREEN: confirm pass. Comment that this is intended behaviour of a hand-maintained lineup
      card, not an accident of the split.

## 5. Locating a roster in the lineup tree

- [x] 5.1 Add a constructor taking the lineup-tree root (e.g. `New(root string) *Tree`) and a path
      resolver stubbed to return a wrong-but-plausible path, so the next task fails on a value.
      Confirm it builds.
- [x] 5.2 RED: test that root `scripts/lineups`, season 2025, week 14, team `wood` resolves to
      `scripts/lineups/2025/14/wood.csv`. Run; confirm it fails on the path string.
- [x] 5.3 GREEN: build the path as `<root>/<season>/<week>/<team>.csv`. Re-run; confirm pass.
- [x] 5.4 RED: test that a team name containing `/` is refused with an error and that **no file is
      opened**. Run; confirm it currently resolves.
- [x] 5.5 GREEN: reject a team name that is not a single path segment. Re-run; confirm pass.
- [x] 5.6 RED: test that a team name of `..` (and one containing `..`) is refused. Run.
- [x] 5.7 GREEN: make it pass; keep the check in one place next to the layout it protects.

## 6. Reading a located roster from disk

- [x] 6.1 Add the exported read-by-season/week/team entry point, stubbed to return an empty roster,
      and add `internal/roster/testdata/` fixtures mirroring the real files' shape — a leading
      comment line and `id,name` records. Confirm it builds.
- [x] 6.2 RED: test, against a `t.TempDir()` tree, that reading season/week/team returns the records
      of the file at the resolved path. Run; confirm it fails on the empty result.
- [x] 6.3 GREEN: open the resolved path and parse it. Re-run; confirm pass.
- [x] 6.4 RED: test that a missing roster file returns an error whose text **names the path it
      looked for**. Run; confirm the error is currently unhelpful or absent.
- [x] 6.5 GREEN: wrap the open error with the resolved path. Re-run; confirm pass.
- [x] 6.6 RED: test that a parse error from a located file names the **file** as well as the line
      number. Run; confirm the file name is missing.
- [x] 6.7 GREEN: wrap parse errors with the file path. Re-run; confirm pass.

## 7. Close out

- [x] 7.1 Re-read `internal/roster` as a whole and delete any comment that restates the code; keep
      only the ones carrying a rule or a constraint (display-only name, the nine, why a bad line is
      refused rather than skipped).
- [x] 7.2 Run `gofmt -l ./internal/roster`, `go vet ./...`, and `go test ./...`; confirm clean and
      that nothing in `internal/score` or `cmd/server` changed.
- [x] 7.3 Confirm `scripts/scores.sh` and `scripts/fantasycast.sh` are untouched by `git status`,
      and that `scripts/scores.sh 2025 14` still produces the same report against a running server.
- [x] 7.4 Walk the `roster-source` spec scenario by scenario and confirm each one has a test that
      exercises it; add any that the loop above did not produce.
- [x] 7.5 Tick the backlog's "Move roster/lineup knowledge out of `scripts/scores.sh`..." line, and
      note in it that the bash parsing deliberately remains until the page replaces the scripts.
