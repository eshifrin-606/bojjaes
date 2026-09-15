## Context

`internal/web/matchup.html` lays out two `.column` cards as subgrids of `.matchup`, which declares
`grid-template-rows: auto repeat(9, auto) auto`: one heading row, nine starter rows, and one total
row (`separate-starter-rows`, design decision 4). Each card's `h2` is auto-placed, the `ol` sits at
`2 / span 9`, and `.total` sits at `-2 / -1` with a `border-top` sum line.

On the deployed page the total renders in row 1, above the team name. The cause is not the negative
line numbers themselves. `.column` also declares `container-type: inline-size` (for the name-form
container query), and that applies layout containment, which stops an element being a subgrid:
in headless Chrome each card computes `grid-template-rows: none`. So each card is its own grid
with no explicit rows, `.total`'s `grid-row: -2 / -1` resolves to the card's top, and starter rows
line up across cards only by coincidence (equal content). With `container-type` removed, the cards
compute `subgrid` and the total goes to row 11. The test suite did not notice.
`TestTheStyleSheetDeclares` pins that `subgrid` and `grid-row: -2 / -1` are declared, not that the
browser honours them, and the manual iPhone check (`separate-starter-rows` 7.1) was run before the
regression was visible, or missed it.

Constraints that still apply:

- *The page implies no winner*: nothing may be styled by score or by which column leads.
- The `@container (max-width: 12.5rem)` name-form threshold depends on starter-row width only. A
  heading change must not add inline padding inside `li`.
- No subgridded element (`.column`, `.column ol`) may be a size container (decision 7).
- Team names are lineup file stems: short, lowercase, and from a small set (`bojjaes`, `wood`,
  `aroma`, `fuego`, `gonads`). Nothing stops a longer one appearing.
- Target: iPhone Safari on iOS 16 or later, and desktop Chrome/Safari. There are no Go module
  dependencies today. CI (`fly-deploy.yml`) deploys and does not run `go test`.

## Goals / Non-Goals

**Goals:**

- The name and total share one heading line at the top of each card, and headings share a height
  across cards.
- The heading reads as a header band (option C from exploration) without leader styling.
- The bottom total row, its sum line, and the negative-line placement are gone.
- A test that measures real layout, so placement regressions fail locally.

**Non-Goals:**

- Page title, week navigation, team colours.
- Running the layout test in CI.
- Any visible change to starter rows or name forms. Moving the size container (decision 7) keeps
  the name-form breakpoint where it is.

## Decisions

### 1. Markup: a `<header>` holding the `h2` and the total

```html
<section class="column">
  <header>
    <h2>{{.Team}}</h2>
    <p class="total">{{.Total}}</p>
  </header>
  <ol>…</ol>
</section>
```

The total keeps its `total` class and its `<p>`, so `renderedTotals` and every totals test keep
working unchanged. `<header>` inside a `<section>` is sectioning-scoped, so there is no page-level
banner landmark.

- **`<header>` wrapper (pick):** one grid item in row 1 that lays out its own two columns. The
  heading's internal layout is independent of the subgrid.
- **`h2` and `.total` as separate subgrid items, both in row 1:** that needs a second column track
  on the card, which the starter `ol` would then have to span. More coupling, for no gain.
- **Total inside the `h2`:** it would make the score part of the heading's accessible name, and
  `renderedTotals` would have to change.

### 2. Placement: explicit positive rows; the total row track is removed

`.matchup { grid-template-rows: auto repeat(9, auto) }`, `.column header { grid-row: 1 }`, and
`.column ol { grid-row: 2 / span 9 }` as today. `.column` stays `grid-row: 1 / -1`, which
resolves against the parent's explicit grid, where it has always worked. Nothing inside the subgrid
uses a negative line number.

The header gets an explicit `grid-row: 1`, not auto-placement. Auto-placement happened to work for
the `h2`, but the bug shows that implicit placement in this subgrid is not something to lean on.

### 3. Heading layout: a two-column grid, baseline-aligned

