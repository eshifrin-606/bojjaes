## Context

`internal/web/matchup.html` puts two `.column` cards in a `1fr 1fr` grid. Each card is a flex
column holding an `h2`, an `ol` of flex `<li>` rows (`.player` and `.points`), and a `.total` pushed
to the bottom with `margin-top: auto`. Each card lays out its rows by itself, so only the totals line
up across the cards. That was an explicit non-goal of `fit-matchup-page-to-phone-width` (its design
decisions 4 and 6), and this change reverses it.

`lineup.Record` already carries `Position` and `Team` (closed sets, never empty) since
`add-lineup-position-and-team`, but the page does not render them. `ShortName` is derived per group
by `fillShortNames`, which falls back to `Name` when two records in a group would collide.

Existing constraints that still apply:

- *The page implies no winner*: nothing may be styled by score or by which column leads.
- The `@container (max-width: 12.5rem)` name-form threshold assumes the row's content width is
  name + row gap + `5ch` points box, with no inline padding inside the row.
- The target device is iPhone Safari on iOS 16 or later.
- CSS tests can only show that rules are present. How the page looks is checked by hand.

## Goals / Non-Goals

**Goals:**

- Each starter reads as one unit: name and points on line one, `POS · TEAM` muted on line two, and a
  hairline between adjacent starters.
- Row N lines up across the two cards at every width.
- The name-form threshold arithmetic is unchanged.
- A short name never grows past its derived form.

**Non-Goals:**

- Zebra stripes, per-row cards, and team colours.
- Disambiguating colliding short names with more of the first name.
- Changing the name-form threshold or the narrow-viewport spacing.

## Decisions

### 1. Separation: hairline dividers plus a muted meta line

`.column li + li { border-top: 1px solid #eee }`, a little `padding-block` on each `li`, and
`.meta { color: #666; font-size: 0.8125rem }`. The gap between players is larger than the gap
between a name and its own meta line, so grouping comes mostly from proximity. The divider confirms
it.

- **Dividers + hierarchy (pick):** no inline padding, so the 12.5rem threshold holds. They match the
  card border and the total's `border-top`. Stagger would stay unobtrusive even without decision 3.
- **Zebra stripes:** strong grouping, but a tint needs side padding to stay off the text, which
  takes name width.
- **Per-row cards:** the most width lost, and a box inside a box is cluttered at 165px of card.

`#666` on white is about 5.7:1, above WCAG AA for small text. The divider (`#eee`) is lighter than
the card border (`#ccc`), so rows never read as cards of their own.

### 2. Row markup: a two-row grid per `<li>`

```
<li>
  <span class="player"><span class="long">…</span><span class="short">…</span></span>
  <span class="points">…</span>
  <span class="meta">WR · CIN</span>
</li>

.column li { display: grid; grid-template-columns: minmax(0, 1fr) auto; column-gap: 1rem; }
.meta      { grid-column: 1; }

┌──────────────────────────┬───────┐
│ .player (wraps under)    │.points│  ← points on the name's first line
├──────────────────────────┼───────┤
│ .meta                    │       │
└──────────────────────────┴───────┘
```

- **Grid `li` (pick):** the points land on line one with no extra wrapper, and `column-gap` does the
  job the flex `gap` did, at the same widths. `.points` keeps `min-width: 5ch`, `white-space:
  nowrap`, and `text-align: right`, and an `auto` track never gives its width to the name. So
  `flex: none` has nothing left to do and its test row is retired.
- **Flex with `flex-wrap` and `.meta { flex-basis: 100% }`:** it works, but the flex `gap` then
  also applies vertically between name and meta, so it needs a split `row-gap`/`column-gap` anyway.
  A wrapped name also sits less predictably.
- **Nested `.line` wrapper keeping today's flex row:** it adds an element only to preserve a
  rule.

The meta line stays in column 1, so it never runs under the points. `OLB · JAX`, the widest
realistic value, is about 65px at 13px and fits the 112px of name room on a 390px phone, so it
does not wrap.

The template writes the separator (`{{.Position}} · {{.Team}}`), and the view model carries the two
fields separately. A test pins the rendered text.

### 3. Cross-column alignment: nested subgrid over fixed row tracks

```
.matchup  grid: columns 1fr 1fr, rows [head] auto  [s1..s9] repeat(9, auto)  [total] auto
            column-gap only; row-gap 0

.column   grid-row: 1 / -1; display: grid; grid-template-rows: subgrid
  h2      row 1
  ol      grid-row: 2 / span 9; display: grid; grid-template-rows: subgrid
    li×9  one track each
  .total  row -2   (the last track)
```

