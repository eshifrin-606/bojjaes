## Why

A roster card that says `Colston Loveland` doesn't say he's a tight end, and the league has always
read rosters by position and team. ADR 0004 decision 9 settled that lineup files carry both as
display labels, frozen at the time the file is written. That's also why old weeks stay correct: a
team read live from Sleeper would show today's team on a 2025 page. The only open item was the
field order.

Adding columns also exposes a second problem. Today's format reads fields by position, so any later
column change breaks every old week at once. This change settles the column layout so future
columns can be added without that.

## What Changes

- **BREAKING** A roster file opens with a header row, `id,name,position,team`. Fields are read by
  column name, so column order is free. The header is the first line that is neither blank nor a
  comment.
- **BREAKING** Every record has all four fields. A header missing a column, naming an unknown
  column, or naming a column twice is refused, as is a record whose field count doesn't match the
  header. Because fields now split on every comma, a name containing a comma is refused.
- A position must be one of a closed set of Sleeper `position` values that can score in this league,
  including Sleeper's IDP values as Sleeper writes them (`DL`, `DE`, `LB`, `DB`, …). A team must be
  one of the 32 current NFL codes or `FA`. Anything else is refused, so a typo fails the file
  instead of reaching the page.
- `Record` gains `Position` and `Team`. They are labels on the same terms as the name: nothing
  identifies a player by them. No page model or template change — how they are shown is a later,
  separate change.
- All eight existing lineup files are backfilled in this change, with values read from Sleeper's
  `/v1/players/nfl` index when the files are written. 2025 teams may reflect later moves; that is
  accepted, since the app didn't exist then.
- ADR 0004 decision 9 is amended in place: the header row and column names, the closed sets, and a
  rule for future columns — a new column is either backfilled in the change that adds it or
  explicitly optional. The field-order open item and its backlog line are removed.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `lineup-source`: the file format becomes a header row plus four named columns. Position and team
  are required and checked against closed sets. The first-comma split and the comma-in-name
  allowance go away. The first-nine split still ignores position even though the file now carries
  one.

## Impact

- `internal/lineup/parse.go` and its tests; `Record` in the same file.
- `internal/lineup/read_test.go` and `internal/lineup/testdata/*.csv`, which are written in the old
  format.
- All eight files under `internal/lineup/data/`, and their `# … "id,name"` comment line.
- `TestEveryEmbeddedWeekIsAReadableMatchup` covers the new validation automatically, since it goes
  through `Tree.Read`.
- `docs/adr/0004-web-frontend-stack.md` and `backlog.md`.
- No change to `internal/web`, `internal/api`, scoring, or the Sleeper client. `scores.sh` and
  `scripts/teams/` are already gone, so the parser is the format's only reader.
