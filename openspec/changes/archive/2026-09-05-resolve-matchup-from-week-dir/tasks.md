Each behaviour below is one red-green pair. A RED task is not done until the test has been run and
has failed **for the expected behavioural reason** — a wrong value, not a build error. So every RED
task assumes the symbols it calls already exist, stubbed to a wrong-but-compiling answer by the
preceding task. GREEN tasks make the smallest change that turns that failure into a pass, then
re-run the test.

## 1. The week directory

- [x] 1.1 Add an unexported helper on `Tree` that resolves a season and week to the week directory
      path, stubbed to return a wrong-but-plausible path so the next task fails on a value. Confirm
      `go build ./...` passes.
- [x] 1.2 RED: test that root `scripts/lineups`, season 2025, week 14 resolves to
      `scripts/lineups/2025/14`. Run; confirm it fails on the path string, not on a build error.
- [x] 1.3 GREEN: build the path from the same layout knowledge `Path` uses, so the directory and the
      files inside it cannot disagree about where the tree is. Re-run; confirm pass.

## 2. Resolving a two-roster week

- [x] 2.1 Add the exported `Matchup(season, week int) (ours, theirs string, err error)` on `Tree`,
      stubbed to return two empty strings and a nil error. Add an unexported `ourTeam = "bojjaes"`
      constant with a comment naming it as our team. Confirm it builds.
- [x] 2.2 RED: test, against a `t.TempDir()` week directory holding empty `bojjaes.csv` and
      `wood.csv`, that `Matchup` returns `bojjaes` and `wood`. Run; confirm it fails on the empty
      strings. Note the test helper seeds **empty** files — resolution must never open them.
- [x] 2.3 GREEN: list the directory, take the two entries, strip `.csv`, return them. Re-run;
      confirm pass.
- [x] 2.4 RED: test that a week holding `bojjaes.csv` and `aroma.csv` — an opponent sorting *before*
      us — still returns `bojjaes` first and `aroma` second. Run; confirm it fails on the order,
      since directory listing is sorted.
- [x] 2.5 GREEN: identify our file by comparing against the `ourTeam` constant and return it first,
      the other second. Re-run; confirm pass. Comment that the opponent is found by elimination, not
      by any property of its own.
- [x] 2.6 RED: test that a week whose `wood.csv` contains content that `parse` would reject — a line
      with no id — still resolves to both names. Run; confirm — if it passes immediately, keep it as
      the regression test that pins "resolution does not read the files" and say so in the commit.

## 3. Refusing a week that is not a matchup

- [x] 3.1 RED: test that a week holding `bojjaes.csv`, `wood.csv`, and `aroma.csv` returns an error,
      and that the error text names the **directory** and **all three** files. Run; confirm a pair
      is currently returned instead.
- [x] 3.2 GREEN: refuse three or more roster files. Re-run; confirm pass. Comment the *why* at the
      check: there is no rule by which two of three are the matchup, and a guess renders a full,
      plausible page for a game nobody plays.
- [x] 3.3 RED: test that a week holding only `bojjaes.csv` returns an error naming the directory.
      Run; confirm it currently returns a matchup with one side empty.
- [x] 3.4 GREEN: refuse fewer than two roster files. Re-run; confirm pass.
- [x] 3.5 RED: test that a week directory that exists but holds no `.csv` files returns an error
      naming the directory. Run.
- [x] 3.6 GREEN: make it pass; confirm the message is the too-few case and not a bare read error.
- [x] 3.7 RED: test that a week directory that does **not exist** returns an error naming the
      directory it looked in, rather than surfacing a bare `os.ReadDir` `ENOENT`. Run; confirm the
      current error does not name the week.
- [x] 3.8 GREEN: wrap the read failure with the resolved directory. Re-run; confirm pass.
- [x] 3.9 RED: test that a week holding `aroma.csv` and `fuego.csv` — exactly two rosters, neither
      ours — returns an error. Run; confirm it currently returns a pair in listing order.
- [x] 3.10 GREEN: require `bojjaes.csv` to be one of the two. Re-run; confirm pass. Comment that
      this resolver answers "who are we playing", which a directory of two other teams cannot.
- [x] 3.11 Confirm the four refusals — too many, too few, missing directory, no `bojjaes.csv` — are
      distinguishable by a caller, not four wordings of one error. Add a test that asserts whichever
      mechanism was chosen (sentinel `errors.Is` targets, or distinct message substrings).

## 4. Entries that are not rosters

- [x] 4.1 RED: test that a week holding `bojjaes.csv`, `wood.csv`, and a `notes.md` resolves to
      `bojjaes` and `wood`. Run; confirm it fails as a three-file error.
- [x] 4.2 GREEN: count only names ending in `.csv`. Re-run; confirm pass.
- [x] 4.3 RED: test that a `.DS_Store` alongside the two rosters does not make the week an error.
      Run; confirm it currently — depending on 4.2's implementation — either passes, in which case
      keep it as the regression test that matters most on this hand-edited tree, or fails.
- [x] 4.4 GREEN: skip entries whose name begins with `.`. Re-run; confirm pass. Comment that Finder
      writes `.DS_Store` into any directory it opens, so counting it would break every week.
- [x] 4.5 RED: test that a dot-prefixed CSV — an editor's `.wood.csv.swp`, and a `.wood.csv` — is
      skipped rather than counted as a roster. Run.
- [x] 4.6 GREEN: confirm the dotfile rule covers it; make it pass if not.
- [x] 4.7 RED: test that a **subdirectory** inside the week directory is not counted, including one
      named `something.csv`. Run; confirm it is currently counted as a file.
- [x] 4.8 GREEN: count only regular files. Re-run; confirm pass.
- [x] 4.9 RED: test that a genuine third roster is still refused when junk is also present —
      `bojjaes.csv`, `wood.csv`, `aroma.csv`, plus a `.DS_Store` — so the filter did not weaken the
      count. Run; confirm.
- [x] 4.10 GREEN: confirm pass; keep this as the test that pins the filter and the count against
      each other.
- [x] 4.11 RED: test that `BOJJAES.csv` and `wood.csv` fails as the no-bojjaes case rather than
      resolving. Run; confirm. This pins the exact, lowercase comparison so the result does not
      depend on the filesystem's case sensitivity.
- [x] 4.12 GREEN: confirm pass; note the constraint in a comment if the comparison is not obviously
      exact.

## 5. Close out

- [x] 5.1 Re-read the new code as a whole and delete any comment that restates it; keep only the
      ones carrying a rule or a constraint (why three files is refused, why `.DS_Store` is skipped,
      that the opponent is found by elimination, what `ourTeam` is).
- [x] 5.2 Run `gofmt -l ./internal/roster`, `go vet ./...`, and `go test ./...`; confirm clean and
      that nothing in `internal/score` or `cmd/server` changed.
- [x] 5.3 Confirm by `git status` that `scripts/**` is untouched, and run `Matchup` once against the
      **real** tree — `scripts/lineups/2025/{14,15,16}` — from a throwaway test or `go run` snippet,
      confirming all three weeks resolve and that the three opponents are `wood`, `aroma`, `gonads`.
      Delete the snippet; the suite stays off the real tree.
- [x] 5.4 Walk the `roster-source` delta scenario by scenario and confirm each has a test that
      exercises it; add any the loop above did not produce.
- [x] 5.5 Tick the backlog's "Resolve a matchup from a week directory" line.
