## Context

`internal/lineup` parses a roster into `[]Record` in file order, and `Lineup` splits it at
`StarterCount = 9` into `Starters()` and `Bench()`. Position is a label only, and slot is implied by
file order (ADR 0004 #9). Of the 22 files in `internal/data/lineups`, every one's first nine rows
are a legal mix (1 QB, 1 K, 2 D, and 5 RB/WR/TE with at least one of each), in about 20 different
orders. Four have more than nine rows: `2025/15/bojjaes` (10), `2025/16/bojjaes` (11),
`2026/1/bojjaes` (24), and `2026/1/fuego` (11).

## Goals / Non-Goals

**Goals:** explicit, deterministic slots on read; legal-mix validation; exactly nine records;
render back to CSV in slot order; matchup page in slot order.

**Non-Goals:** lineup form, write path, current-week source, home page, roster transactions,
showing slot labels on the page.

## Decisions

1. **Slots are a fixed table in `internal/lineup`.** `type Slot string` with `QB RB WR TE FLEX K D`,
   and `var slotOrder = [9]Slot{QB, RB, WR, TE, FLEX, FLEX, K, D, D}`. Each slot gets an "accepts
   position" set, and FLEX's set is RB ∪ WR ∪ TE (so it includes FB). `StarterCount` is either
   kept and set to `len(slotOrder)`, or removed if nothing outside the package needs it. A table
   over a switch per slot, because the render path and a later form will need the same order.

2. **Assignment is a fixed sequence of passes over file order.** First QB, then RB, WR, TE, K: each
   takes the first unclaimed record it accepts. Then FLEX×2 and D×2 take the unclaimed records they
   accept, in file order. Ties are therefore always broken by file order, so the result is
   deterministic. A general matching was rejected. FLEX accepts everything RB, WR, and TE accept,
   so the greedy passes fill every slot whenever any assignment could, and the user fixed the rule.

3. **Validation is where `Lineup` is built.** `newLineup(records) (Lineup, error)` checks
   `len == 9` first (the error states the count), then assigns slots. With exactly nine records an
   unplaced record always leaves some slot empty, so the error names the first empty slot in
   canonical order. `parse` stays unchanged and keeps returning records in file order. `Tree.Read`
   wraps the error with the file name, as it does for parse errors today.

4. **`Lineup` stores the slotted records.** It stores `starters [9]Record` in slot order.
   `Starters()` returns them as a slice. `Bench()` and `benchStart` are deleted, with no
   replacement.

5. **Render is `func (l Lineup) WriteCSV(w io.Writer) error`.** It writes the header
   `id,name,position,team` and nine records in slot order, with positions as read (an FB stays FB).
   It writes no short name and no comments. It writes to an `io.Writer` rather than returning
   `[]Record`, because the future write path needs the bytes and slot-ordered records are already
   available through `Starters()`. A name containing a comma cannot occur, since parse refuses it.

6. **The matchup page needs no code change.** `scoreColumn` already ranges over `Starters()`, which
   now returns slot order. A web test pins the ordering.

7. **Test fixtures become legal lineups in slot order before enforcement lands.** The web fixtures
   (`ourLine` has three Ks; `theirLine` has two QBs and no K or D) and `lineup/testdata`
   (`shortnames.csv` 12 rows, `wood.csv` 3) are relabelled or trimmed so that file order already
   equals slot order. Index-based assertions then keep their meaning, and later RED steps fail only
   on the behaviour under test.

## Risks / Trade-offs

- [The phone store on the volume could hold an over-9-row week that git no longer has] → Seeding
  replaces a week whole whenever the git copy changes, so the trimmed files overwrite it. A week
  git deleted is not touched, and none of the four trimmed weeks is deleted.
- [Reordering on display surprises a reader used to the file's order] → Accepted by the user. The
  files themselves are unchanged apart from the trim.

## Migration Plan

Trim the four files to their first nine records, which are already legal, in the same change and
before enforcement. Drop their `# bench` comment and the rows after it. Rollback is a revert.
