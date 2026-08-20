Follow the red-green TDD loop from `CLAUDE.md`: write the failing test first, then the code that passes it.

## 1. Generalize the yardage threshold helper

- [x] 1.1 Confirm the existing `TestPoints` cases in `internal/score/calc_test.go` pass unmodified, and
      leave them unmodified for the whole of this section — they are the regression net for the refactor
- [x] 1.2 Parameterize `thresholdBonus` with the increment size and per-increment value, updating the
      three rushing/receiving call sites to pass 10 and 0.5
- [x] 1.3 Confirm `TestPoints` is still green with no test edits; this is the only evidence the refactor
      changed nothing

## 2. Passing yardage bonus

- [x] 2.1 Write a failing test: 199 passing yards scores 0
- [x] 2.2 Add `PassYd` to `StatLine` and the 200-yard/3-point award to `Points`, making 2.1 and a
      249-yards-scores-3 case pass
- [x] 2.3 Write a failing test that 250 passing yards scores 4, pinning the 50-yard increment
- [x] 2.4 Write a failing test that 299 scores 4 and 300 scores 5, pinning the floor at the second
      increment rather than only at the first
- [x] 2.5 Implement the increment and make 2.3 and 2.4 pass

## 3. Passing yardage stacks with rushing/receiving

- [x] 3.1 Write a failing test: 250 passing yards and 80 rushing yards scores 7, not 4
- [x] 3.2 Make it pass by adding the passing award to the total rather than into the existing `max()`
- [x] 3.3 Write a test that a stat line with no passing stats scores exactly what it scored before,
      using a case that already exists in `TestPoints`

## 4. Passing touchdowns and the long-touchdown bonus

- [x] 4.1 Write a failing test: 2 passing touchdowns score 12
- [x] 4.2 Add `PassTD` to `StatLine` and the 6-point term to `Points`
- [x] 4.3 Write a failing test: 2 passing touchdowns with one of 40+ yards score 13
- [x] 4.4 Extend `TD40Plus` to cover passing touchdowns and widen its comment; make 4.3 pass

## 5. Interceptions and two-point conversions

- [x] 5.1 Write a failing test: 3 interceptions thrown score -9
- [x] 5.2 Add `PassInt` to `StatLine` and the -3 term to `Points`
- [x] 5.3 Write a failing test that an interception against a scoring line lands on a distinguishable
      number (e.g. 250 passing yards and 1 interception scores 1), so a missing penalty cannot hide
      behind a 0
- [x] 5.4 Write a failing test: one two-point conversion scores 2, and two score 4
- [x] 5.5 Add `TwoPt` to `StatLine` and the 2-point term to `Points`

## 6. Map the Sleeper passing keys

- [x] 6.1 Add a quarterback entry to `internal/score/testdata/week14.json` carrying `pass_yd`, `pass_td`,
      `pass_td_40p`, `pass_int`, and `pass_2pt`, plus rushing stats, taken from the real 2025 week 14
      payload so the fixture stays a true sample (Josh Allen, ID 4984, fits every field)
- [x] 6.2 Write a failing `statLineFrom` test asserting each new field individually with a distinct
      non-zero value, so a mistyped key cannot pass by reading zero
- [x] 6.3 Write a failing test that `pass_rush_yd` does not contribute to `PassYd` — it is passing plus
      rushing yards combined, and the existing running back fixture already carries it, so mapping it
      would give that back phantom passing yardage
- [x] 6.4 Write a failing test that a `rush_2pt` and a `rec_2pt` entry each reach the two-point count,
      not only `pass_2pt`
- [x] 6.5 Map the seven keys in `statLineFrom`, summing `pass_td_40p` into `TD40Plus` alongside the
      rushing and receiving 40+ counts and the three conversion keys into `TwoPt`, and make section 6
      pass
- [x] 6.6 Write a failing end-to-end calculator test scoring the quarterback fixture through
      `statLineFrom` and `Points` against a hand-computed total, covering the mapping and the rules
      together

## 7. Confirm existing behavior is unchanged

- [x] 7.1 Run the full package test suite and confirm the handler, batch, and Sleeper tests pass with no
      edits to their expected scores
- [x] 7.2 Verify the stub-entry fixture (no mapped stat keys) still maps to an all-zero stat line and
      scores 0, now across the passing fields too

## 8. Documentation

- [x] 8.1 Update the sample response in `README.md` to include the new stat fields
- [ ] 8.2 When syncing the delta spec, narrow the capability's Purpose statement: passing and two-point
      conversions are no longer unspecified categories, and a quarterback's score is no longer
      incomplete — kicking and defense remain
- [x] 8.3 Run `scripts/scores.sh 2025 14 scripts/bojjaes.csv` against a local server and confirm Josh
      Allen scores his full line rather than his rushing alone