Each track is sized by the taller of the two cards' items in it, and items stretch, so dividers and
totals line up. The subgrid's own padding and border feed into the edge tracks (heading and total),
so the card padding still works.

- **Subgrid (pick):** CSS only, and the same markup in both columns. Supported in Safari 16, Chrome
  117, and Firefox 71, which covers the iOS 16 floor. Without subgrid, a browser falls back to each
  card laying out its own rows: today's stagger, and not broken.
- **One grid with rows as siblings (`display: contents` on cards):** the card border and padding
  would have to be faked per cell.
- **JS equalising heights:** ADR 0004 keeps client script to the refresh timer.
- **An HTML table of row pairs:** it breaks the "a column is a team" markup and the two-column tests.

**The 9 is hard-coded in CSS.** A subgrid cannot create implicit tracks, so the parent must declare
them. Nine is the league's starter count (`lineup.starterCount`, *The first nine records are the
starting lineup*). A column with fewer starters leaves its last tracks empty, sized by the other
column, and still lines up. Passing the count from Go into an inline style would put a layout
number in the view model for a value that changes only if the league rules do. A test row pins
`repeat(9, auto)`, and a comment in the CSS names the lineup rule.

`.column { display: flex; flex-direction: column }` and `.total { margin-top: auto }` are removed
along with their test rows. The shared tracks now align the totals.

### 4. Gaps under subgrid

A subgrid inherits the parent's `row-gap`, so the `.matchup { gap: 1rem }` would open 1rem between
every starter. `.matchup` changes to `column-gap: 1rem` (and `column-gap: 0.5rem` in the 33rem media
block). Vertical rhythm comes from `li` `padding-block`. The existing `gap` test rows are renamed
to `column-gap`.

### 5. Short names: derived per record, no collision pass

`newLineup` sets each record's `ShortName = shortName(Name)` and `fillShortNames` is deleted, along
with the comment about bench players pushing starters back. Colliding starters now both show
`J Williams`, and position/team under each tells them apart.

- **Always derived (pick):** it is the user's call. A row's height stays predictable, and the rule
  is simpler.
- **Minimal disambiguating prefix of the first name (`Jam Williams`):** considered and deferred.
  Collisions are rare, and the prefix rule has its own edge cases (a shared prefix, hyphens,
  initials).

### 6. Tests

- A per-`<li>` `metaPattern` reads `class="meta">(.*?)</span>` and unescapes it. The expected values
  come from the fixture records read back through `lineup.New(fixtureWeek()).Read`, formatted as
  `Position + " · " + Team`, so the web tests do not restate the CSV.
- CSS rows are added to `TestTheStyleSheetDeclares` one at a time. Retired rows are removed in the
  same step as the rule, and the step says so.
- A regression pin checks that the style sheet has no `nth-child`, `odd`, or `even` selectors and no
  `background` on `li`, so zebra striping cannot creep in. Break it on purpose, then restore.
- `TestTheTwoColumnsCarryTheSameMarkup` already covers identical classes and now includes `meta`.

## Risks / Trade-offs

- **[CSS presence tests cannot show that rows actually line up]** → a required manual check on an
  iPhone in portrait with a week where a long name wraps on one side only. If no fixture name wraps
  at 390px, check at a width where a long name does wrap, e.g. just above the 460px flip.
- **[The hard-coded 9 in CSS drifts from `starterCount`]** → the comment names the lineup rule and
  a test row pins it. A league with more starters is a rules change that touches the spec anyway.
- **[Two starters with the same short name, position, and team]** → accepted. It is vanishingly
  rare, and the long form shows at wider widths.
- **[Taller rows push the total below the fold on a phone]** → 9 rows × about 2.6 lines is still
  under one iPhone screen at 16px. Check by hand.
- **[Subgrid edge-track sizing with card padding behaves differently in Safari]** → covered by the
  manual iPhone check. The fallback is the unaligned layout, not a broken one.
- **[`refresh-page-while-visible` also modifies `matchup-page`]** → it modifies *The page implies no
  winner* and adds refresh requirements. This change modifies *Each column is a team…* and adds a
  new header, so either can be archived first.

## Open Questions

- None blocking. Landscape and Android remain unchecked, as in `fit-matchup-page-to-phone-width`.
