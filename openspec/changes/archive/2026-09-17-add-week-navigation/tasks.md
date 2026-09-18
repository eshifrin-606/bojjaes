Each behaviour below is one red-green pair. A RED task is done only after the test has run and
failed **for the expected behavioural reason**: a wrong value or a missing link, not a build error
or a missing symbol. Where a new symbol is needed, add it first as a stub that compiles and returns
the wrong answer, then write the test against the stub. Where a test passes as soon as it is
written, the task says so: keep it as a regression pin, break the production code on purpose to
watch it fail, then restore.

Rendered-layout tests (section 5) must be seen **running, not skipping**, before any RED/GREEN in
them counts.

## 0. Before starting

- [x] 0.1 Confirm `go test ./...` passes on this branch before touching anything.

## 1. The tree answers whether a week exists

- [x] 1.1 Stub: add `func (t *Tree) HasWeek(season, week int) bool { return false }` to
      `internal/lineup/matchup.go`, next to `weekDir`. Confirm it builds.
- [x] 1.2 RED: in `internal/lineup/matchup_test.go`, add `TestHasWeekFindsAWeekDirectory` over a
      `MapFS` holding `2026/2/bojjaes.csv` and `2026/2/renegades.csv`. Assert `HasWeek(2026, 2)` is
      true. Run it and confirm it fails on `false`.
- [x] 1.3 GREEN: implement with `fs.Stat(t.fsys, t.weekDir(season, week))` and `err == nil`.
      Re-run and confirm it passes.
- [x] 1.4 Pin: `TestHasWeekIsFalseForAMissingWeek`, with the tree holding 2026 weeks 1 and 2, and
      `HasWeek(2026, 3)` false. It passes as written. Break it by temporarily returning `true`,
      watch it fail, restore.
- [x] 1.5 RED: `TestHasWeekIsFalseForAFileWhereAWeekShouldBe`, with a regular file at `2026/3`.
      Run it and confirm it fails (`Stat` succeeds on the file).
- [x] 1.6 GREEN: add `&& info.IsDir()`. Re-run and confirm it passes.
- [x] 1.7 Pin: `TestHasWeekIsTrueForAWeekThatIsNotAMatchup`, table-driven over three rosters, a
      lone roster, no Bojjaes, and a `wood.csv` whose contents would fail to parse. Each is true.
      Break it by temporarily making `HasWeek` return `t.Matchup(...)`'s `err == nil`, watch the
      rows fail, restore.

## 2. The view model carries the adjacent weeks

- [x] 2.1 Add `PrevWeek, NextWeek int` to `matchup` in `internal/web/matchup.go`, with the one-line
      comment that 0 means that week does not exist. Leave both unset. Confirm it builds.
- [x] 2.2 In `internal/web/matchup_test.go`, add a helper `treeFS(weeks ...fstest.MapFS)
      fstest.MapFS` that merges several `weekFS` values. Add a helper that returns a rendered
      page's `rel="prev"` and `rel="next"` hrefs (empty when absent).
- [x] 2.3 RED: `TestAMiddleWeekLinksBothWays`, with the tree holding 2026 weeks 1, 2, and 3 (each a
      valid matchup), requesting `/2026/2`. Assert the prev href is `/2026/1` and the next href is
      `/2026/3`. Run it and confirm it fails with no links found.
- [x] 2.4 GREEN (markup): add the `<nav class="weeks">` of design decision 5 above `<main>` in
      `matchup.html`, with both `{{with}}` slots and the `h1` label. Confirm 2.3 still fails, now
      because both weeks are 0.
- [x] 2.5 GREEN (handler): after both rosters are read and before the fetch, set
      `view.PrevWeek`/`view.NextWeek` to `week∓1` when `tree.HasWeek` says so. Re-run 2.3 and
      confirm it passes. (The view is built after the fetch today. Compute the two weeks into locals
      before the fetch and assign them when the view is built.)
