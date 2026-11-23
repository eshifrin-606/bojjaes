## Context

The pieces this change needs mostly exist. `volume.Lineups` seeds `/data/lineups` from git and
serves it through `os.DirFS`, and `lineup.Tree` opens files on every request, so a written file is
served on the next request with no cache to invalidate. `resolveLineupForm` already runs the team,
current-week, pool, and proposal checks shared by `/lineup`, `/lineup.csv`, and `/lineup.txt`, and
`Lineup.WriteCSV` renders a lineup in slot order. What's missing is the write itself, a handler and
button for it, and a matchup page that tolerates the one-file week a save can create.
`internal/volume` is documented as the only package that writes lineup files, so the writer goes
there.

## Goals / Non-Goals

**Goals:** saving a checked lineup from the form into the store, for the form's week only; at most
two files per week (the Bojjaes and one opponent); atomic writes; a matchup page that shows a
missing side as TBD.

**Non-Goals:** passphrase or any auth, export of unarchived weeks, a lock on game start, replacing
or deleting an opponent, saving any week other than the current one, a save path without a volume.

## Decisions

### 1. `volume.Store` writes; `web` decides what to write

```
POST /lineup ─▶ web: team ─▶ current week ─▶ posted week == current? ─▶ proposal valid?
                                                                          │
                                         volume.Store.Save(season, week, team, lineup)
                                           lock ─▶ opponent rule ─▶ temp file ─▶ rename
```

`volume.NewStore(mount)` returns a `*Store` with `Save(season, week int, team string, l
lineup.Lineup) error`. It owns the on-disk layout (`<mount>/lineups/<season>/<week>/<team>.csv`),
the opponent rule, atomicity, and a `sync.Mutex`. `web` defines a one-method `LineupSaver`
interface so handler tests use a fake. The handler never builds a path.

The opponent rule lives in `Save` rather than in `web` because it has to be checked under the same
lock as the write. Otherwise two concurrent opponent saves could each see an empty slot. `Save`
returns `*OpponentSetError{Existing string}`, which the handler maps to 409 with
`bojjaes' opponent is already <Existing>`.

Rejected: writing through `lineup.Tree`. It sits over a read-only `fs.FS` and is also used over the
embedded tree, so a write method on it would be meaningless half the time.

### 2. Atomic write: `os.CreateTemp(weekDir, "."+team+".csv.*")`, write, `Sync`, `Close`, `Rename`

The temp file is in the same directory, so the rename stays on one filesystem and is atomic. The
dot prefix keeps a leftover from a crash out of `Tree.Matchup` (already ignores dotfiles) and out
of the opponent rule. On any error the temp file is removed. `os.MkdirAll` creates a new week
directory first. There's no directory fsync: a lost rename on power loss leaves the previous
lineup, which is the outcome atomicity is meant to guarantee anyway.

Seeding's `weekHash` counts dotfiles. A stray temp file would only make `logOverwrite` log the week
when git next replaces it, so it's harmless and doesn't need a change.

### 3. The posted week guards against week turnover; the current week chooses the week

The week is still `current.AsOf` → `Settable()`, as on the GET form. The hidden `season`/`week`
fields are compared with it and never used as the target. A mismatch is 409. Without this check, a
tab left open from Monday could file a week-12 lineup under week 13 after Sleeper flips. Rejected:
saving to the posted week. That would reopen "which weeks are writable", which the user closed as
"the week the form shows".

### 4. Resolve from a `url.Values`, not `r.URL.Query()`

`resolveLineupForm` currently reads `r.URL.Query()`. It will take the values as a parameter: GET
passes `r.URL.Query()`, POST passes `r.PostForm` (after `ParseForm`). This keeps one checking path
for all four routes, so Save cannot accept a lineup that Copy or Download would refuse.

### 5. POST outcomes

- Success: `303 See Other` to `/lineup?team=<team>` (Post/Redirect/Get, so a reload doesn't
  re-post). With no proposal in the URL, the page starts from the team's week file, which is now
  the saved one, and the "Saved for week N" line confirms it. No `?saved=1` flag is needed.
- Refusal (invalid proposal, opponent set, week changed): the form is re-rendered from the posted
  values with the reason in `Problems`, at 400 or 409. Re-rendering keeps the user's picks on the
  phone. The CSV and text routes keep their plain-text errors.
