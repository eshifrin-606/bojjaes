## Why

When a whole-week play fetch lands after the 500ms cold wait, the server drops the week's cached
stats, but nothing tells an open page. The corrected scores are ready seconds after the first render
and the reader does not see them until the five-minute timer fires or they refresh by hand.

The same gap applies to anything else that invalidates a week's stats, now or later. The page needs
one general way to learn "what you are showing has been superseded".

## What Changes

- The matchup page carries a **stats version**: an opaque token naming the fetch its scores came
  from, rendered as an attribute outside the script.
- A new endpoint, `GET /{season}/{week}/version`, reports the version the server currently holds for
  that week. It never fetches from the provider.
- While its tab is visible, the page checks that endpoint and reloads itself, as a plain reload of
  the same URL, when the server's version differs from its own.
- **Server-directed cadence**: the page render and every check response say how many seconds to wait
  before the next check, or to stop checking. For now the server gives one default everywhere.
- The stats cache gains a read-only way to report a week's fetch instant without fetching.
- The existing five-minute reload and the reload on returning to a stale tab are unchanged.

Not in this change: varying the cadence by week or by game state, time or idle caps on checking,
backoff, server-sent events, and in-place markup updates.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `matchup-page`: the self-refresh requirement now permits a version check request; new requirements
  cover the stats version on the page, the version endpoint, the check-and-reload behaviour, and
  server-directed cadence.
- `weekly-stats-cache`: a week's fetch instant can be read without causing an upstream call.

## Impact

- `internal/statscache`: one new read-only method on `Cache`.
- `internal/web`: new version handler, two new attributes in `matchup.html`, an extended script.
- `cmd/server/mux.go`: one new route; `newMux` needs a source for the version.
- `cmd/server/server.go`: unchanged. Checks are short requests, so the timeouts stand.
- Fly: a visible tab now makes a request every interval, so the machine stays running while someone
  is watching. The ceiling is the machine running all month, about $2.
- Logs: one request per visible tab per interval reaches the server; the cache does not log them.