- [x] 2.6 Pin: `TestTheFirstWeekHasNoPreviousLink`, with the tree holding a directory at
      `2026/0` (bypassing the tree's layout on purpose) plus weeks 1 and 2, requesting `/2026/1`.
      Assert no prev link and a next link to `/2026/2`. It passes as written: week − 1 is 0, which
      the view model already reads as absent (design decision 4), so no break short of changing
      the template exists.
- [x] 2.7 RED: `TestTheLastWeekOfTheRangeHasNoNextLink`, with the tree holding `2026/17`,
      `2026/18`, and `2026/19`, requesting `/2026/18`. Assert no next link. Run it and confirm it
      fails on a next link to `/2026/19`.
- [x] 2.8 GREEN: skip a candidate week that `score.ValidateSeasonWeek` refuses (design decision 3).
      Re-run 2.6 and 2.7 and confirm both pass.
- [x] 2.9 Pins, each passing as written. Break by temporarily setting both links unconditionally,
      watch them fail, restore:
      - `TestTheLatestWeekHasNoNextLink`: 2026 weeks 1 and 2, `/2026/2`.
      - `TestLinksDoNotCrossASeason`: 2025 week 16 and 2026 week 1. Neither page links to the other.
      - `TestAGapIsNotSkipped`: 2026 weeks 1 and 3, `/2026/3` has no prev link.
- [x] 2.10 Pin: `TestABrokenNeighbourStillGetsALink`: 2026 week 3 holds three rosters, `/2026/2`
      links next to `/2026/3`. It passes as written. Break it by temporarily gating on
      `tree.Matchup` succeeding, watch it fail, restore.

## 3. Only the shown week is fetched

- [x] 3.1 Give `fakeSource` a record of the `(season, week)` of each call. Confirm existing tests
      still pass.
- [x] 3.2 Pin: `TestOnlyTheShownWeekIsFetched`, with 2026 weeks 1, 2, and 3 and `/2026/2`. Assert
      exactly one call, for 2026 week 2. It passes as written. Break it by temporarily fetching
      `week+1` as well, watch it fail, restore.

## 4. The label and the link text

- [x] 4.1 RED: `TestThePageNamesItsWeek`, requesting `/2026/2`, asserts the `h1` text is
      `2026 · Week 2`. If 2.4's markup already makes it pass, record it as a pin: break the
      separator, watch it fail, restore.
- [x] 4.2 Pin: `TestEachLinkNamesTheWeekItLeadsTo`, asserting the prev link's text contains
      `Week 1` and the next link's contains `Week 3`. Break by swapping the two, watch it fail,
      restore.
- [x] 4.3 Pin: `TestTheWeekBarIsIdenticalWhateverTheScore`: render the same three-week tree with
      two different stat sets (ours leading, theirs leading) and assert the `<nav>` markup is
      byte-identical. Break it by temporarily adding `{{if gt ...}}` on a total, watch it fail,
      restore.

## 5. The bar's layout

- [x] 5.1 RED/GREEN, one row at a time in `TestTheStyleSheetDeclares`. For each, change the test
      table, watch it fail, then change the CSS:
      - `.weeks` declares `display: grid`
      - `.weeks` declares `grid-template-columns: 1fr auto 1fr`
      - `.weeks` declares `align-items: baseline`
      - `.weeks` declares `margin-bottom: 1rem`, so the bar stands apart from the cards
      - `.weeks .next` declares `text-align: right`
      - `.weeks h1` declares `margin: 0` and the size chosen to sit below nothing larger than the
        card headings (settle the value while doing 5.3)
- [x] 5.2 Extend the layout harness's measuring script and `layout` struct with the rects of
      `.weeks`, its `h1`, and each `a[rel]`. Confirm existing layout tests still run and pass.
- [x] 5.3 RED (rendered): `TestTheLabelDoesNotMoveWhenALinkIsAbsent`. Render `/2026/2` over a tree
      with weeks 1–3, then over a tree with weeks 2–3. Assert the `h1` left edges are within 0.5px.
      If 5.1 already makes it pass, record it as a pin: break it by temporarily changing the
      columns to `auto auto auto`, watch it fail, restore.
- [x] 5.4 Rendered pin: `TestAPhoneFitsTheWeekBarOnOneLine`, with the narrow fixture CSS (390px,
      applied to `.weeks` as well as `.matchup`), both links present. Assert the three rects'
      vertical ranges overlap and their horizontal ranges do not, that each rect lies inside the
      bar, and that neither link is taller than the label (a link's text is no larger than the
      label's, so a taller link has wrapped). Break it by temporarily giving `.weeks h1` a large
      `min-width`, watch it fail, restore. Overlap alone does not catch the break: the links wrap
      and overflow the bar while still sharing a line.
- [x] 5.5 Confirm the existing rendered tests (`TestNothingSitsAboveTheHeading`, the band and
      alignment tests) still pass with the bar present.

## 6. What only a real phone can tell us

- [x] 6.1 On a real iPhone in portrait, open `/2026/1` and `/2026/2`. Check that the label reads
      clearly, the links are easy to tap, the label does not jump between the two pages, and a
      refresh after navigating stays on the navigated week.

## 7. Close-out

- [x] 7.1 `go test ./...` (with Chrome present, layout tests running) and `go vet ./...` are clean.
- [x] 7.2 `git diff --stat main` touches only `internal/lineup/matchup.go`,
      `internal/lineup/matchup_test.go`, `internal/web/matchup.go`, `internal/web/matchup.html`,
      `internal/web/matchup_test.go`, `internal/web/layout_test.go`, and this change's directory.
- [x] 7.3 `openspec validate add-week-navigation --strict` is clean.
