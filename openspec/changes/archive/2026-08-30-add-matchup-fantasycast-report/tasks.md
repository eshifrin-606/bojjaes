## 1. Prepare `scores.sh` for composition

- [x] 1.1 Remove the `season <n> week <n>` line from `scripts/scores.sh` so the report begins at the
      `STARTERS` heading, and confirm a direct run's first line of output is `STARTERS`
- [x] 1.2 Change the per-player `printf` to a fixed-width right-aligned points field, and confirm a
      one-digit score, a fractional score, and a `no stats` marker all occupy the same field width
- [x] 1.3 Confirm `scripts/scores.sh 2025 14` and `scripts/scores.sh 2025 14 wood` still print the
      same players, totals, and sections as before, against a running server

## 2. Argument handling in `fantasycast.sh`

- [x] 2.1 Create `scripts/fantasycast.sh` with the file header comment, `set -euo pipefail`, and a
      `usage` covering `<season> <week> <team> [team2]` and the `SERVER` env var
- [x] 2.2 Reject zero team arguments and three or more with usage and a non-zero exit, before any
      server contact
- [x] 2.3 Validate season and week as digits, matching `scores.sh`
- [x] 2.4 Resolve the one-argument form to `bojjaes` on the left and the named team on the right;
      resolve the two-argument form to the teams in the order given
- [x] 2.5 Pass team names through to `scores.sh` unchanged, so name resolution stays in one place and
      an unknown name fails with `scores.sh`'s existing message

## 3. Column layout

- [x] 3.1 Capture both reports into variables by running `scores.sh` once per team, before printing
      anything, so a failure of either exits non-zero with no partial columns
- [x] 3.2 Read each captured report into an array with `while IFS= read -r`, preserving blank
      separator lines; use no bash-4 constructs (`mapfile`, `readarray`, `declare -n`)
- [x] 3.3 Compute the left report's widest line and pad every left line to it plus a two-space
      gutter, so the right column starts at the same offset on blank lines, headings, totals, and
      long names alike
- [x] 3.4 Iterate to the longer of the two line counts, substituting an empty left line past the end
      of the shorter report, and confirm no line of the longer column is dropped
- [x] 3.5 Print the season and week once above both columns and nowhere else

## 4. Verify against the specified behavior

- [x] 4.1 Run `scripts/fantasycast.sh 2025 14 wood` against a running server and confirm the Bojjaes
      are the left column, Wood the right, both `TOTAL` lines on the same output line, and no margin
      or winner printed
- [x] 4.2 Run the two-argument form and confirm argument order is column order and the Bojjaes do not
      appear
- [x] 4.3 With rosters of different lengths, confirm every player of the longer roster prints and the
      shorter column simply ends with nothing to its right
- [x] 4.4 Confirm a starter with no stats prints its no-stats marker in its column and adds no
      annotation to either total
- [x] 4.5 Confirm a failing team argument and a server error each exit non-zero with the underlying
      message and print no columns

## 5. Documentation

- [x] 5.1 Update the `scores.sh` example output in `README.md` to drop the season/week line and show
      the aligned points column
- [x] 5.2 Add a matchup section to `README.md`: usage, the one-argument opponent default, column
      order, why there is no margin line, and that each team is scored in its own request so the
      26-id cap is per roster
- [x] 5.3 Move the "improve flexibility of player-week fantasy score feature" style entry in
      `TODO.md` as appropriate and note the matchup report under Recently Completed