```css
.column header {
  align-items: baseline;
  column-gap: 0.5rem;
  display: grid;
  grid-row: 1;
  grid-template-columns: minmax(0, 1fr) auto;
}
```

This mirrors the starter row (`minmax(0, 1fr) auto`): the name column can shrink below its longest
word and wraps (`overflow-wrap: break-word` on the `h2`), and the total never gives way.
`align-items: baseline` puts the smaller-capitals name and the larger total on one baseline, and it
keeps the total on the name's first line when the name wraps. Across cards, the subgrid gives both
headers the same row height, and first-baseline alignment puts both totals at the same height.

### 4. The band: full-bleed background via negative inline margin

```css
.column { padding: 0 1rem 0.5rem; }
.column header {
  background: #f3f3f3;
  border-bottom: 1px solid #ccc;
  border-radius: 5px 5px 0 0;
  margin-inline: -1rem;
  padding: 0.5rem 1rem;
}
@media (max-width: 33rem) {
  .column { padding: 0 0.5rem 0.5rem; }
  .column header { margin-inline: -0.5rem; padding-inline: 0.5rem; }
}
```

- **Negative margin (pick):** the card's padding stays the single source of starter-row inset, so the
  12.5rem container threshold is untouched. The pairs `1rem`/`-1rem` and `0.5rem`/`-0.5rem` must
  match. A test row pins each, and a comment in the CSS says why.
- **Move card padding onto children:** `ol` would need `padding-inline`, which shrinks the
  container's content box differently. It would re-derive the threshold arithmetic for a header
  change.
- **`overflow: hidden` on the card to clip the band corners:** it would also silently clip anything
  that ever overflows, like a long name, and the spec forbids hiding names. So instead the band gets
  `border-radius: 5px 5px 0 0`: the card's 6px radius minus its 1px border, which is the inner
  radius.

Colours: `#f3f3f3` is lighter than the card border `#ccc` and the row divider `#eee` stays inside
the list. Black text on `#f3f3f3` is about 18:1. The band's `border-bottom: 1px solid #ccc` matches
the card border, so the header reads as part of the card frame, not as another starter divider.

### 5. Type

- `h2`: `font-size: 1.125rem`, `letter-spacing: 0.04em`, `margin: 0`, `text-transform: uppercase`,
  `overflow-wrap: break-word`. Uppercase comes from CSS, so the DOM text stays `bojjaes` and tests
  that look for team names are unaffected.
- `.total`: `font-size: 1.5rem`, `font-variant-numeric: tabular-nums`, `font-weight: 700`,
  `margin: 0`, `white-space: nowrap`. The old `text-align: right`, `padding-top`, `border-top`, and
  `grid-row: -2 / -1` are retired. The `auto` track puts it at the right.
- In a narrow card, `@container (max-width: 12.5rem) { .total { font-size: 1.25rem } }`. In headless
  Chrome at 390px, `BOJJAES 112.5` fit with only a few px to spare. Safari's system font is a bit
  wider, and the container threshold is the existing "this card is tight" signal. The query's
  container is `.column header` (decision 7).
- The existing narrow `.column h2 { margin: 0.25rem 0 }` row is retired. The band's padding owns
  heading spacing now.

### 6. Rendered-layout test: run the Chrome binary, no Go dependency

`internal/web/layout_test.go` adds a helper, `renderedLayout(t, body) layout`:

1. It finds Chrome from `$BOJJAES_CHROME`, then
   `/Applications/Google Chrome.app/Contents/MacOS/Google Chrome`, then `chromium`, `google-chrome`,
   or `chrome-headless-shell` on `PATH`. If none is found, `t.Skip("no Chrome: …")`. It also skips
   under `testing.Short()`.
2. It writes the handler's HTML to `t.TempDir()`, with the refresh `<script>` stripped and a
   measuring script appended. That script collects `getBoundingClientRect()` for each column's
   `header`, `h2`, `.total`, and first and second `li` (the second from task 1.3), plus the column
   box, and writes the result as JSON into a `<pre id="layout">` element.
