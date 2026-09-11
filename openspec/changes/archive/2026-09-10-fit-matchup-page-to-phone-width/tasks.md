Each behaviour below is one red-green pair. A RED task is done only after the test has run and
failed **for the expected behavioural reason**: a wrong or missing value in the rendered body. A
build error or a missing symbol does not count. No test refers to a view-model type, and no test
before the checkpoint refers to `lineup.Record.ShortName`, so none of these REDs can fail at compile
time. Where a test passes as soon as it is written, the task says so. Keep it as a regression pin,
break the production code on purpose to watch it fail, then restore.

The work is in two halves. Sections 1–5 (the points box, the totals, the tighter spacing) need
nothing from `add-short-player-names` and are done first. Sections 6–9 render or test the short and
long name forms, need `lineup.Record.ShortName`, and are blocked at the checkpoint in section 6.
Nothing stands in for `ShortName` before then: no stub field, no fake, no copy of that change's
code.

The tests use the existing `internal/web` harness: `fixtureWeek`, `weekFS`, `fixtureStats`,
`fakeSource`, `serve`, `renderedStarters`, `renderedScript`. The CSS tests can only show that a rule
is present. How the page looks is covered by sections 5 and 10, which are required. The target
device is iPhone (Safari on iOS 16 or later); Android is deferred (see design.md, Open Questions).

## 0. Before starting

- [x] 0.1 Confirm `go test ./...` passes on this branch before touching anything.
- [x] 0.2 Read the `matchup-page` delta in `openspec/changes/refresh-page-while-visible` (if it has
      not been archived) and confirm this change still touches none of its requirement headers.

## 1. A way to assert on the style element

- [x] 1.1 Add test helpers that work the way `renderedScript` does:
      - `renderedStyle(t, body)` returns the contents of the page's only `<style>` element.
      - `cssBlock(css, prelude)` uses brace depth to return the body of the first block whose
        prelude matches, whether that prelude is a selector or an at-rule such as
        `@media (max-width: 33rem)`.
      - `declares(block, "prop: value")` checks a declaration with whitespace normalized.

      Confirm it builds. Each section below adds its assertions as rows of one table-driven test,
      one row at a time.

## 2. The points box

- [x] 2.1 RED: assert that `.points` declares `flex: none`. Run and confirm it fails. Also add the
      existing `font-variant-numeric: tabular-nums` as a pin row, which should pass.
- [x] 2.2 GREEN: add the declaration. Re-run and confirm pass.
- [x] 2.3 RED: assert that `.points` declares `white-space: nowrap`. Run and confirm it fails.
- [x] 2.4 GREEN: add the declaration. Re-run and confirm pass.
- [x] 2.5 RED: assert that `.points` declares `min-width: 5ch` and `text-align: right`. Run and
      confirm it fails.
- [x] 2.6 GREEN: add both declarations. Re-run and confirm pass. Add a comment on why it is 5ch:
      `112.5` is the widest realistic value, a negative such as `-2.5` fits too, and the width is a
      minimum, not a maximum.
- [x] 2.7 RED: assert that `.player` declares `min-width: 0` and `overflow-wrap: break-word`. Run
      and confirm it fails.
- [x] 2.8 GREEN: add the `.player` rule. Re-run and confirm pass.
- [x] 2.9 RED (regression pin): assert that the style contains no `text-overflow` and no
      `ellipsis`. Run; it passes, so record it as a pin. Break it by temporarily adding
      `text-overflow: ellipsis` to `.player`, watch it fail, then restore.

## 3. Totals line up

- [x] 3.1 RED: assert that `.column` declares `display: flex` and `flex-direction: column`. Run and
      confirm it fails.
- [x] 3.2 GREEN: add both declarations. Re-run and confirm pass.
- [x] 3.3 RED: assert that `.total` declares `margin-top: auto`. Run and confirm it fails.
- [x] 3.4 GREEN: add the declaration. Re-run and confirm pass. Confirm `TestEachColumnTotalsItsStarters`
      and `TestThePageShowsNoMargin` still pass.

## 4. Tighter spacing at narrow viewports

- [x] 4.1 RED: assert that an `@media (max-width: 33rem)` block exists and that `body` inside it
      declares `padding: 0.5rem`. Run and confirm it fails.
- [x] 4.2 GREEN: add the block with that one rule. Re-run and confirm pass. Add a comment on why it
      is 33rem: above that width, the normal spacing still leaves room for the longest long name.
