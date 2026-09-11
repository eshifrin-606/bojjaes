Each behaviour below is one red-green pair. A RED task is not done until the test has been run and
has failed **for the expected behavioural reason** — the page rendering `no stats` where `--` is
expected, or a placeholder where a number is expected — not a build error. Every change here is a
string value, so no scaffolding step is needed to keep REDs off the compiler.

Tests use the existing `fixtureWeek`, `fixtureStats`, `fakeSource`, and `renderedStarters` /
`renderedTotals` helpers in `internal/web/matchup_test.go`.

## 1. An absent starter renders `--`

- [x] 1.1 RED: in `TestAMissingStarterIsNotZero`, change the expected line from
      `"Brandon Aubrey=no stats"` to `"Brandon Aubrey=--"`. Leave the `=0` assertion as it is. Run
      `go test ./internal/web -run TestAMissingStarterIsNotZero`; confirm it fails because the
      rendered starters contain `Brandon Aubrey=no stats`.
- [x] 1.2 GREEN: change `noStats` in `internal/web/matchup.go` to `"--"`. Re-run; confirm pass.
- [x] 1.3 RED: add an assertion (in the same test or beside it) that the rendered page body
      contains no `no stats` anywhere, so the old wording cannot survive in another element. Break
      it on purpose by adding `no stats` to the template (e.g. inside the column), watch it fail,
      restore, and confirm pass.
- [x] 1.4 Confirm `TestAnAbsentStarterContributesNothingToTheTotal` and `TestAScorelessStarterIsZero`
      pass unchanged — the total arithmetic and the scoreless `0` are untouched.

## 2. The total never renders the placeholder

- [x] 2.1 RED: add a test serving the page with a stats fixture that holds none of one column's
      nine starters. Assert every starter line in that column ends `=--` and that column's rendered
      total is `0`. Run; this passes against today's `scoreColumn`, so keep it as the regression
      test that pins the spec's "a column with no stats at all totals zero" scenario and say so.
      Then break it on purpose — have `scoreColumn` set `Total` to `noStats` when no starter
      played — watch it fail on the total, restore, and confirm pass.

## 3. Close-out

- [x] 3.1 In `scoreColumn`, remove the comment sentence saying the wording matches
      `scripts/scores.sh`; keep the sentence about absence and a scoreless week being different
      facts. Confirm the `noStats` declaration comment still reads true for `--`.
- [x] 3.2 Confirm `scripts/scores.sh`, the README, and `internal/web/matchup.html` are unchanged
      (`git diff --stat`).
- [x] 3.3 `go test ./...` and `go vet ./...` clean; `openspec validate show-dashes-for-missing-stats
      --strict` clean.
