Each behaviour is one red-green pair. A RED task is done only when the test has been run and has
failed **on a value mismatch or a missing error**, not on a build error. Section 2 adds compiling
stubs that are deliberately wrong, so every later RED fails for the behavioural reason. A "Guard"
is a test that already passes: break the production code on purpose, watch the test fail, then
restore it.

Out of scope: lineup form, write path, current-week source, home page, roster transactions. Do not
add slot labels to the page.

## 1. Legal fixtures and data, before any enforcement (all tests stay green)

- [x] 1.1 Relabel `ourLine` and `theirLine` in `internal/web/matchup_test.go` so each is a legal
      lineup already in slot order (QB RB WR TE FLEX FLEX K D D). Keep ids, names, teams, points,
      and stats where scoring allows. Replace the bench-player test (≈ line 594) with nothing; its
      spec scenario is removed. Run `go test ./internal/web/`: green.
- [x] 1.2 Make `internal/lineup/testdata/shortnames.csv` and `wood.csv` nine-record legal lineups in
      slot order, and keep the names `tree_test.go` asserts on. Rework that test's bench
      assertions into starter assertions. Run `go test ./internal/lineup/`: green.
- [x] 1.3 Delete `Lineup.Bench()`, `benchStart`, and every Bench test in `record_test.go` (the
      split table, "exactly nine", "position does not decide", "reordering promotes"). Fix
      `Starters()` to slice the first `StarterCount` records. Run: green.
- [x] 1.4 Scan `internal/data/lineups` for files over nine player rows. Expect exactly
      `2025/15/bojjaes`, `2025/16/bojjaes`, `2026/1/bojjaes`, and `2026/1/fuego`. Trim each to its
      first nine records, dropping `# bench` and everything after it. Run `go test ./...`: green.

## 2. Compiling stubs to fail against

- [x] 2.1 In `record.go`, add `type Slot string`, constants `QB RB WR TE FLEX K D`, and
      `slotOrder = [9]Slot{...}`. Change `newLineup` to return `(Lineup, error)`, still slicing in
      file order and returning `nil` error, and propagate the error in `Tree.Read`. Add
      `func (l Lineup) WriteCSV(w io.Writer) error` that returns `nil` and writes nothing. Run
      `go build ./... && go test ./...`: green.
- [x] 2.2 Add a test helper that builds a roster CSV from `(id, position)` pairs with dummy names
      and teams.

## 3. Records fill slots by position

- [x] 3.1 Guard: a file already in slot order (QB RB WR TE WR RB K LB CB) yields those nine in
      file order.
- [x] 3.2 RED: K LB WR WR TE RB RB QB DE → starters QB, RB₁, WR₁, TE, WR₂, RB₂, K, LB, DE. Confirm
      it fails because the starters are in file order.
- [x] 3.3 GREEN: the passes from design decision 2, covering QB, K, and the RB/WR/TE dedicated
      slots, then FLEX×2 and D×2 in file order. Store `[9]Record` in `Lineup`.
- [x] 3.4 RED-or-guard: an FB with no RB fills the RB slot and keeps position `FB`. If it already
      passes, break the RB slot so it accepts only `RB`, see it fail, then restore.
- [x] 3.5 Guard: three RBs before the WR and TE → RB₁ in RB, RB₂ and RB₃ in FLEX in file order.
      Guard: three TEs → TE₁ in TE, TE₂ and TE₃ in FLEX.
- [x] 3.6 Guard: CB above DE → CB is the first D. Guard: swapping two RB lines swaps the RB and the
      FLEX, and no other slot changes.

## 4. Rosters that cannot fill the slots are refused

- [x] 4.1 RED: ten records → `Read` fails, naming the file, and the error states 10. Confirm it
      fails because no error is returned.
- [x] 4.2 GREEN: check the count in `newLineup`.
- [x] 4.3 Guard: eight records → error stating 8. Guard: nine records interleaved with comments and
      blank lines → success.
- [x] 4.4 RED: nine records with no TE (QB, RB, RB, WR, WR, WR, K, D, D) → error naming `TE`.
      Confirm it fails because no error is returned, or because the TE slot holds a zero record.
- [x] 4.5 GREEN: after the passes, an empty slot is an error naming the first empty slot.
- [x] 4.6 Guard: two QBs → error (a FLEX is empty). Guard: three defenders → error.
- [x] 4.7 Update the stale "skipped line" comment in `parse`, which mentions promoting a bench
      player.

## 5. Render back to CSV in slot order

- [x] 5.1 RED: read the K-first/QB-last file from 3.2 and call `WriteCSV`. Expect the exact text
      `id,name,position,team\n` followed by nine lines in slot order. Confirm it fails because the
      output is empty.
- [x] 5.2 GREEN: write the header and `starters` in order.
- [x] 5.3 Guard: an FB in the RB slot renders as `FB` (break it by writing the slot name, then
      restore). Guard: round trip, where reading the rendered text gives equal `Starters()`.

## 6. Matchup page in slot order

- [x] 6.1 Guard: in `internal/web`, a fixture whose bojjaes file lists K first and QB last renders
      that column with the QB's name first, the K seventh, and the defenders last. Temporarily make
      `Starters()` return file order, see it fail, then restore.
- [x] 6.2 `internal/data/lineups_test.go`: replace the `StarterCount` check with
      `len(Starters()) == 9`, or keep it if `StarterCount` survives. The every-week walk now proves
      every embedded file is a legal lineup. Run `go test ./...`.

## 7. Docs

- [x] 7.1 Amend ADR 0004 #9 in place. Position is still a label. Slot is no longer implied by file
      order: it is assigned by position (QB RB WR TE FLEX FLEX K D D), with file order breaking
      ties, and a file holds exactly nine records. Add a dated amendment note to the status line.
- [x] 7.2 Amend in place the "nine starters plus bench" phrase in ADR 0005 (≈ line 174) and the
      `internal/lineup` rows in `docs/package-dependencies.md` (the starters/bench split and "the
      first nine records are starters").
- [x] 7.3 Final: `go vet ./... && go test ./...`, then `openspec validate lineup-slots`.