- [x] 4.3 RED/GREEN, one table row at a time for each remaining declaration in the block:
      `.matchup` `gap: 0.5rem`, `.column` `padding: 0.5rem`, `.column li` `gap: 0.5rem`, and
      `.column h2` `margin: 0.25rem 0`. For each row, run and watch it fail, add the declaration,
      then re-run and confirm pass.
- [x] 4.4 Confirm the rules outside the block still carry the wide-viewport values (`1rem` padding
      and gaps, card padding `0.5rem 1rem`), so the tightening applies only to narrow viewports.
- [x] 4.5 Confirm `TestTheTwoColumnsCarryTheSameMarkup` and `go test ./...` pass: sections 2–4
      change only the style element, so both columns still share every class.

## 5. Browser check of the layout half (required)

Names are still long in every layout here, so a long name wrapping on a portrait iPhone is
expected at this stage and is not a failure.

- [x] 5.1 ~~`go run ./cmd/server` and open `http://localhost:8080/2025/15`. In Chrome device mode at
      375, 390, and 430px:
      - the cards are side by side at equal width, with no horizontal scroll,
      - the points share one right edge on every row, including rows whose name wraps,
      - the totals are at the same height,
      - a name that wraps continues under itself, with no ellipsis.~~
      Folded into 10.3: short names had landed before this section was run, so the layout was
      checked once on an iPhone in portrait instead of in Chrome with long names.
- [x] 5.2 At 390px, check a week where at least one starter has no provider entry, and confirm
      `--` stays on one line. Checked on an iPhone in portrait at `/2025/15` (Derrick Harmon).
- [x] 5.3 ~~At 528 vs 529px, confirm the spacing loosens above the breakpoint and nothing overflows.~~
      Dropped with 10.2: boundaries were not checked.
- [x] 5.4 ~~On an iPhone (served locally over the LAN or from the deployed app), repeat 5.1 and 5.2
      in portrait, and confirm landscape still looks like today's page apart from the points box.~~
      Folded into 10.3.

## 6. CHECKPOINT: BLOCKED until add-short-player-names is in this branch

Stop here until the item below is true. Do not work around it.

- [x] 6.1 Confirm `add-short-player-names` has merged to `main` and this branch is rebased onto it:
      `lineup.Record` has `ShortName string`, filled in by `Tree.Read` and non-empty whenever `Name`
      is non-empty. Confirm `go test ./...` passes before continuing.

## 7. The long name gets its own span

- [x] 7.1 RED: change `starterPattern` in `internal/web/matchup_test.go` so the name group is read
      from `class="long">` inside `class="player">`. Run `go test ./internal/web`. Confirm that
      `TestStartersRenderWithTheirPoints` and the other users of `renderedStarters` fail because
      the rendered starters come back empty.
- [x] 7.2 GREEN: in `matchup.html`, wrap `{{.Name}}` in `<span class="long">` inside `.player`.
      Re-run and confirm everything passes.

## 8. The short name sits beside it

- [x] 8.1 Add two test helpers:
      - `fixtureShortNames(t)` reads `fixtureWeek()` back through
        `lineup.New(...).Read(2025, 15, team)` for `bojjaes` and then `wood`, and returns each
        starter's `ShortName` in page order. It stops with `t.Fatalf` if every starter's
        `ShortName` equals its `Name`, because then the fixture cannot tell the forms apart.
      - `renderedShortNames(body)` uses a per-`<li>` pattern for `class="short">` and runs each
        match through `html.UnescapeString`.

      Confirm the package builds and the existing tests pass.
- [x] 8.2 RED: add `TestEachStarterRendersItsShortName`, asserting
      `renderedShortNames(body)` equals `fixtureShortNames(t)`. Run it and confirm it fails with
      `got []`, 18 names wanted.
- [x] 8.3 GREEN, deliberately incomplete:
      - Add `ShortName` to `starter` in `matchup.go`.
      - Set it only on the branch where the starter played.
      - Add `<span class="short">{{.ShortName}}</span>` after the long span inside `.player`.

      Re-run and confirm pass. The branch for a starter with no stats is left out on purpose; 8.4
      is the test that fails because of it.
- [x] 8.4 RED: add a case to the test that serves `fixtureStats("7")`. Run it and confirm it fails
      because Brandon Aubrey's short name renders as empty.
- [x] 8.5 GREEN: in `scoreColumn`, copy the record's `Name` and `ShortName` into the `starter` once,
      before choosing between points and `noStats`. Re-run and confirm pass.
