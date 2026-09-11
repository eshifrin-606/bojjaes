## Context

`internal/web/matchup.html` puts two `.column` cards in a `1fr 1fr` grid. Each card contains:

- an `h2` with the team name,
- an `ol` of nine `<li>` flex rows (`.player` and `.points`, spread with `space-between`),
- a `.total` paragraph.

All styling is in one inline `<style>`. Tests render the page through the mux and match against
the HTML. `starterPattern` / `renderedStarters` turn each row into `name=points`.

On a phone held upright (390 CSS px), each card has little room:

```
today:  390 − 2×16 body padding − 16 grid gap = 342 → 171 per track
        171 − 2×1 border − 2×16 card padding  = 137 content
        137 − 16 row gap − 45 points box      =  76 for the name
```

The points span has no box of its own, so it shrinks when a name wraps. The flex row gives the
space to the name. `--` then breaks at its hyphen, and the points column shifts left and right from
row to row.

The parallel change `add-short-player-names` adds `lineup.Record.ShortName`. This change only reads
that field. The in-flight change `refresh-page-while-visible` changes *The page implies no winner*
in `matchup-page`. This change leaves that requirement alone.

## Goals / Non-Goals

**Goals:**

- A portrait phone (375–430px) shows both cards side by side, with short names, points in one
  straight column, `--` on one line, and totals at the same height.
- Wider layouts (landscape phone, iPad, a narrow desktop window) keep showing long names.
- Everything is done with markup and CSS. No script, and no server-side device detection.

**Non-Goals:**

- The rules for shortening a name (`add-short-player-names`).
- Stacked cards, tabs, or a separate mobile template.
- Lining up starter rows *across* the two cards when only one card has a wrapped name. Only the
  totals line up.
- Styling `--` differently from a number.

## Decisions

### 1. Choose the name form with a container query on `.column`

- **Container query (pick):** it depends on the width the card actually has, whatever the spacing
  or `max-width` around it, so a later layout change does not move the threshold. Size containers
  need Safari 16 (iOS 16), Chrome/Edge 105, or Firefox 110. The target device is iPhone, so iOS
  Safari 16 or later is the support floor that matters. Older browsers ignore the `@container`
  block and show long names, which is today's content plus the other fixes.
- **Viewport media query:** supported everywhere, but its breakpoint is tied to how much spacing
  surrounds the card. Every spacing tweak would mean redoing the arithmetic.
- **JS measuring `offsetWidth`:** exact, but ADR 0004 keeps client script to the refresh timer, and
  it can flash the wrong form before the script runs.
- **Server-side User-Agent sniffing:** it guesses the device, not the width, and gets tablets and
  split-screen wrong.

The card has `container-type: inline-size`. Containing its inline size also means a column's
contents no longer count toward its minimum width, so an unbreakable name cannot push `1fr 1fr`
apart. That helps the equal-width requirement.

### 2. Render both forms and hide one

Each row renders
`<span class="player"><span class="long">{{.Name}}</span><span class="short">{{.ShortName}}</span></span>`.
CSS alone cannot switch text unless both forms are in the markup. `display: none` also removes the
hidden form from the accessibility tree and from find-in-page, so a screen reader reads one name.

- **Both spans (pick):** there is one template and one view model, and CSS decides.
- **`data-short` attribute with `::after { content: attr(...) }`:** generated content is not
  reliably read by screen readers or selectable, and it moves roster text into an attribute context.

Both columns get the same classes, so the no-winner rule still holds.

### 3. The container threshold: short names below 12.5rem (200px) of card content

Container size queries compare against the container's **content box**. At 16px `system-ui` (SF
Pro on iOS), the estimates are:

- The longest real name, `Amon-Ra St. Brown` (17 characters), is about 140px wide.
- `5ch` is about 45px, because a tabular `0` is about 9px wide.
- The tightened row gap is 8px.

```
long name fits when:  140 name + 8 row gap + 45 points box = 193px of content
threshold:            193 + ~7px for font differences      ≈ 200px = 12.5rem
rule:                 @container (max-width: 12.5rem) → show .short, hide .long
```

Here is what that means for viewport width, using the tightened spacing from decision 5:

