Each behaviour below is one red-green pair. A RED task is done only after the test has run and
failed **for the expected behavioural reason**: a wrong value or position, not a build error or a
missing symbol. Where a test passes as soon as it is written, the task says so: keep it as a
regression pin, break the production code on purpose to watch it fail, then restore.

CSS rows go into `TestTheStyleSheetDeclares` one at a time. Rendered-layout tests (section 1) must
be seen **running, not skipping**, before any RED/GREEN in them counts.

## 0. Before starting

- [x] 0.1 Confirm `go test ./...` passes on this branch before touching anything.
- [x] 0.2 Confirm `refresh-page-while-visible` still modifies only *The page implies no winner* in
      `matchup-page`, and none of this change's three requirement headers.

## 1. A harness that measures the rendered page

- [x] 1.1 In `internal/web/layout_test.go`, add `chromePath()` (per design decision 6: env var, the
      macOS app path, then `PATH`) and `renderedLayout(t, body)`. It skips under `testing.Short()`
      or with no Chrome found, strips the refresh `<script>`, appends the measuring script, runs
      `--headless --disable-gpu --dump-dom` against a temp file with a timeout, and decodes the JSON
      from `<pre id="layout">`. For each column the layout holds the rects of `header` (may be
      absent), `h2`, `.total`, the first `li`, and the column. Confirm it builds.
- [x] 1.2 Add `TestTheHarnessMeasuresTheStarterList` as a smoke test: the first `li` of each column
      has a non-zero height and the two columns' first `li` tops are equal. Run it and confirm it
      **runs** (not skipped) and passes on the current code. Try to break it by temporarily adding
      `.column:first-child ol { grid-row: 3 / span 9 }` to the fixture page's CSS (not the
      template). Expect it to **still pass**: `.column`'s `container-type` stops the cards being
      subgrids (design decision 7), so moving the `ol` changes nothing. Record that and restore;
      the break check is redone in 1.6.
- [x] 1.3 RED (rendered): add the second `li` rect to the measuring script and layout struct, and
      confirm it builds. Then add `TestStarterRowsLineUpAcrossCards`: a `weekFS` fixture with the
      fixture page's `.matchup { max-width: 390px }`, where one roster's first starter has a name
      whose **short** form wraps at that width (the short form shows below 12.5rem) and the other
      roster's does not. As a fixture precondition, assert the first `.player` of the wrapping card
      is taller (add a first `.player` rect; not the `li`, which a working subgrid stretches to the
      shared row height). Then assert both columns' second `li` tops are equal. Run it and confirm
      it fails on unequal second `li` tops.
- [x] 1.4 RED/GREEN (CSS row): `.column li` declares `container-type: inline-size`. Change the test
      table, watch it fail, add the declaration, re-run. 1.3 still fails.
