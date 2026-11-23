Each behaviour is one red-green pair. A RED counts only when the test has run and failed on a
value mismatch or a missing error, never on a build error. The stub sections exist so that this
holds. A "Guard" test already passes: break the production code on purpose, watch the test fail,
then restore it. Run the package's tests after every step.

Out of scope: passphrase, export, game-start lock, replacing or deleting an opponent.

## 1. One-lineup weeks in `lineup.Tree`

- [x] 1.1 RED: flip `TestMatchupRefusesLoneLineup`. A week holding only `bojjaes.csv` resolves to
      `("bojjaes", "", nil)`. GREEN.
- [x] 1.2 RED: a week holding only `wood.csv` resolves to `("", "wood", nil)`. GREEN.
- [x] 1.3 Guard: an empty week still fails with `ErrTooFewLineups`. Two files without bojjaes still
      fail with `ErrNotOurMatchup`. Three files still fail with `ErrTooManyLineups`.
- [x] 1.4 Guard: `bojjaes.csv` with `.wood.csv.tmp` resolves to `("bojjaes", "")`.

## 2. TBD column on the matchup page

- [x] 2.1 RED: flip the matchup test that expects 500 for a lone `bojjaes.csv`. It now gets 200,
      the left column holds the Bojjaes' starters, and the right heading reads `TBD`. GREEN:
      `renderWeek` skips reading an empty side and the view marks that column as TBD.
- [x] 2.2 RED: the right TBD column has no total and no starter rows. GREEN in the template.
- [x] 2.3 RED: a lone `wood.csv` puts `TBD` on the left and wood on the right. GREEN.
- [x] 2.4 Guard: the stats stub is called once for a one-lineup week.
- [x] 2.5 Guard: `GET /2026` with a one-lineup latest week answers 200 (season test).

## 3. `volume.Store` stubs (all green)

- [x] 3.1 Add `Store`, `NewStore(mount string) *Store`, a `Save(season, week int, team string,
      l lineup.Lineup) error` that returns nil, and `OpponentSetError{Existing string}` with
      `Error()`. Test helper: a valid nine-player `lineup.Lineup` built with `FromSlots`.

## 4. `Store.Save`

- [x] 4.1 RED: saving bojjaes to an empty mount writes `lineups/2026/12/bojjaes.csv`, and
      `lineup.New(os.DirFS(...)).Read` returns the same starters. GREEN: `MkdirAll`, then
      `WriteCSV` to the file.
- [x] 4.2 RED: re-saving bojjaes with a different lineup replaces it. GREEN if not already.
- [x] 4.3 RED: with `bojjaes.csv` present, saving wood adds `wood.csv`. Guard: the directory holds
      exactly those two files.
- [x] 4.4 RED: with `bojjaes.csv` and `wood.csv` present, saving fuego returns
      `*OpponentSetError{Existing: "wood"}`, no `fuego.csv` exists, and `wood.csv`'s bytes are
      unchanged. GREEN: list the week's `.csv` non-dot files before writing.
- [x] 4.5 RED: with only `wood.csv` present, saving fuego returns
      `OpponentSetError{Existing: "wood"}`. Saving bojjaes succeeds. GREEN if not already.
- [x] 4.6 Guard: re-saving wood when wood is the opponent succeeds.
- [x] 4.7 Guard: a `.fuego.csv.tmp` and a `notes.txt` in the week don't count as an opponent.
- [x] 4.8 RED (atomicity): no file other than the target is left in the week after a save, and a
      save whose write fails (inject the failure through an unexported `writeLineup` hook in the
      test) leaves the previous `bojjaes.csv` bytes intact and no temp file behind. GREEN:
      `CreateTemp` with a dot prefix, then `Sync`/`Close`/`Rename`, removing the temp file on
      error.
- [x] 4.9 RED (serialization): 20 goroutines alternately save wood and fuego to a week holding
      only `bojjaes.csv`. Afterward the week holds exactly two `.csv` files, and every refused save
      got `OpponentSetError`. Run with `-race`. GREEN: a `sync.Mutex` around check-then-write.

## 5. Form plumbing (all green)

- [x] 5.1 Refactor `resolveLineupForm` to take a `url.Values` (GET routes pass `r.URL.Query()`).
      No behavior change; the existing tests stay green.
- [x] 5.2 Declare `web.LineupSaver interface { Save(season, week int, team string, l lineup.Lineup)
      error }`. Add a recording fake in `lineup_test.go`. Add `LineupSaveHandler(rosters, tree,
      current, games, saver)` returning 200 with nothing written.

## 6. `POST /lineup`

