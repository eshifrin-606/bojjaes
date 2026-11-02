## Context

Every piece this page needs exists except the page: `nflweek.Current.Settable()` and
`weekcache.Single` (current-week change), slot rules and `Lineup.WriteCSV` (lineup-slots),
`Tree.Read`/`LatestLineup` over the volume store, embedded `data.Rosters` (no reader yet), and
`gameState`/`kickoffText` for a starter's game line. ADR 0004 fixes the stack: `html/template`, no
framework, handler ordering as the guard against upstream volume, no margin. ADR 0005 keeps writes
for later rows; this page writes nothing.

## Goals / Non-Goals

**Goals:** a phone page that builds a legal lineup for the current week, round-trips through a
shareable GET URL, validates without 5xx, and yields a pasteable summary and a committable CSV.

**Non-Goals:** POST/persist, passphrase, home page, roster transactions, picking a season or week,
localising option kickoffs to the reader's zone, lineup-lock timing.

## Decisions

### 1. Routes: `GET /lineup` and `GET /lineup.csv` — mirrors `newMux`'s method-qualified routes

Both literal patterns beat `/{season}` in Go 1.22's mux by specificity, so neither captures the
other; mux tests pin it both ways. `newMux` grows two parameters:
`newMux(stats, games, current web.CurrentWeekSource, tree, rosters *lineup.Rosters)`. `main` wires
`weekcache.NewSingle("nflstate", client.CurrentWeek, weekcache.CurrentWeekTTL)` and sets
`Client.V1BaseURL = sleeper.V1BaseURL`, as the current-week design (decision 5) planned.

The CSV is a second route, not a `data:` link: `html/template` rewrites a `data:` href to
`#ZgotmplZ` unless the value is laundered through `template.URL`, and a route gives the download a
real `Content-Disposition` and its own tests.

### 2. Query scheme: one parameter per slot, lowercase slot name, Sleeper ID values

`/lineup?team=wood&qb=4984&rb=…&wr=…&te=…&flex=…&flex=…&k=…&d=…&d=…`. The select `name` is the
slot name, so the browser builds this URL from the form with no script; `url.Values` keeps
repeated keys in order, so `flex`/`d` fill their two slots in URL order. "Proposal mode" is "any
slot key present"; otherwise the starting lineup is shown. The CSV link's query is built by hand in
slot order (not `Values.Encode`, which sorts keys) so it reads like the form's own URL.
Rejected: one `lineup=id,id,…` parameter (positional, not what a form submits) and slot-indexed
keys `s0…s8` (opaque in a texted URL).

### 3. Handler ordering — mirrors `Handler`/`renderWeek`

1. Read `team` (default `lineup.OurTeam`); refuse unless it is in `rosters.Teams()` → 400.
2. `current.AsOf` (hour-cached) → 502 on error; `Settable()` false or week > `lineup.LastWeek`
   (17) → "no week to set" page (CSV route: 404).
3. Read the roster (failure → 500: our embedded data is wrong, as `renderWeek` treats a bad week
   dir) and the starting lineup via `tree.LatestLineup(team, season, week)` + `tree.Read`.
4. Parse and check the proposal against that pool.
5. `games.AsOf(season, week)` last, failure only drops game text (mirrors stats-then-games).

No query parameter names a season or week, so the only upstream keys reachable are the one state
read and the current week's games — URL enumeration cannot widen Sleeper volume (ADR 0004 §7).
`lineup.LastWeek = 17` lives in `internal/lineup` because it is a league rule and the persist row
will need it too; `nflweek` stays NFL-only.

### 4. `lineup` package additions — mirror `Tree`

- `func Slots() [StarterCount]Slot` and `func (s Slot) Accepts(position string) bool`, read from
  the existing `slotOrder`/`slotAccepts`, so the form filters options by the rule `newLineup` uses.
- `Rosters` in `internal/lineup/roster.go`, over an `fs.FS` (`NewRosters(data.Rosters)`), with `Teams() []string` and
  `Read(team) ([]Player, error)` reusing `parse` and `checkTeam`. Same shape as `Tree` over
  `data.Lineups`; no nine-record rule, since a roster is not a lineup.
- `FromSlots([StarterCount]Player) (Lineup, error)`: keeps the given slots, refuses empty ID,
  ineligible slot, or duplicate ID. Not `newLineup`, which reassigns by position and so cannot
  detect "WR chosen for QB". Its output feeds `WriteCSV` unchanged.
- `LatestLineup(team, season, week)` gains an at-or-before bound. Needed: the store may already
  hold next week (ADR 0005 writable weeks), and "this week's file, else an earlier one" is then one
  call. It has no production callers, so the signature changes rather than a second method.
- `OurTeam` exported from the existing unexported `ourTeam` for the default team.

### 5. Option pool and validation live in `internal/web/lineup.go`

Pool = roster records in file order, then starting-lineup records not on the roster (by ID). A
slot's options are the pool filtered by `Slot.Accepts`. A proposed ID outside the pool is unknown.
An ineligible proposed player is added as an extra selected option in that slot so a shared URL
shows what was proposed. Problems collect into a `[]string` rendered as a list; only an empty list
and nine filled slots call `FromSlots`, and its error, if any, is shown the same way. Option text is
`name · pos · team[ · game]`, with the game from `gameState` + `kickoffText` exactly as
`scoreColumn` builds a starter's meta line.

### 6. Template `lineup.html` — mirrors `matchup.html`

Its own `template.Must(ParseFS)` at init. Same body font, `max-width`, padding, `#ccc` card, and
33rem breakpoint. Phone-first: one slot per row, label above a full-width `<select>` at 16px (no iOS
zoom). Two GET forms: team picker (with `onchange="this.form.submit()"` plus a visible button so it
works without script) and the slots form with hidden `team` and "Check". Summary in a
`<textarea readonly>` for easy select-all on a phone. No refresh script, no version check, no
points: the no-margin rule holds trivially.

## Risks / Trade-offs

- [Option kickoffs are Central and unlabelled; `<option>` cannot hold a `<time>` for the page
  script to localise] → accepted; all three users are Central, same as the matchup page's no-JS text.
- [CSS duplicated from `matchup.html`] → small and stable; extracting a shared partial is a later
  refactor if a third page appears.
- [A stale roster CSV hides a newly acquired player] → known; roster transactions are out of scope,
  and an off-roster starter still shows.
- [Sleeper flips the week mid-week] → the page sets whatever week Sleeper reports (backlog).