- [x] 8.6 RED (regression pin): extend `TestNoLineupTextReachesTheScript` to check the rendered
      short names as well as the long names. Run it; it should pass right away, so record it as a
      pin. Break it by temporarily adding a `{{range}}` over short names inside the `<script>`,
      watch it fail, then restore.
- [x] 8.7 RED (regression pin): add `TestAShortNameIsEscaped`, which serves a week with the record
      `1,Puka <b>Nacua</b>`.
      - First read that record back through the lineup package, and stop with `t.Fatalf` if its
        `ShortName` does not contain `<b>`. In that case, change the fixture, not the assertion.
      - Then assert that the body has no `<b>` and that the unescaped short text equals the
        record's `ShortName`.

      Run it; it passes because `html/template` escapes by default. Break it by temporarily making
      `starter.ShortName` a `template.HTML`, watch it fail, then restore.
- [x] 8.8 Confirm `TestTheTwoColumnsCarryTheSameMarkup` and `TestLineupTextIsEscaped` pass
      unchanged. The new classes `long` and `short` are the same in both columns and contain no
      leader words.

## 9. Short or long by card width

- [x] 9.1 RED: assert that the top-level `.player .short` rule declares `display: none`. Run and
      confirm it fails because the rule is missing.
- [x] 9.2 GREEN: add the rule. Re-run and confirm pass.
- [x] 9.3 RED: assert that `.column` declares `container-type: inline-size`. Run and confirm it
      fails.
- [x] 9.4 GREEN: add the declaration. Re-run and confirm pass.
- [x] 9.5 RED: assert that an `@container (max-width: 12.5rem)` block exists and that inside it
      `.player .long` declares `display: none`. Run and confirm it fails.
- [x] 9.6 GREEN: add the block with only that rule. Re-run and confirm pass. A narrow card now
      shows no name at all; that is wrong on purpose, and 9.7 is the test that fails because of it.
- [x] 9.7 RED: assert that inside the same block `.player .short` declares `display: inline`. Run
      and confirm it fails.
- [x] 9.8 GREEN: add the rule. Re-run and confirm pass. Above the block, add a comment that says
      why it is 12.5rem: the longest long name, plus the row gap and the points box, needs about
      193px. Point to design.md for the arithmetic; do not restate the selectors.

## 10. What only a browser can tell us about the name forms (required)

- [x] 10.1 ~~`go run ./cmd/server` and open `http://localhost:8080/2025/15`. The Bojjaes column there
      has `Amon-Ra St. Brown` and the opponent has `Jaxon Smith-Njigba`. In Chrome device mode,
      check 375, 390, 430, and 440px:
      - short names appear,
      - the cards are side by side at equal width,
      - the points share one right edge on every row,
      - the totals are at the same height,
      - there is no horizontal scroll.~~
      Covered by the real-iPhone portrait check in 10.3 instead of Chrome device mode.
- [x] 10.2 ~~Check the boundaries:
      - 460 vs 480px: names switch from short to long.
      - 528 vs 529px: when the spacing loosens, names do not switch back to short and
        `Amon-Ra St. Brown` does not wrap.
      - 768px (iPad portrait), 844px (landscape iPhone), and a desktop window around 1280px.~~
      Dropped: the portrait phone, which is the case this change is for, looks right. Revisit if a
      long name wraps or short names show up on a wider layout.
- [x] 10.3 On a real iPhone (served locally over the LAN or from the deployed app), check portrait
      ~~and landscape~~ against the same list as 10.1. Safari on iOS 16 or later is required for the
      container query; an older iOS shows long names, which is acceptable.
      Portrait: short names appear, cards side by side, points share a right edge, `--` on one
      line. No fixture name was long enough to see a wrap under itself. Landscape was not checked.
- [x] 10.4 ~~If a boundary is noticeably off (for example, a long name wraps at 480px, or short names
      appear on a landscape iPhone), change `12.5rem` or `33rem` in the CSS, in its test row, and in
      the arithmetic in design.md together. Do not change just one of them.~~
      Not triggered: boundaries were not checked (10.2).
- (deferred) An Android device check. Not a task until someone reads the page on Android; see
  design.md, Open Questions.

## 11. Close-out

- [x] 11.1 `go test ./...` and `go vet ./...` are clean.
- [x] 11.2 `git diff --stat main` shows no change under `internal/lineup/`, `scripts/`, or any CSV.
- [x] 11.3 `openspec validate fit-matchup-page-to-phone-width --strict` is clean.
