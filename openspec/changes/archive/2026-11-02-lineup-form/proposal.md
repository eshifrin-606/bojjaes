## Why

Setting a lineup today means editing a CSV on a laptop. Before any write path exists, the
co-managers need a phone page that builds a legal lineup for the current week, shows it as a
shareable URL to text each other, and hands back a CSV to commit. It also proves the form on its
own before a `POST` is bolted on (backlog: "Save from the lineup form").

## What Changes

- New page `GET /lineup?team=…`: team picker (the ten teams under `internal/data/rosters`, default
  `bojjaes`; changing it reloads). Season and week are labels from the current-week source; when no
  week is settable, or it is past NFL week 17 (the league's last), the page says there is no week to
  set.
- Nine `<select>`s in slot order (QB RB WR TE FLEX FLEX K D D), options limited to the team's roster
  players that slot accepts, each showing name, position, NFL team, and the player's game for the
  week (kickoff, live state, Final, or Bye).
- Slots start from the team's file for the current week, else its latest lineup before it, else
  empty.
- "Check" is a GET: the URL carries the proposed lineup as one parameter per slot
  (`?team=wood&qb=…&rb=…&wr=…&te=…&flex=…&flex=…&k=…&d=…&d=…`, Sleeper IDs). Loading such a URL
  renders those choices. An ineligible, duplicate, unknown, or missing player is an error on the
  page, never a 500.
- A complete, valid lineup also shows a plain-text summary to paste into a text, and a CSV download
  (`GET /lineup.csv`, same query) in canonical slot order.
- `lineup` package: read a team's roster; build a lineup from chosen slots; `LatestLineup` bounded
  to "at or before" a week (**BREAKING** for its signature; it has no production callers).
- No write path, no `POST`, no passphrase, no home page.

## Capabilities

### New Capabilities
- `lineup-form`: the `/lineup` page and `/lineup.csv` download — current week, team picker, slot
  selects, URL-encoded proposals, validation, summary, CSV.

### Modified Capabilities
- `lineup-source`: adds reading a team's roster file, building a lineup from per-slot choices, and
  finding a team's latest lineup at or before a week.

## Impact

- `internal/lineup` (new roster reader, `FromSlots`, slot exports, bounded `LatestLineup`).
- `internal/web` (new `lineup.go`, `lineup.html`, `CurrentWeekSource` interface).
- `cmd/server` (`newMux` gains the current-week source and rosters; `main` wires
  `weekcache.NewSingle` over `Client.CurrentWeek` and sets `V1BaseURL`).
- Upstream: one cached state read per hour plus the already-cached games read for the current week.
  No query parameter names a season or week, so the page cannot widen upstream keys.
