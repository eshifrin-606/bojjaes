Each behaviour below is one red-green pair. A RED task is not done until the test has been run and
has failed **for the expected behavioural reason** — a wrong week, wrong team, or wrong roster, not
a build error or a missing symbol. So section 1 lands a compiling stub that is deliberately wrong
(it returns the archive untouched), and every later RED fails on a value mismatch against it.

All `internal/lineup` tests build both layers as `fstest.MapFS` and assert through
`lineup.New(Layered(...))` — `LatestWeek`, `HasWeek`, `Matchup`, `Read` — never on the layer's
internals. No test touches a real volume or the real embedded tree except where stated.

Out of scope: any write to the volume, auth, the form, boot-time clearing, export, `fly.toml`. If a
task seems to want one, it is the wrong task.

## 1. A compiling stub to fail against

- [x] 1.1 Add `Layered(archive, volume fs.FS, logf func(string, ...any)) fs.FS` in a new
      `internal/lineup/layered.go` that returns `archive`. Add a small test helper that builds a
      `fstest.MapFS` week (`<s>/<w>/bojjaes.csv` + opponent, valid headered rosters). Confirm it
      builds; no assertions yet.

## 2. A later volume week is served

- [x] 2.1 RED: archive holds 2026/1–3; volume holds `2026/4/`. Assert `LatestWeek(2026)` is 4. Run;
      confirm it fails with 3.
- [x] 2.2 GREEN: implement the layer's `Open`/`ReadDir` enough to route the active volume week to
      the volume and merge it into the season listing. Re-run; confirm pass.
- [x] 2.3 RED: same fixture; assert `Matchup(2026, 4)` yields the volume's teams and `Read` of the
      opponent yields the volume roster's first id. Run; confirm it fails (or, if 2.2 already
      routes files, record that it passed and why, and move on — do not weaken the test).
      *Passed first time: 2.2 routes `Open` as well as `ReadDir` for the active week, so `Read` and
      `Matchup` already hit the volume. Seen failing with `Open` forced to the archive.*
- [x] 2.4 GREEN as needed. Re-run; confirm pass. *No change needed (see 2.3).*
- [x] 2.5 RED: same fixture; assert `Matchup(2026, 2)` and a `Read` of week 2 still come from the
      archive. Run against a deliberately over-broad route if 2.2 routed too much; otherwise record
      it as a guard that passed first time.
      *Guard: passed first time; 2.2 routes only names at or under the active week.*

## 3. The volume week at the archive's latest shadows it whole

- [x] 3.1 RED: archive 2026/3 holds `bojjaes.csv` + `wood.csv`; volume 2026/3 holds `bojjaes.csv`
      + `aroma.csv`. Assert `Matchup(2026, 3)` is `bojjaes`/`aroma` (not a three-roster error) and
      `Read(2026, 3, "bojjaes")` returns the volume's roster. Run; confirm it fails on
      `ErrTooManyLineups` or the archive's `wood`.
      *Passed first time: 2.2 routing is already whole-week (decision 2). Seen failing with
      `ErrTooManyLineups` against a temporary per-file overlay of the week listing.*
- [x] 3.2 GREEN: hide the archive's copy of the active week entirely. Re-run; confirm pass.
      *No change needed (see 3.1).*
- [x] 3.3 RED: same fixture; assert `Read(2026, 3, "wood")` fails as not found. Confirm the
      failure is the embedded `wood.csv` leaking through (if 3.2 already hides it, record as guard).
      *Guard: passed first time. Seen failing (`wood` read from the archive) with `Open` temporarily
      falling back to the archive on a volume miss.*

## 4. The volume week can open a new season

- [x] 4.1 RED: archive ends at 2025/16; volume holds `2026/1/`. Assert `HasWeek(2026, 1)`,
      `LatestWeek(2026) == 1`, and `Matchup(2026, 1)` from the volume; `LatestWeek(2025) == 16`.
      Run; confirm it fails because season `2026` does not exist in the archive.
      *Failed with `LatestWeek(2026) = 0, false`; `HasWeek` already passed, since `Open` of the
      active week routes to the volume.*
- [x] 4.2 GREEN: make a season directory that exists only on the volume openable and listable, and
      compare weeks by `(season, week)` so 2026/1 counts as after 2025/16. Re-run; confirm pass.
      *Only the season listing changed here; the `(season, week)` ordering had no failing test yet
      and landed in 5.2, driven by 5.1.*

## 5. An older volume week is ignored, logged, untouched

- [x] 5.1 RED: archive latest 2026/3 with 2026/2 = `bojjaes`/`wood`; volume 2026/2 =
      `bojjaes`/`aroma`. Assert `Matchup(2026, 2)` is `wood`. Run; confirm it fails with `aroma`
      (the router so far ignores ordering).
      *Failed with `aroma`.*
- [x] 5.2 GREEN: activate the volume week only when it is at or after the archive latest. Re-run.
- [x] 5.3 RED: same fixture with a recording `logf`; assert exactly one call whose message names
      2026 and 2. Run; confirm it fails with no call.
      *Asserts the line contains `2026/2` (a bare `2` is satisfied by `2026`). Failed with no call.*
