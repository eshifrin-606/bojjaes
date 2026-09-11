Each behaviour below is one red-green pair. A RED task is not done until its test has run and
failed **on a value mismatch**, meaning a wrong short name, not a build error. Section 1 adds every
new symbol as a stub first so the REDs that follow compile. Where a scenario already passes against
the code written for an earlier step, the task says so. It is then kept as a pin, and the task
breaks the code on purpose to show the test can fail.

The derivation tests live in a new `internal/lineup/shortname_test.go`. Read-level tests extend
`internal/lineup/read_test.go`, using its `seedLineup` / `fstest.MapFS` helpers.

## 0. Gate

- [x] 0.1 Confirm the open decisions in `design.md` with the user. Confirmed: collisions fall back
      to full name; collision scope is starters only, with the bench resolved among itself;
      initials-style first names are kept as written; suffixes are dropped only from names of
      three or more words. Spec, design, proposal, and these tasks are updated to match.

## 1. Scaffold (no behaviour)

- [x] 1.1 Add `ShortName string` to `lineup.Record`, beside `Name`, with a comment saying it is
      display text only. Add `internal/lineup/shortname.go` with stubs:
      `shortName(name string) string` that returns `name`, and `fillShortNames(group []Record)`
      that does nothing. Run `go test ./...` and `go vet ./...`; confirm both are clean. Existing
      `Record` literals are keyed, so nothing else changes.

## 2. Derive a short name from one name

Add a table-driven `TestShortName` with one `{name, want}` row per step.

- [x] 2.1 RED: row `Josh Allen` → `J Allen`. Run `go test ./internal/lineup -run TestShortName`;
      confirm it fails with got `Josh Allen`.
- [x] 2.2 GREEN: split with `strings.Fields`, then return the first byte of the first field, a
      space, and the remaining fields joined by single spaces. Re-run; confirm pass.
- [x] 2.3 RED: row `Mahomes` → `Mahomes`. Confirm it fails with got `M ` (trailing space).
- [x] 2.4 GREEN: return `name` unchanged when it has fewer than two fields. Re-run; confirm pass.
- [x] 2.5 Pin: row `""` → `""`. It passes against 2.4's guard, so note that. Change the guard to
      `== 1` temporarily and confirm the row fails; restore.
- [x] 2.6 RED: row `Émile Smith` → `É Smith`. Confirm it fails with a mangled first byte in place
      of `É`.
- [x] 2.7 GREEN: take the first rune of the first field, not its first byte. Re-run; confirm pass.
- [x] 2.8 RED: row `Will McDonald IV` → `W McDonald`. Confirm it fails with got `W McDonald IV`.
- [x] 2.9 GREEN: drop the last field when it is `IV`. Re-run; confirm pass.
- [x] 2.10 RED: rows `Marvin Harrison Jr.` and `Marvin Harrison Jr` → `M Harrison`. Confirm both
      fail with the suffix still present.
- [x] 2.11 GREEN: replace the single `IV` check with the spec's full suffix set (`Jr.`, `Jr`, `Sr.`,
      `Sr`, `II`, `III`, `IV`, `V`). Re-run; confirm pass. Add rows `Luther Burden III` →
      `L Burden`, plus `Sr.`, `Sr`, `II`, and `V` rows, as pins that pass immediately.
- [x] 2.12 RED: row `Tony V` → `T V`, a two-word name whose last word looks like a suffix. Confirm
      it fails with got `T ` (the suffix was dropped, leaving nothing after the initial).
- [x] 2.13 GREEN: drop a suffix only when the name has more than two fields. Re-run; confirm pass.
- [x] 2.14 Pins, which pass against 2.13 (say so in the task log): `Amon-Ra St. Brown` →
      `A St. Brown`, `Ja'Marr Chase` → `J Chase`, and `Josh  Allen` (two spaces) → `J Allen`.
- [x] 2.15 RED: row `A.J. Brown` → `A.J. Brown`. Confirm it fails with got `A Brown`.
- [x] 2.16 GREEN: use the whole first field as the lead when it contains a period. Re-run; confirm
      pass.
- [x] 2.17 RED: row `KC Concepcion` → `KC Concepcion`. Confirm it fails with got `K Concepcion`.
- [x] 2.18 GREEN: also use the whole first field when it is exactly two runes and both are
      uppercase. Add a brief comment on why three capitals don't count. Re-run; confirm pass. Add
      rows `DJ Moore`, `DK Metcalf`, `TJ Watt`, and `JK Dobbins` (each unchanged) as pins that pass
      immediately.
- [x] 2.19 Pins for what is not initials, which pass against 2.18: `Al Smith` → `A Smith` (drop the
      uppercase check temporarily, confirm it fails, restore), `JOE Smith` → `J Smith` (drop the
      two-rune check temporarily, confirm it fails, restore), and `CeeDee Lamb` → `C Lamb`.
