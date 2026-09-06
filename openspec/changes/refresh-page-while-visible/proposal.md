## Why

The matchup page is a snapshot. Once `/2025/15` is loaded, it shows the scores as of that request
forever — which is exactly the failure mode [ADR 0004](../../../docs/adr/0004-web-frontend-stack.md)
warns about: a terminal report is obviously something the reader just produced, but a page left
open on a phone during a Sunday afternoon has no such anchor and quietly goes stale.

ADR 0004 decision #5 settled the fix: the client polls, the server caches. This change is the
client half — the page re-fetches roughly every five minutes, and only while the tab is visible, so
upstream load is bounded by who is actually looking rather than by how many tabs are open.

## What Changes

- The matchup page re-fetches itself roughly every five minutes by reloading, in a few lines of
  vanilla JS inline in `matchup.html`. No framework, no build step, no fetch-and-swap.
- The refresh only happens while the tab is visible: a hidden tab makes no request, and a tab left
  hidden for an hour makes one request when the reader comes back, not twelve.
- Returning to a tab that has been hidden past the refresh interval reloads immediately rather than
  waiting out the remainder of the timer, so what the reader sees on returning is never older than
  the interval.
- The script interpolates nothing. No roster text, no team name, no score reaches the `<script>`
  element — the refresh interval is the only value in it, and it is a literal.

Out of scope, each already its own backlog line:

- **The as-of timestamp.** Refreshing tells the reader the page is trying to stay current; the
  timestamp is what tells them whether it succeeded. They are complementary, and the timestamp is
  the larger change of the two because it also changes what `matchup-page` renders. This change
  leaves the page with no freshness indicator, which is the state it is in today.
- **The TTL/single-flight cache over the Sleeper fetch.** Until it lands, every refresh is a
  Sleeper fetch, so three open tabs are three upstream calls every five minutes. That is a real
  increase over today's on-demand-only load and it is worth being explicit about — but at three
  viewers it is roughly one call a minute at peak, well under anything we would expect to be rate
  limited for, and the cache is the next backlog line under "Not hammering Sleeper".
- Fetch-and-swap, `<meta http-equiv="refresh">`, server-sent events, and any form of push.

The page must still imply no winner. That is a constraint on what this change may add, not a
feature it delivers.

## Capabilities

### New Capabilities

None. The refresh is a property of the page that already exists.

### Modified Capabilities

- `matchup-page`: gains a requirement that the served page refreshes itself on an interval while
  its tab is visible, and refreshes on return from a hidden tab when the interval has elapsed. The
  existing requirements — the URL, the two columns, the totals, the absent-starter placeholder, the
  no-winner rule, HTML escaping — are unchanged, and the no-winner rule now also binds the script.

## Impact

- `internal/web/matchup.html`: one `<script>` element. No change to the markup, the CSS, or the
  view model.
- `internal/web/matchup.go`: no change expected. The refresh interval is a template literal, not a
  field, unless pinning it in a test reads better as a rendered constant.
- `internal/web/matchup_test.go`: assertions on the rendered script — that it is present, that it
  is guarded by the tab's visibility, and that no roster text reaches it. What the browser actually
  does with the timer is confirmed by hand against a running server; a Go test cannot run it.
- No change to `cmd/server/main.go`, `POST /scores`, the roster CSV format, `scripts/**`, or the
  Sleeper client. Upstream request volume changes, but no code on the fetch path does.