3. It runs `chrome --headless --disable-gpu --dump-dom file://…` with a timeout, then extracts and
   decodes the `<pre>`.

- **Exec the binary (pick):** zero module dependencies, as the repo has today, and it is the same
  mechanism used during exploration. `--dump-dom` runs inline scripts before serializing.
- **chromedp:** proper device emulation (a real 390px viewport, DPR). But it adds the repo's first
  third-party module and a CDP client to maintain for a handful of tests.
- **Playwright:** a Node toolchain in a Go repo. Out of proportion.

**Viewport limitation.** Headless Chrome clamps `--window-size` to a minimum of about 500px, so the
harness cannot show a true 390px phone. Placement tests (heading above starters, total beside
name, headings aligned, totals aligned) do not depend on width and run at the default viewport.
For narrow-card behaviour, the fixture page sets `.matchup { max-width: 390px }`, which is enough
for the container query. The `@media (max-width: 33rem)` spacing still does not apply, so the
390px fit scenario stays a manual iPhone check (tasks section 6). If the clamp turns out to be
avoidable (for example with `chrome-headless-shell`), promote that check into the harness.

### 7. Size containers move off the subgridded card

`.column` drops `container-type: inline-size`. `.column li` declares it instead, so the existing
`@container (max-width: 12.5rem)` long/short name rule queries the starter row. A `li` has no
inline padding, so its width equals the card's content width and the 12.5rem breakpoint keeps its
meaning. `.column header` also declares it, for the narrow `.total` font-size (decision 5). Its
content box also equals the card's content width, because its `padding-inline` cancels its negative
`margin-inline` at both widths (decision 4). Neither `li` nor `header` is a subgrid, so the
containment they get is harmless.

- **Container on `li` and `header` (pick):** no markup change, and each query keeps measuring the
  card's content width.
- **An extra wrapper `div` inside the card as the container:** more markup, and it has to sit
  outside the subgrid chain or break it again. It also conflicts with the tests that pin identical,
  minimal column markup.
- **A viewport `@media` query instead of a container query:** the fit-to-phone design chose card
  width on purpose, so the name form does not depend on the device or orientation.

## Risks / Trade-offs

- [Something reintroduces a size container on `.column` or `.column ol`] → subgrid silently turns
  off again. `TestStarterRowsLineUpAcrossCards` (task 1.3) measures alignment in the browser, so the
  regression fails locally. Safari applies the same rule (layout containment prevents subgrid), so
  this is not a Chrome-only concern.
- [The `header` container's width drifts from the card's content width] → only if the band's
  margin and padding stop cancelling, which decision 4's test rows already pin.

- [The layout test skips everywhere without Chrome, including CI] → CI runs no tests today. The
  test names the skip reason, and `tasks.md` requires seeing it run (not skip) locally before this
  change is done.
- [Negative-margin band and card padding drift apart] → one test row per pair, plus a CSS comment
  naming the coupling.
- [Uppercase plus letter-spacing widens team names, and a future long team name wraps] → the
  heading's `minmax(0, 1fr)` column and `overflow-wrap` keep it inside the card, and the subgrid
  keeps rows aligned. Covered by a rendered-layout test with a long fixture team name.
- [Totals side by side at the top make comparing them easier] → accepted by the user during
  exploration. Styling stays identical in both cards, and no difference is computed.
- [Safari resolves something differently from Chrome] → the harness is Chrome-only, so the manual
  iPhone check in tasks is still required. Any Safari-only difference goes in Risks here before
  the CSS changes.

## Migration Plan

Template and CSS only. Merge to `main` and it deploys through `fly-deploy.yml`. To roll back,
revert the commit. No data or cache changes.

## Open Questions

- Whether `chrome-headless-shell` or `--headless=old` allows a 390px viewport, which would let the
  harness check the phone-fit scenario directly. This isn't a blocker.