```
content = (viewport − 2×8 body padding − 8 gap) / 2 − 2×1 border − 2×8 card padding
        = (viewport − 24) / 2 − 18
content = 200  ⇔  viewport = 2×(200 + 18) + 24 = 460px
```

| Viewport | Card content | Name form |
|---|---|---|
| 375 (iPhone mini/SE) | 157.5 | short |
| 390 (iPhone) | 165 | short |
| 430 (iPhone Pro Max) | 185 | short |
| 440 (iPhone 16 Pro Max) | 190 | short |
| 460 | 200 | short (boundary) |
| 490 (narrow desktop window) | 215 | long |
| 768 (iPad portrait, untightened) | 326* | long |
| 844 (landscape phone, untightened) | 326* | long |

\* `body { max-width: 48rem }` caps the content at (768 − 48)/2 − 34 = 326.

With short names at 390px, the room for a name is 165 − 8 − 45 = **112px**, up from 76px. At
roughly 8.5px per character, that is about 13 characters.

- **12.5rem (pick):** about 7px of slack above the measured need. It flips at 460px, which is
  above every current phone held upright.
- **12rem (192px):** it flips at 444px, but `Amon-Ra St. Brown` needs 193px, so a 445–450px window
  would wrap the long form.
- **A `ch`-based threshold:** `ch` in a container query resolves against the container's font, which
  sounds tidy. But what varies is name width in proportional glyphs, not in zero-widths, so it adds
  nothing.

### 4. The points box never shrinks or wraps

`.points { flex: none; white-space: nowrap; min-width: 5ch; text-align: right; }`. The existing
`font-variant-numeric: tabular-nums` stays.

- **Why `5ch`:** points come in 0.5 steps, so the widest realistic value is three digits and a half,
  like `112.5`, which is 5 characters. A total like `212.5` is the same width. Negative scoring
  **does** exist (`docs/scoring.md`: interception thrown −3, fumble lost −3), but a realistic
  negative such as `-6` or `-2.5` fits in 5ch. `min-width` sets a minimum, not a maximum, so a rare
  `-12.5` makes the box one glyph wider instead of overflowing.
- **Why a fixed box:** every row in a card then gives the name the same width. Names start at the
  same x-position, and the points' right edges line up.

`.player { min-width: 0; overflow-wrap: break-word; }`. A flex item is normally at least as wide as
its longest word. With `min-width: 0` a single word longer than the name room breaks inside the
name, and the points box does not get pushed out of the card. Normal names still wrap only at
spaces and hyphens, and continue under themselves. There is no `text-overflow: ellipsis`.

- **Fixed box on a flex row (pick):** a small CSS change, and the markup shape stays the same.
- **`grid-template-columns: 1fr 5ch` per `<li>`:** does the same job, but replaces the flex rule and
  its `gap` for no gain.
- **Subgrid rows across the `ol`:** would also line up rows between the two cards, but that is a
  non-goal and it restructures the card.

### 5. Tighten the spacing with a viewport media query at 33rem (528px)

Inside `@media (max-width: 33rem)`:

| Rule | Declaration |
|---|---|
| `body` | `padding: 0.5rem` |
| `.matchup` | `gap: 0.5rem` |
| `.column` | `padding: 0.5rem` |
| `.column li` | `gap: 0.5rem` |
| `.column h2` | `margin: 0.25rem 0` |

- **Media query (pick):** a container query can only style its container's *descendants*. It cannot
  change `body` padding, the grid gap, or the card's own padding, which make up most of the space
  being recovered. Keeping all the spacing in one rule keeps it in one place.
- **Making `.matchup` a container too:** it could shrink card padding and row gap, but not body
  padding or the grid gap. The spacing would then be split across two mechanisms.

**Why 33rem:** above the breakpoint the normal spacing comes back, and long names must still fit.
With normal spacing, `content = (viewport − 48)/2 − 34`, and the longest name needs
140 + 16 row gap + 45 = 201px. That holds from 518px up. At 529px the content is 206.5px, so
nothing wraps and nothing flips back to short names. Any breakpoint below 496px would have a band
where normal spacing drops the content under 200px and names flip back to short right above the
breakpoint. 33rem clears both limits.

### 6. Totals line up with a flex-column card

