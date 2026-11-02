## Why

A lineup's slot is implied by file order (ADR 0004 #9), but the 22 lineup files on hand use about
20 different orderings, so nothing in the code knows which player is the FLEX or that a file is a
legal lineup at all. The coming lineup form needs a lineup made of named slots it can fill and
write back. This change makes slots explicit on the read side and adds the render-back step, with
no UI and no write path.

## What Changes

- A starting lineup is exactly 9 slots in canonical order: QB, RB, WR, TE, FLEX, FLEX, K, D, D.
  FLEX takes any RB/WR/TE (FB counts as RB); D takes any defensive position in ADR 0004's closed set.
- Reading assigns records to slots by position, whatever the file order. File order only breaks
  ties (which RB is the RB and which is a FLEX, and the order of the two D slots).
- A file whose records cannot fill the slots exactly is refused. **BREAKING**: a file with more or
  fewer than 9 player rows is now an error.
- **BREAKING**: there is no bench any more. `Lineup.Bench()` and its tests are removed, and the four
  over-9-row files in `internal/data/lineups` are trimmed to their 9 starters.
- A lineup can be rendered back to CSV records in canonical slot order, ready for a later write path.
- The matchup page lists starters in slot order. Old files show reordered, but nothing in them
  changes except the trim.
- ADR 0004 #9's "slot is implied by file order" is amended in place.

Out of scope: lineup form, current-week source, home page, roster transactions, write path.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `lineup-source`: slots replace the positional first-nine/bench split; mix validation; exactly
  nine records; render back to CSV in slot order; bench references dropped.
- `matchup-page`: starters are shown in slot order; bench and short-column scenarios removed.

## Impact

- `internal/lineup/record.go`: slot assignment, `Bench()` removed, CSV rendering; `tree.go` surfaces
  slot errors through `Read`.
- `internal/web/matchup.go`: unchanged call to `Starters()`, which now returns slot order.
- Tests in `internal/lineup`, `internal/web`, `internal/data`; fixtures relabelled into legal lineups.
- Data: `internal/data/lineups/2025/{15,16}/bojjaes.csv`, `2026/1/{bojjaes,fuego}.csv` trimmed.
- Docs: `docs/adr/0004-web-frontend-stack.md` #9, `docs/adr/0005-...` (the "nine starters plus
  bench" line), `docs/package-dependencies.md`.