- [x] 6.1 RED: a valid bojjaes proposal with the current season and week → 303, `Location:
      /lineup?team=bojjaes`, and the fake recorded one save with that week, team, and starters.
      GREEN.
- [x] 6.2 RED: a duplicate player → 400, the body lists the problem and shows the posted picks
      still selected, and nothing was saved. GREEN by re-rendering the form view.
- [x] 6.3 RED: `team=../wood` → 400 with the current-week stub at 0 calls and no save. GREEN if
      not already.
- [x] 6.4 RED: posted `week=12` while the current week is 13 → 409, a body saying the week changed,
      and no save. GREEN.
- [x] 6.5 Guard: the current week is NFL week 18 → 404, no save. The current-week source errors →
      502, no save.
- [x] 6.6 RED: the fake returns `&volume.OpponentSetError{Existing: "wood"}` → 409 and the body
      contains `bojjaes&#39; opponent is already wood` (escaped). GREEN.
- [x] 6.7 RED: the fake returns another error → 500, logged. GREEN.

## 7. Form page

- [x] 7.1 RED: with a saver, the slots form has a `Save` button with `formmethod="post"` and hidden
      `season`/`week` inputs. Without a saver there's no `Save`. GREEN: a `CanSave` view field.
- [x] 7.2 RED: a tree holding wood for the current week → `Saved for week 12`. GREEN: have
      `startingLineup` return the week it found.
- [x] 7.3 RED: newest lineup is week 11 → `Not saved for week 12` and `week 11`. No lineup at all →
      `Not saved for week 12` only. GREEN.
- [x] 7.4 RED (chrome test, alongside `lineup_chrome_test.go`): pressing Save posts the form and
      leaves the clipboard untouched, and pressing Copy still copies. GREEN: the submit listener
      returns early when `event.submitter` is the Save button.

## 8. Wiring

- [x] 8.1 RED (mux test): with no saver, `POST /lineup` → 405. With a fake saver, it reaches the
      handler (303 on a valid post). GREEN: `newMux(..., saver)` registers the route only when
      `saver` isn't nil.
- [x] 8.2 Guard: `POST /lineup.csv` → 405, and `GET /lineup` and `GET /2025` still route as before.
- [x] 8.3 `main`: pass `volume.NewStore(cfg.lineupVolume)` when the volume is set, else nil.
- [x] 8.4 End to end (cmd/server test over a temp mount): seed, POST a valid bojjaes lineup,
      `GET /2026/<week>` shows those starters with the opponent `TBD`. Re-run `volume.Lineups`
      (a restart) and the save is still served.

## 9. Docs

- [x] 9.1 Amend `docs/adr/0005-writable-lineups-on-a-volume.md` in place, with an amended date in
      the status line:
      - Decision 3: the writable week is the form's current week (Sleeper, ≤ 17). The posted week
        must match it.
      - Decision 7: the passphrase is deferred and the POST is open for now.
      - Decision 8: a week may hold one lineup (TBD), and a third team is refused rather than
        replacing the opponent.
      Update the Consequences and Follow-ups to match.
- [x] 9.2 `backlog.md`: check "Persist a submitted lineup" and "Save from the lineup form" with
      this change's shape. Close the "Writable weeks vs. current week" thread. Leave the
      passphrase and export rows open.
- [x] 9.3 Run `openspec validate save-lineup-from-form`, `go vet ./...`, and `go test -race ./...`.

## 10. Local volume

- [x] 10.1 Add `/.volume/` to `.gitignore`. Then add `LINEUP_VOLUME=.volume` to the gitignored
      `.env.local`: create it if missing, keep any existing lines, and leave
      `config/local.env` unchanged. The path is relative, so the server must run from the
      repo root. Run `direnv allow` (or `direnv reload`) and confirm `echo $LINEUP_VOLUME`
      prints `.volume`.
- [x] 10.2 README: add a "Run with a local volume" section. It should say that `.env.local` sets
      `LINEUP_VOLUME=.volume`, that boot seeds `.volume/lineups` from git, that saves land
      there, and that `rm -rf .volume` resets the store. Also show the one-off form without
      direnv: `LINEUP_VOLUME=.volume go run ./cmd/server`.
- [ ] 10.3 Manual check before archiving, with `go run ./cmd/server` and the local volume:
      1. Save bojjaes on `/lineup`. `cat .volume/lineups/2026/<week>/bojjaes.csv` shows that
         lineup, and `/2026` shows it with the opponent as TBD.
      2. Restart the server. The save is still served.
      3. Save wood, then fuego. Fuego is refused with "bojjaes' opponent is already wood".
      4. Add or edit that week's CSV under `internal/data/lineups`, so it differs from the save,
         and restart. Git's copy is served, and the log names the overwritten file. Revert the
         edit afterwards.