- [x] 5.4 GREEN: scan the volume once in `Layered` and log each ignored week. Re-run; confirm pass.
      *The construction-time scan is for logging only; routing still rescans per call (decision 4).
      The scan now carries each week's `DirEntry`, so `ReadDir` no longer re-lists the volume and
      its guard against the volume changing between two listings is gone.*
- [x] 5.5 RED: volume holds `2026/2/` and `2026/4/` over archive latest 2026/3. Assert week 4 is
      served, week 2 comes from the archive, and 2026/2 is logged. Confirm the failure is a value
      mismatch, then GREEN: take the greatest volume week as the candidate.
      *The 2 + 4 fixture passed first time (stale weeks were already skipped, leaving 4 as the only
      candidate); kept as a guard, seen failing with stale weeks not skipped. Added a 2026/4 +
      2026/5 case, which failed with `LatestWeek = 4`, then GREEN.*
- [x] 5.6 RED: volume holds `2026/4/` and `2026/5/` over archive latest 2026/3, with a recording
      `logf`; assert exactly one call naming 2026 and 4. Confirm it fails with no call, then GREEN:
      log each volume week the greater one supersedes, at construction like stale weeks.
      *Failed with `logged []`.*

## 6. An absent or empty volume reads as the archive

- [x] 6.1 Guard: with an empty `fstest.MapFS` volume, assert `LatestWeek`, `HasWeek`, `Matchup`,
      and `Read` over the archive fixture match `lineup.New(archive)`. Expected to pass first time;
      break the layer temporarily (e.g. treat an empty volume as an error) to see it fail, then
      restore.
      *Guard: passed first time; seen failing with an empty volume turned into `ErrNotExist`.*
- [x] 6.2 RED: volume is an `os.DirFS` of a non-existent path under `t.TempDir()`. Assert the same
      answers as 6.1 and that the path still does not exist afterwards. Run; confirm it fails if
      the layer surfaces the volume's `ErrNotExist`, then GREEN: treat any error listing the volume
      root as "no volume week".
      *Passed first time: the volume walk already treats a listing error as no weeks. Kept as a
      guard; seen failing with `ReadDir` surfacing the volume root's error.*
- [x] 6.3 Guard: volume holds a non-numeric directory and a stray file at the root and at season
      level (e.g. `lost+found/`, `notes.txt`). Assert they are ignored, like `LatestWeek` ignores
      them.
      *Guard: passed first time; also asserts nothing is logged. Seen failing (`2026/draft` logged
      as stale) with the numeric-name check dropped.*

## 7. fs.FS conformance

- [x] 7.1 Run `fstest.TestFS` over the layer with the section 3 and section 4 fixtures, naming the
      expected files. Fix any contract violation (sorted `ReadDir`, `fs.ValidPath` rejection,
      `*fs.PathError` on missing names) with its own red-green if one surfaces.
      *Section 3 fixture conformed. Section 4 failed: the root listing omitted the volume-only
      season (fixed by merging it into `ReadDir(".")`), then `Open` of `.` disagreed with
      `ReadDir(".")` (fixed by opening `.` and the active season as a merged directory file).*

## 8. Configuration in `cmd/server`

- [x] 8.1 Add `resolveLineupVolume(getenv func(string) string) string` returning a deliberately
      wrong constant. RED: test that `LINEUP_VOLUME=/data` yields `/data`. Confirm value mismatch.
- [x] 8.2 GREEN: read `LINEUP_VOLUME`. RED/GREEN pairs for unset and empty → `""`.
- [x] 8.3 Add `lineupTree(volume string) fs.FS`. RED: `""` yields a tree whose `LatestWeek` for the
      embedded latest season matches `lineup.New(lineup.Embedded)`; with a `t.TempDir()` holding
      `lineups/<embedded latest season>/<latest+1>/` two valid rosters, `LatestWeek` is latest+1.
      Confirm failure against a stub that always returns `lineup.Embedded`, then GREEN: an empty
      `LINEUP_VOLUME` (unset or `""`) returns `lineup.Embedded`; otherwise
      `lineup.Layered(lineup.Embedded, os.DirFS(filepath.Join(volume, "lineups")), log.Printf)`.
- [x] 8.4 Wire `main.go`: `lineup.New(lineupTree(resolveLineupVolume(os.Getenv)))`. `go build ./...`
      and `go test ./...` green.

## 9. Hand verification

- [x] 9.1 `go run ./cmd/server` with `LINEUP_VOLUME` unset: `/2026` redirects to the embedded
      latest week and the page is unchanged.
- [x] 9.2 With `LINEUP_VOLUME` pointing at a scratch dir holding `lineups/<next week>/` (two copied
      rosters with a changed opponent): `/<season>` redirects to the new week and it renders from
      the volume; an older week on the volume is logged at startup and not served; the scratch dir
      is unmodified afterwards.
- [x] 9.3 Reread `design.md` and the delta spec against the code; if anything drifted, change one
      side, not both.
