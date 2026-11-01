Each behaviour below is one red-green pair. A RED task is not done until the test has been run and
has failed **for the expected behavioural reason** — a wrong file, a missing week, a wrong hash, a
missing log line — not a build error or a missing symbol. Section 1 lands compiling stubs that are
deliberately wrong (a hash of `""`, a `Seed` that does nothing), so every later RED fails on a value
mismatch.

`internal/lineup` seeding tests use a `fstest.MapFS` source and a `t.TempDir()` mount, and assert
on files under the mount (and through `lineup.New(os.DirFS(<mount>/lineups))` where a read is the
behaviour). Only section 7 touches `lineup.Embedded`.

Out of scope: the write handler, passphrase, form, export, `fly.toml`. If a task seems to want one,
it is the wrong task.

## 1. Compiling stubs to fail against

- [x] 1.1 In a new `internal/lineup/seed.go`, add `weekHash(fsys fs.FS, season, week string)
      (string, error)` returning `"", nil`, and `Seed(src fs.FS, mount string, logf func(string,
      ...any)) error` returning `nil`. Add a test helper that builds a `fstest.MapFS` week
      (`<s>/<w>/bojjaes.csv` + opponent, valid headered rosters) and one that reads a mount's
      week back as `map[name]contents`. Confirm `go build ./...`; no assertions yet.

## 2. The week hash

- [x] 2.1 RED: two MapFS trees with the same week, same names and contents, different `ModTime`s →
      equal hashes; and a tree whose contents differ → different hash. Run; confirm the
      "different" assertion fails (both `""`).
- [x] 2.2 GREEN: SHA-256 over sorted regular files as `name NUL len NUL contents`, hex (design 2).
- [x] 2.3 RED: same contents (every file identical bytes, so sort order can't tell the weeks
      apart), opponent named `wood.csv` vs `aroma.csv` → hashes differ. If 2.2 already passes
      it, record as a guard and see it fail with names dropped from the hash.
- [x] 2.4 Guard: two-file week {`1.csv`:`a`, `2.csv`:`bc`} vs one `1.csv` holding
      `"a2.csv\x00bc"` hash differently (length prefix); see it fail with the length dropped,
      then restore.

## 3. A fresh volume is seeded

- [x] 3.1 RED: source 2026/1–2; empty mount. After `Seed`, assert the mount's `lineups/2026/1` and
      `/2` hold the source files byte-for-byte. Run; confirm it fails with the week missing.
- [x] 3.2 GREEN: walk numbered weeks of `src` (move the walk out of `layered.go` into `seed.go`, do
      not delete `layered.go` yet), create `lineups/<s>/<w>/`, copy files.
- [x] 3.3 RED: same fixture; assert `seed-manifest.json` parses to `{"weeks": {"2026/1": h1,
      "2026/2": h2}}` with `weekHash` values. Confirm it fails with no file.
- [x] 3.4 GREEN: write the manifest after each copied week, temp-then-rename (design 3).
- [x] 3.5 Guard: source holds `lost+found/1/bojjaes.csv` (non-empty, so the walk reaches it), `notes.txt`, `2026/draft/`; assert none are copied
      or recorded. See it fail with the numeric check removed.
- [x] 3.6 Guard: mount has no `lineups/` at all (only `lost+found/`) → seeded as 3.1; `lost+found`
      untouched.

## 4. Replace-or-leave per week

- [x] 4.1 RED: mount pre-seeded from source v1; manifest records v1's hash; store's 2026/2 then
      phone-edited. Seed again from v1. Assert the store's 2026/2 is still the phone edit. Run;
      confirm it fails with the edit reverted (Seed currently copies unconditionally).
- [x] 4.2 GREEN: read the manifest; skip a week whose entry equals its hash.
- [x] 4.3 RED: manifest holds v1's hash for 2026/2; source is v2 (changed roster). Assert the store
      holds v2 and the entry is v2's hash. Confirm failure, or record as guard if 4.2 already
      copies on mismatch.
- [x] 4.4 RED: store holds 2026/2 but the manifest has no entry → replaced with source, entry
      recorded. Same guard rule as 4.3.
- [x] 4.5 RED: unchanged second seed writes nothing — record the manifest's mtime/bytes and the
      store's files, seed again, assert identical. Confirm it fails if the manifest is rewritten
      on a no-op boot; GREEN: write the manifest only when a week was copied.

## 5. Whole-week replacement and kept weeks

- [x] 5.1 RED: store 2026/2 = `bojjaes.csv` + `wood.csv` (entry = old hash); source 2026/2 =
      `bojjaes.csv` + `aroma.csv`. Assert the store week holds exactly those two files and
      `lineup.New(os.DirFS(<mount>/lineups)).Matchup(2026, 2)` is `bojjaes`/`aroma`. Confirm it
      fails with `ErrTooManyLineups` (copy overlays, does not remove).
