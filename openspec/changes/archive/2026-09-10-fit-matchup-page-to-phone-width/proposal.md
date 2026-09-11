## Why

The matchup page reads well on a desktop, an iPad, and a phone held sideways, but not on a phone
held upright. That upright phone is how the page mostly gets opened on a Sunday. At about 390 CSS
px each card has roughly 139px of content width. Long names wrap, and because every starter row is
its own flexbox, the points span shrinks to make room:

- `--` breaks at the hyphen onto two lines.
- The points column drifts left and right from row to row.
- The two totals end up at different heights.

The user chose to keep the two cards side by side and make them tighter, rather than stacking them
or switching to tabs.

## What Changes

- Each starter row renders the player's name twice, once in the long form (`Record.Name`) and once
  in the short form (`Record.ShortName`). A CSS container query on the card shows one form and hides
  the other based on the card's own width, so portrait phones get short names and wider layouts get
  long ones. No JavaScript, and nothing keyed on the viewport for this choice.
- The points span becomes a fixed box: it never shrinks or wraps, it is right-aligned, and it keeps
  at least `5ch` of width. Points line up from row to row, and `--` stays on one line.
- A name that still does not fit wraps under itself. The points box stays where it is, and nothing
  is cut off with an ellipsis.
- At narrow viewports the spacing tightens: body padding, grid gap, card padding, the gap inside
  each row, and the team heading's margins all get smaller. This gives names about 114px instead
  of 78px on a 390px phone.
- The two totals line up. Each card becomes a flex column with its total pushed to the bottom, and
  the grid already stretches both cards to the same height.

Out of scope:

- How a short name is derived from a long one. That belongs to the parallel change
  `add-short-player-names`. This change treats `ShortName` as text it does not inspect.
- Anything inside `internal/lineup`, the lineup CSV format, and `scripts/`.
- Stacking the cards, tabs, a separate mobile layout, and truncating names with an ellipsis.

## Dependencies

Half of this change depends on **`add-short-player-names`**. The long/short name markup, the
container query that switches between them, and their tests read `lineup.Record.ShortName`
(non-empty whenever `Name` is non-empty), and do not compile until that field exists. The other
half does not: the points box, the lined-up totals, and the tighter narrow-viewport spacing touch
only the style element. Tasks do that half first, then stop at a checkpoint until this branch
includes `add-short-player-names`. Nothing stands in for `ShortName` in the meantime.

The target device is iPhone (Safari on iOS 16 or later). Android verification is deferred.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `matchup-page`:
  - *Each column is a team, its starters, and one total*: a starter line shows the name in its long
    or short form, chosen only by how wide the column is. The two columns stay side by side and
    equal in width at every viewport width, including a portrait phone.
  - *Roster text is escaped as HTML*: now explicitly covers the short form too.
  - New requirement: a starter's points stay in a fixed place in the row, and the two totals line
    up.
  - *The page implies no winner* is not touched here. The in-flight change
    `refresh-page-while-visible` already modifies it, and the rule already covers the new markup.

## Impact

- `internal/web/matchup.go`: `starter` gains a `ShortName` field, filled from `rec.ShortName` in
  both branches of `scoreColumn`.
- `internal/web/matchup.html`: the starter-row markup gets nested `long`/`short` spans inside
  `.player`, and the inline `<style>` gets the container query, the points box, the narrow-viewport
  spacing, and the flex-column card.
- `internal/web/matchup_test.go`: `starterPattern` / `renderedStarters` now read the long name.
  New helpers read the short names and the `<style>` element. CSS tests can only check that rules
  are present, so how the page actually looks is checked by hand at 375, 390, and 430px and on a
  real iPhone.
- No change to `cmd/server`, the JSON endpoints, the stats cache, or the lineup tree.