- [x] 1.5 RED/GREEN: retire the `.column` `container-type: inline-size` row ("a card is a container
      its name form can be chosen by") and its declaration together; update the row's name for the
      `li` row if it still reads as the card. Re-run 1.3 and confirm it passes. Confirm the name-form
      tests (long/short at narrow and wide fixtures) still pass.
- [x] 1.6 Redo the 1.2 break check: add `.column:first-child ol { grid-row: 3 / span 9 }` to the
      fixture page's CSS, watch `TestTheHarnessMeasuresTheStarterList` fail on unequal first `li`
      tops, restore.
- [x] 1.7 Run `go test -short ./internal/web` and confirm the layout tests report as skipped with
      the reason, and everything else passes.

## 2. The total sits beside the team name, above the starters

- [x] 2.1 RED (rendered): add `TestTheTotalSitsOnTheTeamNameLine`. For each column, the `.total`
      rect vertically overlaps the `h2` rect, its left is at or right of the `h2`'s right, and both
      bottoms are at or above the first `li` top. Run it and confirm it fails because the total is
      below the starters (the bottom row) and does not overlap the `h2`. With the cards now real
      subgrids (1.5), this is where the old placement was meant to put it.
- [x] 2.2 RED (markup): add `TestTheTeamNameAndTotalShareAHeading`, asserting each
      `<section class="column">` contains a `<header>` whose content includes both the `h2` with
      the team and the `class="total"` element. Run it and confirm it fails with no header found.
- [x] 2.3 GREEN (markup): in `matchup.html`, wrap `<h2>` and `<p class="total">` in `<header>`, and
      remove the total from after the `ol`. Re-run 2.2 and confirm it passes. Confirm
      `TestEachColumnTotalsItsStarters`, `TestThePageShowsNoMargin`, and
      `TestTheTwoColumnsCarryTheSameMarkup` still pass. Confirm 2.1 **still fails**, now because the
      total sits under the name inside a block header (overlap check), not in the bottom row.
- [x] 2.4 RED/GREEN, one CSS row at a time. For each, change the test table, watch it fail, change
      the CSS, re-run:
      - `.column header` declares `grid-row: 1`
      - `.column header` declares `display: grid`
      - `.column header` declares `grid-template-columns: minmax(0, 1fr) auto`
      - `.column header` declares `align-items: baseline`
      - `.column header` declares `column-gap: 0.5rem`
- [x] 2.5 Retire the `.total` `grid-row: -2 / -1` test row and declaration together. This comes
      before the 2.1 re-run: inside the header grid that placement puts the total on its own
      implicit row in the name column, so 2.1 cannot pass while it remains.
- [x] 2.6 Re-run 2.1 and confirm it passes.
- [x] 2.7 RED/GREEN: the `.matchup` row becomes `grid-template-rows: auto repeat(9, auto)`. Change
      the test row, watch it fail, then change the CSS and the comment above `.matchup`.
- [x] 2.8 Remove `.total`'s `text-align: right` and `padding-top`. Re-run 2.1 and confirm it still
      passes.

## 3. No sum line; nothing above the heading

- [x] 3.1 RED: add `TestTheTotalHasNoSumLine`, asserting the `.total` block declares no `border`
      (any `border` property). Run it and confirm it fails on `border-top: 1px solid #ccc`.
- [x] 3.2 GREEN: remove `.total`'s `border-top`. Re-run and confirm it passes.
- [x] 3.3 RED (rendered, regression pin): add `TestNothingSitsAboveTheHeading`. In each column, the
      `header` top is at or above every measured element's top. It passes now, so record it as a
      pin. Break it by temporarily setting `.column header { grid-row: 3 }`, watch it fail, restore.
      (`grid-row: 2` does not break it: the header then shares row 2 with the `ol`, so its top
      equals the first `li`'s and "at or above" still holds.)

## 4. The heading is a band

- [x] 4.1 RED/GREEN, one row at a time:
      - `.column header` declares `background: #f3f3f3`
      - `.column header` declares `border-bottom: 1px solid #ccc`
      - `.column header` declares `border-radius: 5px 5px 0 0`
      - `.column header` declares `padding: 0.5rem 1rem`
      - `.column header` declares `margin-inline: -1rem`
- [x] 4.2 RED/GREEN: the wide card padding row becomes `.column` `padding: 0 1rem 0.5rem`.
- [x] 4.3 RED/GREEN, in the 33rem block, one row at a time:
      - `.column` `padding: 0 0.5rem 0.5rem`
      - `.column header` `margin-inline: -0.5rem`
      - `.column header` `padding-inline: 0.5rem`
- [x] 4.4 Add a CSS comment on `.column header` saying that its negative inline margin must equal
      the card's inline padding at each width, and why the band has a radius instead of the card
      using `overflow: hidden`.
- [x] 4.5 RED (rendered): add `TestTheBandSpansTheCard`. Each column's `header` left and right are
      within 1px of the column's inner border edges (column rect inset by 1px). Before 4.1–4.3 it
      would have failed. If it passes as soon as it is written, record it as a pin: break it by
      temporarily removing `margin-inline: -1rem`, watch it fail, restore.
- [x] 4.6 Regression pin: extend `TestTheTwoColumnsCarryTheSameMarkup` (or add a sibling test) so
      each `<header>` carries no `class` or `style` attribute. It passes, so break it by temporarily
      adding `class="lead"` to the first column's header, watch it fail, restore. (Done as a sibling test, `TestTheHeadingsCarryNoClassOrStyle`; the break used `{{range $i, $c := .Columns}}` so only the first header got the class.)

## 5. Type, and a long team name

- [x] 5.1 RED/GREEN, one row at a time:
      - `.column h2` `font-size: 1.125rem`
      - `.column h2` `letter-spacing: 0.04em`
      - `.column h2` `margin: 0`
      - `.column h2` `text-transform: uppercase`
      - `.column h2` `overflow-wrap: break-word`
      - `.total` `font-size: 1.5rem`
      - `.total` `font-variant-numeric: tabular-nums`
      - `.total` `font-weight: 700`
      - `.total` `margin: 0`
      - `.total` `white-space: nowrap`
- [x] 5.2 Retire the narrow `.column h2` `margin: 0.25rem 0` row and declaration together. Confirm
      `TestBothTeamsAppearWithUsFirst` still finds lowercase team names in the DOM.
- [x] 5.3 RED/GREEN, one row at a time:
      - `.column header` declares `container-type: inline-size` (design decision 7)
      - `@container (max-width: 12.5rem)` block `.total` declares `font-size: 1.25rem`
- [x] 5.4 RED (rendered): add `TestAWrappedTeamNameKeepsTheHeadingsAligned`. Use a `weekFS` fixture
      whose opponent file stem is long enough to wrap at the narrow fixture width (e.g.
      `extraordinarilylongteamname`), and set `.matchup { max-width: 390px }` in the fixture page per
      design decision 6. Assert:
      - both `header` rects have equal top and height,
      - both `.total` tops are equal,
      - each `.total` right is within its column,
      - both first `li` tops are equal,
      - the long name's `h2` is taller than the other.

      If it passes as soon as it is written, record it as a pin. Break it by temporarily removing
      `overflow-wrap: break-word` from `.column h2` and `minmax(0, 1fr)` from the header (so the
      total is pushed out), watch it fail, restore. (It passed as written. The h2-height precondition is
      `t.Errorf`, not `t.Fatalf`, so the break reports the total pushed out, not just equal h2 heights.)

## 6. What only a real phone can tell us (required)

- [x] 6.1 On a real iPhone in portrait (deployed app or served over the LAN), check that:
      (Confirmed by the user 2026-09-14: band, line, corners, equal bands with level totals, and
      starter rows. The `112.5` no-touch check was skipped.)
      - each card opens with a grey band holding `BOJJAES` at the left and the total at the right,
        on one line, with a grey line under the band and no line above the total,
      - the band's corners follow the card's rounded top with no gap or square corner,
      - `BOJJAES` and a three-digit total such as `112.5` do not touch (use a local fixture week if
        no real week scores that high),
      - both bands are the same height and both totals share a baseline,
      - starter rows still line up across the cards.
- [ ] 6.2 Check desktop Safari and desktop Chrome at a wide window for the same points.
- [ ] 6.3 If anything differs only in Safari, record it in design.md Risks before changing the CSS,
      and change the CSS and its test row together.

## 7. Close-out

- [x] 7.1 `go test ./...` (with Chrome present, layout tests running) and `go vet ./...` are clean.
- [x] 7.2 `git diff --stat main` touches only `internal/web/matchup.html`,
      `internal/web/matchup_test.go`, `internal/web/layout_test.go`, and this change's directory.
- [x] 7.3 `openspec validate card-header-team-and-total --strict` is clean.