- [x] 5.2 GREEN: remove the store's week directory before copying (design 4).
- [x] 5.3 Guard: while replacing 2026/2, the store's 2026/1 (phone-edited, entry matching) is
      byte-for-byte unchanged.
- [x] 5.4 RED/guard: store holds 2026/9, source does not; manifest has entries for 2026/9 and
      2026/3 (absent from source). After seeding both store weeks and both entries are unchanged.
      Confirm a failure if manifest rewriting drops unknown entries; GREEN: merge into the loaded
      manifest rather than rebuilding it.

## 6. Overwrite log and failure modes

- [x] 6.1 RED: recording `logf`; store 2026/2 phone-edited `aroma.csv`, entry = v1 hash, source v2.
      Assert a log line names `2026/2`, and a line carries `aroma.csv` and the phone-edited
      contents (`%q`). Confirm it fails with no log calls.
- [x] 6.2 GREEN: before removing an existing store week, hash it; if it equals neither the entry
      nor the new hash, log header + one line per file (design 5).
- [x] 6.3 Guard: store week equals v1 (what git last seeded), source v2 → replaced, no overwrite
      logged. Fresh week (absent in store) → no overwrite logged. See the first fail with the
      check reduced to "store week exists", and the second fail with every replaced week logged.
- [x] 6.4 RED: store-only phone-set 2026/9, no entry, source adds a different 2026/9 → logged and
      replaced. Guard: store already holds exactly the new source (crash-before-manifest case) → no
      log, entry recorded.
- [x] 6.5 RED: manifest file holds `not json` → `Seed` returns nil, logs that the manifest was
      unreadable, and every week is re-seeded (with overwrites logged). Confirm it fails with a
      parse error returned.
- [x] 6.6 RED: mount is a read-only directory (`chmod 0o555` on `t.TempDir()`; skip if running as
      root) → `Seed` returns a non-nil error. Confirm it fails against a `Seed` that swallows
      errors, or record as guard.

## 7. `cmd/server` wiring and removing the layered read

- [x] 7.1 Change `lineupTree` to `lineupTree(volume string, logf func(string, ...any)) (fs.FS,
      error)`, keeping today's body (Layered) so it compiles. RED: rewrite `volume_test.go` —
      with an empty `t.TempDir()` volume, the returned tree's `LatestWeek` for every embedded
      season equals the embedded answer *and* `<vol>/seed-manifest.json` exists. Confirm it fails
      on the missing manifest.
- [x] 7.2 GREEN: `Seed(lineup.Embedded, volume, logf)` then `os.DirFS(<vol>/lineups)`.
- [x] 7.3 RED: build the tree once over an empty volume (seeds it), then replace the opponent file
      in the store's embedded-latest week by hand and build the tree again → `Matchup` returns the
      edited opponent (the second seed left the week alone, and reads come from the volume).
      Confirm it fails while the tree still layers or reads embedded; otherwise record as guard.
- [x] 7.4 Guard: `lineupTree("", ...)` returns `lineup.Embedded`, nil error, and writes nothing
      (cwd and a temp dir untouched).
- [x] 7.5 RED: unwritable volume (as 6.6) → `lineupTree` returns an error. Then wire `main.go`:
      `log.Fatalf` on that error before `newServer`/`run`.
- [x] 7.6 Delete `internal/lineup/layered.go` and `layered_test.go`; move any helper still used to
      `seed.go`. `go vet ./... && go test ./...` green.

## 8. Hand verification

- [x] 8.1 `go run ./cmd/server` with `LINEUP_VOLUME` unset: `/2026` shows the embedded latest week;
      no files written.
- [x] 8.2 With `LINEUP_VOLUME` at an empty scratch dir: startup creates `lineups/` with every
      embedded week and `seed-manifest.json`, then listens; pages match 8.1. Restart: no week
      replaced, manifest unchanged.
- [x] 8.3 Phone-edit a store week by hand (change the opponent file), restart: edit is served. Then
      clear that week's manifest entry, restart: the edit is logged (name + contents) and git's
      week is served.
- [x] 8.4 Reread `design.md` and the delta specs against the code; if anything drifted, change one
      side, not both. Update `backlog.md`'s checkbox for this line and strike the now-replaced
      two-layer line's "Replaces" note if it no longer reads right.
- [x] 8.5 Rewrite the lineup-storage section of `docs/architecture.md` (status table, diagram,
      prose) to match the code and amended ADR 0005: volume-only reads and boot seeding with the
      per-week manifest marked built; drop the two-layer read, stale-week ignoring, boot-time
      clear, and "at most one week" wording; the deploy row is one machine with
      `min_machines_running = 0`, not `min = 1`. Amend in place.