- [x] 2.20 Pin: row `A.J. Brown Jr.` → `A.J. Brown`. It passes against 2.13 and 2.16, since the
      suffix rule is independent of the lead.

## 3. Resolve collisions within a group

Add a `TestFillShortNames` that builds `[]Record` by hand, calls `fillShortNames`, and compares
each record's `ShortName`.

- [x] 3.1 RED: records `Josh Allen`, `Puka Nacua` → short names `J Allen`, `P Nacua`. Confirm it
      fails with empty short names from the stub.
- [x] 3.2 GREEN: set each record's `ShortName` to `shortName(rec.Name)`. Re-run; confirm pass.
- [x] 3.3 RED: records `Jameson Williams`, `Javonte Williams`, `Caleb Williams` → short names
      `Jameson Williams`, `Javonte Williams`, `C Williams`. Confirm it fails with `J Williams`
      twice.
- [x] 3.4 GREEN: derive every short name first and count them; a record whose derived short name
      appears more than once gets `Name` instead. Re-run; confirm pass.
- [x] 3.5 Pin: two records with different ids, both named `Josh Allen`, so both short names are
      `Josh Allen`. It passes against 3.4.
- [x] 3.6 Pin: records `""`, `""`, `Josh Allen` → short names `""`, `""`, `J Allen`. It passes
      against 3.4, since the fallback for an empty name is empty.

## 4. Reading a lineup fills short names

- [x] 4.1 RED: in `TestTreeReadReturnsParsedRecords` and
      `TestTreeReadReadsThroughTheSuppliedFilesystem`, add `ShortName` to each expected record
      (`A One`, `B Two`, `C Three`). Run `go test ./internal/lineup -run TestTreeRead`; confirm
      both fail with empty short names.
- [x] 4.2 GREEN: in `Tree.Read`, call `fillShortNames(records)` before building the `Lineup`.
      Re-run; confirm pass. Confirm `parse_test.go` is unchanged and passing.
- [x] 4.3 RED (scope): add a fixture `internal/lineup/testdata/shortnames.csv` with twelve records:
      `Jameson Williams`, `Josh Allen`, `Javonte Williams`, `Caleb Williams`, `Will McDonald IV`,
      `A.J. Brown`, `KC Concepcion`, `Puka Nacua`, `Bijan Robinson`, then bench `Jaylen Allen`,
      `Jordan Allen`, `Dak Prescott`. Read it through `Tree.Read` and assert `Starters()[1]` has
      short name `J Allen`. Confirm it fails with got `Josh Allen` (the bench Allens collide with
      it across the whole list).
- [x] 4.4 GREEN: in `Tree.Read`, build the `Lineup` first, then call
      `fillShortNames(l.Starters())` and `fillShortNames(l.Bench())`. Add a brief comment on why
      each group is resolved separately (a bench player must not push a starter to its full name).
      Re-run; confirm pass.
- [x] 4.5 Pins on the same fixture, which pass against 4.4: starter short names `Jameson Williams`,
      `Javonte Williams`, `C Williams`, `W McDonald` (with `Name` still `Will McDonald IV`),
      `A.J. Brown`, `KC Concepcion`; bench short names `Jaylen Allen`, `Jordan Allen`,
      `D Prescott`. Remove the `Bench()` call temporarily, confirm the bench assertions fail, and
      restore.
- [x] 4.6 Pin (crossing the split): seed ten records where the 2nd is `Josh Allen`, and read two
      orderings, one with `Jaylen Allen` 10th and one with it 9th. Assert `Josh Allen`'s short
      name is `J Allen` in the first and `Josh Allen` in the second. It passes against 4.4.
- [x] 4.7 Pin: seed two teams in the same week, one holding `Josh Allen` and the other holding
      `Jaylen Allen`. Read each; assert both short names are `J Allen`. It passes against 4.4.
- [x] 4.8 Pin: add a read test for empty names, since today only `parse_test.go` covers them. Seed
      a roster holding `4984,` and `4985` (no comma), read it through `Tree.Read`, and assert both
      records have an empty `Name` and an empty `ShortName`. It passes against 4.4.

## 5. Close-out

- [x] 5.1 Remove any comment that restates the code. Keep the `Record.ShortName` display-only note,
      the per-group scope comment, and the initials comment.
- [x] 5.2 `git diff --stat`: confirm nothing changed under `internal/web`, `scripts/`, or
      `internal/lineup/data/`, and that the only new testdata file is `shortnames.csv`.
- [x] 5.3 Run `go test ./...` and `go vet ./...`; confirm both are clean. Run
      `openspec validate add-short-player-names --strict`; confirm it is clean.
