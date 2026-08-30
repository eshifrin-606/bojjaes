## 1. Clear the non-roster players file

- [x] 1.1 Delete `scripts/players/quarterbacks.csv` and the `scripts/players/` directory it leaves empty
- [x] 1.2 Grep the repo for references to `players/quarterbacks.csv` and remove or reword any that remain

## 2. Split the output into sections

- [x] 2.1 Add a named lineup-size constant (9) near the top of `scripts/scores.sh`
- [x] 2.2 Replace the single print loop with a starters loop over records 1–9 and a bench loop over the rest, both keeping the existing `printf '%-24s %s\n'` line and the `no stats` fallback
- [x] 2.3 Print the starters heading before the first section and the bench heading before the second, unconditionally
- [x] 2.4 Run `scripts/scores.sh 2025 14` against a live server and confirm nine starters, an empty bench section, and unchanged per-player values

## 3. Total the starters

- [x] 3.1 Add a single `jq` expression that takes the starter ids and sums the points of those present in `.scores`
- [x] 3.2 Print the total after the last starter and before the bench heading, aligned with the points column
- [x] 3.3 Verify against `scripts/bojjaes.csv` for 2025 week 14, where Derrick Harmon has no stats: the total must be the sum of the other eight, printed as a plain number with no marker
- [x] 3.4 Verify a fractional total is exact — Amon-Ra St. Brown's 3.5 must land the week 14 total on a clean `.5`, not a drifting decimal

## 4. Exercise the roster shapes

- [x] 4.1 Run against a team file from `scripts/teams/` (5 records) and confirm five starters and an empty bench section
- [x] 4.2 Build a temporary players file of more than nine records and confirm the split point, that both sections print in file order, and that the total covers only the first nine
- [x] 4.3 Confirm a leading comment line does not consume a starter slot

## 5. Update the docs

- [x] 5.1 Update the "Scoring a roster" example output in `README.md` to show the sectioned report with its total
- [x] 5.2 Describe the starters/bench split there: first nine records, positional convention with no lineup validation, bench shown without a total, and a no-stats starter contributing nothing to the total
- [x] 5.3 Correct the "output in file order" wording, now true within each section
- [x] 5.4 Update the header comment in `scripts/scores.sh` if it describes the flat output
