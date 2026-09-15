Each behaviour below is one red-green pair. A RED task is done only after the test has run and
failed **for the expected behavioural reason**: a wrong or missing value, not a build error or a
missing symbol. Where a test passes as soon as it is written, the task says so. Keep it as a
regression pin, break the production code on purpose to watch it fail, then restore.

CSS tests use the existing `renderedStyle` / `cssBlock` / `declares` helpers and add rows to
`TestTheStyleSheetDeclares` one at a time. They show that a rule is present, not how the page looks,
so section 7 is required.

## 0. Before starting

- [x] 0.1 Confirm `go test ./...` passes on `main` before touching anything.
- [x] 0.2 Confirm `refresh-page-while-visible` (if not yet archived) touches none of this change's
      requirement headers: *Each column is a team, its starters, and one total*, *Starters are
      separate rows that line up across the columns*, and the two `lineup-source` headers.

## 1. Short names no longer fall back on a collision

- [x] 1.1 RED: in `internal/lineup/read_test.go`, rewrite
      `TestTreeReadResolvesShortNameCollisionsWithinEachGroup` to expect starters `Jameson Williams`,
      `Javonte Williams`, and `Caleb Williams` to read back as `J Williams`, `J Williams`, and
      `C Williams`. Run it and confirm it fails because the first two come back as full names.
- [x] 1.2 GREEN: in `newLineup`, set each record's `ShortName` to `shortName(rec.Name)` and stop
      calling `fillShortNames`. Re-run and confirm it passes. Rename the test to say what it now
      shows (e.g. `TestTreeReadKeepsCollidingShortNames`).
- [x] 1.3 RED (regression pin): add a case with two starters of different ids named `Josh Allen`,
      both expecting `J Allen`. It passes after 1.2, so record it as a pin. Break it by temporarily
      restoring the `fillShortNames` call, watch it fail, then restore.
- [x] 1.4 Rewrite `TestTreeReadShortNameCollisionFollowsTheStarterSplit` to the new scenario:
      `Josh Allen` second, `Jaylen Allen` ninth vs tenth, and both are `J Allen` either way. It
      passes as a pin. Break it the same way as 1.3.
- [x] 1.5 Delete `fillShortNames`, `TestFillShortNames`, and the `newLineup` comment about bench
      players pushing starters back. Keep `TestTreeReadDoesNotResolveShortNamesAcrossTeams` only if
      it still says something the new rule does not already cover; otherwise delete it. Confirm
      `go test ./internal/lineup` and `go vet ./...` pass.

## 2. A fixture that can tell position from team

- [x] 2.1 In `internal/web/matchup_test.go`, give `ourLine` / `theirLine` records their own
      position and team (e.g. `WR,CIN`, `RB,ATL`, `K,KC`) instead of the hard-coded `WR,FA`, all
      from the closed sets. Add a guard, like `fixtureShortNames`, that stops with `t.Fatalf` unless
      the fixture has at least two distinct positions, at least two distinct teams, and no record
      whose position equals its team. Confirm the whole package still passes.

## 3. Position and team render under the name

- [x] 3.1 Add helpers:
      - `fixtureMeta(t)` reads both fixture lineups back through `lineup.New(fixtureWeek()).Read`
        and returns each starter's `Position + " · " + Team` in page order.
      - `renderedMeta(body)` uses a per-`<li>` pattern for `class="meta">(.*?)</span>` and runs
        each match through `html.UnescapeString`.

      Confirm it builds.
- [x] 3.2 RED: add `TestEachStarterRendersItsPositionAndTeam`, asserting `renderedMeta(body)` equals
      `fixtureMeta(t)`. Run it and confirm it fails with `got []`.
- [x] 3.3 GREEN, deliberately incomplete: add `Position` and `Team` to `starter`, set them only on
      the branch where the starter played, and add `<span class="meta">{{.Position}} ·
      {{.Team}}</span>` after `.points` in the row. Re-run and confirm it passes.
- [x] 3.4 RED: add a case serving `fixtureStats("<an id>")`. Run it and confirm it fails because
      that starter's meta renders as ` · `.
- [x] 3.5 GREEN: in `scoreColumn`, copy `Position` and `Team` into the `starter` alongside `Name`
      and `ShortName`, before choosing between points and `noStats`. Re-run and confirm it passes.
- [x] 3.6 Confirm `renderedStarters`, `renderedShortNames`, `TestTheTwoColumnsCarryTheSameMarkup`,
      and `TestNoLineupTextReachesTheScript` still pass. If `starterPattern` no longer matches with
      the extra span, fix the pattern, not the markup order.

## 4. The row is a two-line grid with points on the name's line

- [x] 4.1 RED: assert that `.column li` declares `display: grid`. Run and confirm it fails.
- [x] 4.2 GREEN: change `.column li` from flex to `display: grid`. Re-run and confirm it passes.
- [x] 4.3 RED: assert that `.column li` declares `grid-template-columns: minmax(0, 1fr) auto`. Run
      and confirm it fails.
