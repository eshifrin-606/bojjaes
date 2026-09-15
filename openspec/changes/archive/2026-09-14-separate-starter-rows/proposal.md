## Why

On the matchup page, nothing separates one starter from the next except line height. When a name
wraps, its second line sits as close to the next player as to its own first line. Lineups now carry
each player's position and team (`add-lineup-position-and-team`), and the page should show them. But
a second line under every name would make the rows run together completely unless each player
reads as one visible unit.

A column is also laid out on its own, so one wrapped name on the left pushes every later row on the
left below its partner on the right. Once every row has two lines, that stagger shows much more.

## What Changes

- Each starter row shows the player's **position and team on a second line** under the name, e.g.
  `WR · CIN`. The line is always shown, in both name forms and at every width. It is smaller and
  muted, so it reads as belonging to the name above it.
- The points stay on the **name's line**, not centered across both lines.
- Starter rows are separated by **hairline dividers**. There is no zebra striping and no per-row
  box. The styling is identical in both columns and never depends on points.
- **Row N on the left lines up with row N on the right**, whatever wraps. The two cards share one
  set of row tracks (CSS subgrid). That also lines up the team headings and the totals, so it
  replaces the flex-column trick that aligns totals today.
- **BREAKING (display text only):** a short name is **always** the initial-and-surname form, even
  when two starters on the same team would share it. The fallback to the full name on a collision
  is removed. Position and team tell such players apart, and a short name never gets longer than
  its derived form. Nothing identifies a player by short name, so no data or lookup changes.

Out of scope:

- Showing more of the first name to tell colliding players apart. This was considered and set aside.
  Collisions are rare, and position and team usually settle them.
- Zebra striping, per-row cards, and any score-dependent styling.
- Bench players, the lineup CSV format, and the position and team closed sets.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `matchup-page`:
  - *Each column is a team, its starters, and one total*: each starter line also shows position and
    team, always, below the name.
  - New requirement: starters are visibly separate rows, and row N lines up across the two columns.
  - *The page implies no winner* is not touched. The in-flight `refresh-page-while-visible` modifies
    it, and its rule already covers the new markup and styling.
- `lineup-source`:
  - *Colliding short names fall back to the full name* is removed.
  - New requirement: a record's short name depends only on its own name.

## Impact

- `internal/lineup/lineup.go`, `shortname.go`: the per-group collision pass (`fillShortNames`) goes
  away, and each record's `ShortName` comes straight from `shortName(Name)`. The collision tests in
  `read_test.go` and `shortname_test.go` are rewritten to the new rule.
- `internal/web/matchup.go`: `starter` gains `Position` and `Team`, copied in `scoreColumn` for
  every starter whether or not the provider has stats.
- `internal/web/matchup.html`: the row gets a `.meta` span. The `<style>` element gets the subgrid
  row tracks, the row grid, the dividers, and the muted meta line. The `.column` flex column and
  `.total { margin-top: auto }` are retired.
- `internal/web/matchup_test.go`: new rows in `TestTheStyleSheetDeclares`, retired rows for the
  flex-column rules, and a rendered-meta helper and test. How the page looks is still checked by
  hand on an iPhone.
- No change to the lineup CSV files, `cmd/server`, the stats cache, or the provider.
