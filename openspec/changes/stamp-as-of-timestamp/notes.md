# Notes

## Why the stamp is the fetch time, not when the stats moved

The page says "Sleeper stats fetched <time>" and means exactly that: the instant our server's
upstream call completed. That is deliberately *not* the same as when the underlying stat line
changed on Sleeper. Two lags sit between them:

- **Upstream lag.** Sleeper's weekly aggregate updates some seconds-to-minutes after a play is
  scored and stat-corrected. We have no visibility into that and no timestamp from them for it.
- **Cache lag.** `weekly-stats-cache` serves an entry for up to a five-minute TTL. A play that
  landed 30 s after our last fetch is invisible on the page until the next miss, up to ~5 min
  later.

The honest thing the server can state is its own fetch time, so that is what the label commits to.
An "as of when the game data actually moved" line would need a per-stat upstream timestamp we do
not have, and would over-promise currency the cache cannot deliver.

## Deferred: live-Sunday drift observation

Not code. To be run by the maintainer on a Sunday with games in progress, once this change is
deployed:

1. Open the matchup page for the live week and note its `<time datetime>` value.
2. Pick a starter whose game is live. Watch Sleeper's own app/site for a scoring play that changes
   that player's line.
3. When the play lands, record wall-clock time. Reload the page (or wait for the 5-min auto-reload)
   and record when the new points first appear and what the page's as-of says at that moment.
4. drift = (time points appeared on our page) − (time the play landed on Sleeper). Repeat for a
   handful of plays across the afternoon.

**What it feeds:** the still-open backlog lines "Probe Sleeper's rate-limit tolerance at the
cadence we're actually going to deploy at" and the TTL choice in
[[0004-web-frontend-stack]] decision #5. If observed drift is dominated by the 5-minute cache lag
rather than upstream lag, that is evidence the TTL is the knob to turn; if upstream lag is large
and variable, no TTL change helps and the label wording is doing the right job by saying "fetched".

Record the numbers here when gathered, then promote a conclusion to the probe change or an ADR
amendment — this note is not authoritative.

## Layout decisions the spec left open

`matchup-page`'s new requirement fixes the constraints (one `<time>` element, RFC 3339 in
`datetime`, `America/Chicago` visible text with the zone shown, labelled "fetched", never "live" /
"current", outside both columns). The exact visible layout string is settled in the red-green loop
(tasks 4.1–4.2) against a readable assertion and recorded here once chosen.

Related: [[weekly-stats-cache]] (stores `entry.fetchedAt`), [[refresh-page-while-visible]] (makes a
stale tab the common case), [[matchup-page]].
