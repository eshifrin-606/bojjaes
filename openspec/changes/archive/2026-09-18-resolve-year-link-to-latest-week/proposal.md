## Why

The page is only addressable as `/{season}/{week}`. A reader who wants "this season's matchup"
has to already know the current week number; a bare `/2026` is a 404. With the week bar
(`add-week-navigation`) now showing the season and letting readers step through weeks, the season
itself is a natural link to land on — from a bookmark, a shared link, or typed by hand — and it
should resolve to something instead of erroring.

## What Changes

- A new route, `GET /{season}`, that redirects to `/{season}/{week}` for the highest week the
  lineup tree holds for that season. "Highest" is the greatest week number with a directory,
  not the highest that is also a well-formed matchup — the same "exists, not resolved" rule the
  week bar's neighbour links already use.
- A season with no week directories at all is a `404`, the same as a season/week pair that does
  not exist today.
- A season segment that is not an integer, or is outside the plausible range `internal/score`
  validates, is a `400`, the same as today's season/week check.
- Resolving the redirect reads only the lineup tree's directory listing. It never opens a roster
  file and never calls the stats provider.

Out of scope:

- Any change to `GET /{season}/{week}` itself, or to the week bar's prev/next links.
- A true `/` landing page across all seasons (still backlog).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `lineup-source`: adds *A season's latest existing week can be found without resolving it*. The
  tree reports the greatest week number it holds a directory for, in a season, without reading
  rosters or deciding whether that week is a matchup.
- `matchup-page`: adds *A season alone redirects to its latest week*.

## Impact

- `internal/lineup`: a new `Tree` method that scans a season directory and returns its highest
  week number, built on the same layout `weekDir` already encodes. Tested over `fstest.MapFS`.
- `internal/web`: a new handler for `GET /{season}` that validates the season, asks the tree for
  the latest week, and issues a redirect; `cmd/server/mux.go` registers the route.
- No change to scoring, the stats cache, the provider, or `internal/web/matchup.go`'s existing
  handler.