- [x] 4.4 GREEN: add it. Re-run and confirm it passes.
- [x] 4.5 RED: assert that `.meta` declares `grid-column: 1`. Run and confirm it fails.
- [x] 4.6 GREEN: add the `.meta` rule. Re-run and confirm it passes.
- [x] 4.7 RED/GREEN: rename the row gaps to `column-gap`, one table row at a time. The wide
      `.column li` row becomes `column-gap: 1rem`, and the 33rem-block `.column li` row becomes
      `column-gap: 0.5rem`. For each, change the test row, watch it fail, change the CSS, and
      re-run.
- [x] 4.8 Retire the `.points` `flex: none` row and declaration together; the `auto` track does its
      job now (design.md decision 2). Confirm the other `.points` rows still pass, and that
      `justify-content`/`gap` leftovers are gone from `.column li`.

## 5. Rows are divided; the meta line is muted

- [x] 5.1 RED: assert that `.column li + li` declares `border-top: 1px solid #eee`. Run and confirm
      it fails.
- [x] 5.2 GREEN: add the rule. Re-run and confirm it passes.
- [x] 5.3 RED: assert that `.column li` declares `padding-block: 0.25rem`. Run and confirm it fails.
- [x] 5.4 GREEN: add it. Re-run and confirm it passes. Also confirm with a test row that `.column li`
      declares no `padding:` shorthand and no `padding-inline`, so name width is unchanged.
- [x] 5.5 RED/GREEN, one row at a time: `.meta` `color: #666`, then `.meta` `font-size: 0.8125rem`.
- [x] 5.6 RED (regression pin): add `TestRowsAreNotStriped`, asserting the style sheet contains no
      `nth-child`, `:nth-of-type`, `odd`, or `even`, and that no `li` block declares a `background`.
      It passes, so record it as a pin. Break it by temporarily adding
      `li:nth-child(odd) { background: #f6f6f6 }`, watch it fail, then restore.

## 6. Row N lines up across the columns

- [x] 6.1 RED: assert that `.matchup` declares `grid-template-rows: auto repeat(9, auto) auto`. Run
      and confirm it fails.
- [x] 6.2 GREEN: add it, with a one-line comment that 9 is the league's starter count from the
      lineup rules. Re-run and confirm it passes.
- [x] 6.3 RED: assert that `.column` declares `grid-row: 1 / -1`, `display: grid`, and
      `grid-template-rows: subgrid`, one row at a time.
- [x] 6.4 GREEN, one declaration per row. In the same step as `display: grid`, remove `.column`'s
      `display: flex` and `flex-direction: column` and their two test rows.
- [x] 6.5 RED/GREEN: `.column ol` declares `grid-row: 2 / span 9`, then `display: grid`, then
      `grid-template-rows: subgrid`.
- [x] 6.6 RED/GREEN: `.total` declares `grid-row: -2 / -1`. In the same step, remove `margin-top:
      auto` and its test row.
- [x] 6.7 RED/GREEN: the `.matchup` gap rows become `column-gap: 1rem` (wide) and `column-gap:
      0.5rem` (33rem block), and `.matchup` declares no `gap:` shorthand. The subgrid would otherwise
      inherit a row gap between every starter (design.md decision 4).
- [x] 6.8 Confirm `TestEachColumnTotalsItsStarters`, `TestThePageShowsNoMargin`,
      `TestTheTwoColumnsCarryTheSameMarkup`, and `go test ./...` pass.

## 7. What only a browser can tell us (required)

- [x] 7.1 Serve the page and open a week on a real iPhone in portrait (served over the LAN or from
      the deployed app). Check that:
      - short names show, with `POS · TEAM` under each in smaller grey text,
      - points are on the name's line and share one right edge,
      - hairlines separate starters, and nothing separates a name from its own meta,
      - both cards are side by side at equal width with no horizontal scroll,
      - headings, every row's divider, and the totals sit at the same heights in both cards.
- [ ] 7.2 (Skipped at archive, 2026-09-14: not verified in a browser.) Force a one-sided wrap and confirm row N still lines up. Either use a desktop window just
      above the 460px flip where `Amon-Ra St. Brown` wraps, or temporarily give a local fixture
      week an unusually long name on one side only. Confirm the later rows and dividers still line
      up.
- [ ] 7.3 (Skipped at archive, 2026-09-14: not verified in a browser.) Open a week where one side has fewer than nine starters, if one exists (or a temporary
      local fixture), and confirm the totals still line up.
- [x] 7.4 (No Safari-only misalignment seen on phone or desktop.) If anything misaligns only in Safari, record it in design.md Risks before changing the
      CSS, and change the CSS and its test row together.

## 8. Close-out

- [x] 8.1 `go test ./...` and `go vet ./...` are clean.
- [x] 8.2 `git diff --stat main` shows no change to any lineup CSV, `cmd/server`, or `internal/statscache`.
- [x] 8.3 `openspec validate separate-starter-rows --strict` is clean.
