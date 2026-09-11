## Why

On a portrait phone the matchup page gives each player name about 114–139px, so long names such as
"Amon-Ra St. Brown" wrap and break the row. The page should show a short name on narrow cards and
the full name otherwise. Before the page can choose, the lineup needs to provide a short name.

This change covers only providing it. Choosing which name to render belongs to
`fit-matchup-page-to-phone-width`, which relies on the field this change adds.

## What Changes

- `lineup.Record` gains `ShortName string`, which, like `Name`, is display text only.
- Reading a lineup (`Tree.Read`, then `Lineup.Starters()` / `Lineup.Bench()`) fills `ShortName`
  on every record. The value is derived in Go from `Name`; it is not a new CSV field:
  - first initial, a space, then the rest of the name with no period after the initial:
    `Amon-Ra St. Brown` → `A St. Brown`, `Josh Allen` → `J Allen`
  - a first name written as initials (a first word with a period, or exactly two capital letters)
    is kept as written: `A.J. Brown` → `A.J. Brown`, `KC Concepcion` → `KC Concepcion`
  - a trailing generational suffix (`Jr.`, `Sr.`, `II`, `III`, `IV`, `V`, with or without a
    period) is dropped from the short form only, and only when the name has three or more words:
    `Will McDonald IV` → `W McDonald`
  - an empty name gives an empty short name, and a one-word name stays as it is
  - when two starters would get the same short name, both use their full `Name` instead
    (`Jameson Williams` and `Javonte Williams` in `2026/1/fuego.csv`). Bench records are checked
    against the other bench records the same way, and never against the starters.
- The CSV format, `Name`, `scripts/scores.sh`, and everything in `internal/web` stay as they are.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `lineup-source`: adds a requirement that every record read from a roster carries a short name
  derived from its name, including the rule for collisions among the starters and among the bench.
  The "id is authoritative and the name is a label" requirement is amended so the short name is
  covered by the same display-only, no-placeholder rules as the name.

## Impact

- `internal/lineup`: `Record` gets a new field. A new unexported derivation, plus a collision pass
  run once over the starters and once over the bench, is called from `Tree.Read`. `parse` stays
  unchanged.
- Existing tests that compare whole `Record` values returned by `Tree.Read` need a `ShortName` in
  their expected values. `parse` tests are unaffected.
- `internal/web` compiles unchanged: it builds starters from `rec.Name` and never uses a `Record`
  literal. The other change adopts `ShortName`.
- No new dependencies. The roster CSV and the embedded lineup tree do not change.
