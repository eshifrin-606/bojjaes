## Why

Since `separate-starter-rows` (commit 2e7e548), each card on the deployed page opens with a stray
rule and the column total, then a gap, then the team name. The total should be at the bottom. The
cause is `container-type: inline-size` on the card: its layout containment stops the card being a
subgrid (in Chrome and Safari alike), so the card is its own grid and `.total { grid-row: -2 / -1 }`
resolves to its first row, not the page's last. Starter rows line up across cards only by
coincidence. The markup and style-sheet tests all passed, because they check that a rule is
present, not where the browser puts the element.

The top of the card needs rework anyway. The one-line fix would put the total back at the bottom,
under its sum line, a scroll away from the team it belongs to. Putting the total next to the team
name gives a scoreboard-style header that shows both scores without scrolling on a phone.

## What Changes

- Each card opens with a **heading row**: the team name at the left and the column total at the
  right, on one line. This replaces the separate total row at the bottom of the card.
- The heading is a **tinted band** across the full width of the card. The team name is set in
  letter-spaced capitals, the total is bold with tabular figures, and a 1px line separates the
  band from the first starter. Both cards are styled identically, and nothing depends on points or
  on which total is larger.
- The **sum line above the total is removed**, along with the bottom total row. The page grid drops
  its last row track, so the placement that went wrong no longer exists.
- In a **narrow card** (the same container threshold that switches to short names), the total is
  set a little smaller, so `BOJJAES 112.5` keeps some room on a 390px phone.
- The name-form **size container moves** from the card to each starter row (and the heading gets
  its own), so the cards are real subgrids again. Breakpoints and name forms look the same.
- A **rendered-layout test** loads the page in headless Chrome and checks where the heading, total,
  and first starter actually sit. It skips when no Chrome is available. This is the check that
  would have caught the regression.

Out of scope:

- A page-level header (season and week), and previous/next week navigation.
- Any change to starter rows, name forms, the as-of line, or the refresh script.
- Team colours or logos.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `matchup-page`:
  - *Each column is a team, its starters, and one total*: the team name and total share a heading
    row at the top of the column, set apart from the starters by a band and a dividing line.
  - *A starter's points hold their place in the row*: *Totals line up* now concerns totals on the
    heading row, including when one team name wraps.
  - *Starters are separate rows that line up across the columns*: the heading rows, which now hold
    the totals, share a height. There is no separate total row.
  - *The page implies no winner* is not touched. The in-flight `refresh-page-while-visible` modifies
    it, and its rule already covers the heading's styling.

## Impact

- `internal/web/matchup.html`: the card's `h2` and `.total` move into a `<header>`, and the bottom
  `<p class="total">` goes away. `<style>` gains the heading band and drops the total's row
  placement and `border-top`. `.matchup` rows become `auto repeat(9, auto)`, and the card's top
  padding moves onto the band.
- `internal/web/matchup_test.go`: rows in `TestTheStyleSheetDeclares` are updated or retired, and
  there is a markup test that the name and total share the heading. `renderedTotals` keeps working
  because the total keeps its `total` class.
- New `internal/web/layout_test.go`: a headless-Chrome harness and placement tests. There is no
  new Go module dependency, since the test runs the Chrome binary directly.
- `internal/web/matchup.go` is unchanged: the view model already carries `Team` and `Total`.
- No change to lineups, the stats cache, the provider, `cmd/server`, or deploys.