- Store write error: 500, logged.

### 6. Routing: `POST /lineup` registered only with a store

`newMux` gains a `saver web.LineupSaver` argument that may be nil. `main` passes
`volume.NewStore(cfg.lineupVolume)` when the volume is set, else nil. With nil, the route isn't
registered, so Go's mux answers 405 as it does now, and the view's `CanSave` is false, so no button
renders.

### 7. The Save button posts; Copy's script leaves it alone

`<button formmethod="post" formaction="/lineup" name="save">Save</button>` sits in the same slots
form. The Copy script's submit listener checks `event.submitter`: for Save it returns without
calling `preventDefault`, so the browser does a normal POST. An edit resets Copied to Copy, which
doesn't apply to Save.

### 8. Saved-state line from `Tree.LatestLineup`

`startingLineup` already calls `tree.LatestLineup(team, season, week)`. It will also return the
week it found. If that week equals the current week, the page shows "Saved for week N". Otherwise
it shows "Not saved for week N · starting from week M", or "Not saved for week N" when there's no
earlier lineup. This adds no extra disk reads.

### 9. One-file weeks: `Tree.Matchup` returns `""` for the missing side; `renderWeek` skips it

`Matchup` returns `(ours, "")` or `("", theirs)` for a lone file and still errors on 0, 3+, or two
files without the Bojjaes. `renderWeek` reads only the non-empty sides. For an empty side the view
column is `TBD` with no starters and no total. Stats are still fetched once, and the existing
"implies no winner" styling is unchanged. Neighbor links and `LatestWeek` need no change: a
one-file week is a directory like any other.

The error `ErrTooFewLineups` now applies only to an empty directory. It stays, since the four
error values distinguish four mistakes.

### 10. Form polish: Save first, Download checks like Copy

The action row is a grid: `2fr 1fr 1fr` (Save, Copy, Download) with a saver, `3fr 1fr` (Copy,
Download) without. Save gets a dark fill and white text; the others keep today's grey. Download is
`<button formaction="/lineup.csv" aria-label="Download">` holding an inline SVG, so it is in the
form like Copy. With Save first, the script finds Copy and Download by selector, not as the form's
first button.

Download is never disabled, so `CSVLink`/`csvLink` go. The submit listener branches on
`event.submitter`. For Download it fetches `/lineup.csv?<query>`: on `ok` it does
`location.assign` to the same URL, and the attachment downloads without leaving the page; otherwise
it reloads at `/lineup?<query>`, as Copy does, so the problems are listed. The clipboard check moves
inside Copy's branch, so Download needs no clipboard. The valid case fetches twice; that's cheap,
and it keeps the server's file name. Rejected: a blob download, which needs its own file name.

No-JS fallback: the button's plain GET to `/lineup.csv`. An invalid lineup answers today's 400,
whose plain-text body already lists the problems. Rejected: redirecting an invalid `/lineup.csv` to
the form. That would change the route's statuses, which `/lineup.txt` and `POST /lineup` are
specified to share.

The status line becomes `Unsaved changes` on the slots form's `change` event. It is client-side
only, so a reload shows the server's line again.

## Risks / Trade-offs

- [`POST /lineup` is unauthenticated; anyone who finds it can overwrite this week's lineups] →
  accepted for now. Writes are limited to the current week and to two files. Re-saving or a git
  change to the week restores it. The passphrase is the next backlog row.
- [A wrong opponent once saved can't be replaced from the form] → by design (no deletes). Fix it
  through git, whose seeding replaces the week whole.
- [A git change to a week overwrites a phone save of that week] → existing ADR 0005 loss path. The
  overwrite is logged, so download before committing.
- [Sleeper's state endpoint down → no saves] → same as the GET form (502). Acceptable.
- [The opponent rule and atomic writes rely on a single process] → ADR 0005 pins exactly one
  machine. The mutex covers concurrent requests in that process.

## Migration Plan

No data migration. The volume is already mounted in production (`LINEUP_VOLUME=/data`), so
deploying turns Save on. Rollback is a redeploy of the previous build. Saved files remain valid
lineups, and one-file weeks would then 500 on the matchup page until the opponent is added through
git.