`.column { display: flex; flex-direction: column; }` and `.total { margin-top: auto; }`. Grid items
already stretch to the height of their row, so both cards are the same height. `margin-top: auto`
pushes each total to the bottom.

- **Flex column (pick):** two declarations.
- **`min-height` on the list:** a guess that breaks as soon as a name wraps one more line.
- **Subgrid rows:** see decision 4.

### 7. How tests get `ShortName` without encoding the shortening rules

- **Expected short names come from the lineup package.** `lineup.New(fixtureWeek()).Read(2025, 15,
  team)` returns the records, and each starter's `ShortName` is the expected text. The web tests
  never state what `Josh Allen` becomes.
- **Guard against a vacuous pass.** A fixture helper stops the test with `t.Fatalf` if every
  fixture starter's `ShortName` equals its `Name`. Otherwise a template that printed `Name` twice
  would pass. This checks only that the fixture tells the forms apart, not how.
- **Compare unescaped text.** Rendered text is compared after `html.UnescapeString`. `html/template`
  escapes `'` and `+` as numeric entities that `html.EscapeString` does not produce, so comparing
  escaped strings would test the escaper rather than the page.
- **Escaping test.** It uses a record whose surname carries the markup (`Puka <b>Nacua</b>`) and
  first requires that the record's `ShortName` still contains `<b>`. If a later rule change drops
  it, the test fails loudly and asks for a different fixture instead of passing without checking
  anything.
- **CSS tests.** They extract the one `<style>` element, like `renderedScript` does for the script.
  A brace-depth helper returns the body of a block by its prelude (a selector or an at-rule) and
  checks declarations with whitespace normalized. These tests prove a rule is present, not how the
  page looks. The manual checks in tasks sections 5 and 10 cover the rendering.

### 8. Test helpers follow the markup

- `starterPattern` reads the long name from `class="long"`, so `renderedStarters` still returns
  `longname=points` and existing assertions stay the same.
- A separate per-`<li>` pattern reads `class="short"`, so a short name that ended up outside its
  row would not be matched.

## Risks / Trade-offs

- **[The font-width estimates come from SF Pro. Roboto (Android) and Segoe UI (Windows) differ]**
  → about 7px of slack in the threshold. The target device is iPhone, where SF Pro is what renders,
  so the arithmetic matches it. Elsewhere, if an estimate is wrong, the worst case is a long name
  wrapping onto two lines, which the spec allows, or short names starting a little wider. Manual
  check at the boundaries in tasks section 10.
- **[Without CSS (reader mode, text-only clients) both forms run together, e.g. `Josh AllenJ.
  Allen`]** → accepted. The page is for three people using a normal browser.
- **[CSS presence tests pin the rules, not the result, and a rule can be present yet overridden]**
  → there are few rules and they are close together, and manual verification is a required task,
  not an optional one.
- **[A future roster name longer than about 17 characters, or a short name longer than about 13]**
  → it wraps under itself and the points box stays in place. Nothing is cut off.
- **[Rows across the two cards can sit at different heights when only one card has a wrapped name]**
  → accepted. Only the totals are required to line up.
- **[Merge order]** → the name-form half (decisions 1–3, 7, 8) cannot compile before
  `add-short-player-names` lands, but the layout half (decisions 4–6) touches only the style
  element and does not read `ShortName`. Tasks do the layout half first and stop at a checkpoint
  (tasks section 6) until this branch includes `add-short-player-names`. Nothing stands in for
  `ShortName` before then. `refresh-page-while-visible` modifies a different requirement header, so
  either change can be archived first.
- **[Between the halves, a portrait iPhone still shows long names]** → accepted for that interim.
  Long names wrap under themselves while the points and totals stay in place, which is already
  better than today.

## Open Questions

- **Deferred: Android.** The target device is iPhone for now, so manual checks cover iOS Safari
  only. If someone starts reading the page on Android, add one Android device to the manual check,
  since Roboto's widths are not covered by the arithmetic above.

Settled:

- Negative scoring exists (−3 for an interception thrown or a fumble lost). Decision 4 keeps `5ch`
  because realistic negatives fit and `min-width` lets the box grow.
- `add-short-player-names` guarantees a non-empty `ShortName` whenever `Name` is non-empty, so a
  short form is never blank. The tasks checkpoint confirms it before the name-form half starts.
