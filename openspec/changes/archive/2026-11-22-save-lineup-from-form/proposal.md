## Why

The lineup form builds and checks a lineup on a phone, but the only way to make it the served
lineup is still to download the CSV, commit it, and deploy. The volume store and per-week seeding
(ADR 0005) already keep runtime data across restarts and deploys; what is missing is the write.
This change adds it, so a lineup set from a phone is the one the scoreboard shows.

## What Changes

- A **Save** button beside Copy on `/lineup`. It posts the form's slots to `POST /lineup`, which
  applies the same team, current-week, and proposal checks as Copy and Download, then writes the
  lineup to the volume store and redirects back to the form.
- Only the week the form shows can be saved: the current week from Sleeper, ≤ 17. The form carries
  the season and week it showed; a save whose week no longer matches the current one is refused
  rather than filed under the new week.
- `bojjaes` saves to `bojjaes.csv`. Any other team saves as the Bojjaes' opponent for that week:
  created when the week has no opponent, overwritten when it is the same team, refused when the
  week already holds a different one ("bojjaes' opponent is already wood"). A save never deletes a
  file and never makes a third.
- Writes are atomic (dot-prefixed temp file in the week directory, then rename), serialized in
  process, and create the week directory when it is new.
- The form states whether the team's lineup for the week is saved, or which earlier week the slots
  start from.
- Form polish: Save is the primary button, on the left at half the row; Copy and an icon-only
  Download share the rest. Download is never greyed out: it checks the lineup as Copy does,
  downloading when valid and reloading with the problems otherwise. An edit turns the saved-state
  line into "Unsaved changes".
- A week directory holding one lineup is now a matchup with one side to be decided. The matchup
  page shows the missing side — the Bojjaes or the opponent — as **TBD**. Two lineups without the
  Bojjaes, or three lineups, are still refused.
- With no volume configured (local dev, tests), there is no write path: no Save button and no
  `POST /lineup` route.
- Docs: ADR 0005 amended in place (decisions 3, 7, 8); backlog rows for persist and Save checked,
  the "writable weeks vs. current week" thread closed.

Out of scope: passphrase (`POST /lineup` is open — an accepted gap), export of unarchived weeks,
a lock on game start. Known limitation: a wrong opponent once saved can only be corrected through
git.

## Capabilities

### New Capabilities
- `lineup-save`: saving a checked lineup from the form into the volume store — the route, the
  week guard, the Bojjaes/opponent file rule, atomic writes, and the redirect.

### Modified Capabilities
- `lineup-form`: "The form writes nothing" is replaced by a Save button when a store is
  configured; the page states whether the week's lineup is saved. Save leads the action row, and
  Download checks the lineup instead of being greyed out.
- `lineup-source`: a week directory holding one lineup resolves to a matchup with one side
  missing, instead of being refused.
- `matchup-page`: a one-lineup week renders with the missing side as TBD instead of a 500.

## Impact

- `internal/volume`: new store writer (`Save`), the only code that writes lineup files at runtime.
- `internal/lineup`: `Tree.Matchup` accepts a lone lineup.
- `internal/web`: `POST /lineup` handler, Save button and saved-state line in `lineup.html`, TBD
  column in `matchup.go`/`matchup.html`.
- `cmd/server`: `newMux` takes an optional store; `main` builds it from `LINEUP_VOLUME`.
- No new dependencies, no new upstream calls: a save reads the hour-cached current week only.
- Seeding is unchanged: a saved week survives restarts and deploys until git's copy of that week
  changes, which overwrites it and logs the saved files.
- `docs/adr/0005-writable-lineups-on-a-volume.md`, `backlog.md`.
