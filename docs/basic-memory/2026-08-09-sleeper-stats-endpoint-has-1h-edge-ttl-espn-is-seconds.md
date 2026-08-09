---
title: 2026-08-09-sleeper-stats-endpoint-has-1h-edge-ttl-espn-is-seconds
type: note
permalink: bojjaes-memory/basic-memory/2026-08-09-sleeper-stats-endpoint-has-1h-edge-ttl-espn-is-seconds
tags:
- nfl-data
- espn
- sleeper
- provider
- adr-0002
- freshness
- caching
- tier-3
---

Partial resolution of Tier 3 (freshness) in the ESPN-vs-Sleeper probe. Full tables live in
`docs/probe-espn-sleeper.md` §3.0. This note records the reversal and the reasoning.

Freshness was assumed to be a non-issue and, if anything, a Sleeper advantage. Measuring HTTP
cache headers — which *is* possible in the offseason, unlike time-to-appear — suggests the
opposite. The measurement cost was one `curl` per endpoint.

## The reversal

The earlier probe concluded Sleeper had a freshness edge because it serves the whole league's
weekly stats in **one** call while ESPN needs ~26 calls per Sunday slate. That is a call-count
argument, and it does not survive contact with the cache headers.

## Observations

- [finding] Sleeper's weekly stats endpoint (`/v1/stats/nfl/regular/{yr}/{wk}`) is served from
  Cloudflare with `cache-control: public, s-maxage=3600, stale-while-revalidate=300`. Observed
  `cf-cache-status: HIT` with `age: 1739`. A **one-hour edge TTL**. #sleeper #freshness
- [finding] ESPN's live surfaces are configured for seconds: `nfl/scoreboard` is `max-age=8`,
  `nfl/summary?event=` is `max-age=1`. These are the values of a genuinely live feed. #espn
- [finding] ESPN's core `plays` endpoint returned `max-age=900, stale-while-revalidate=7200` via
  Varnish — but this was measured on a **completed** game, so it is probably a finished-game
  policy and must be re-measured in-progress. #espn #caveat
- [reversal] Fewer round trips does not beat a one-hour edge cache. If Sleeper's in-season TTL
  matches its offseason TTL, no polling strategy can get us inside the ~5-minute staleness
  budget from [[0002-live-scoreboard-backend]] — we would be up to **12× over budget**.
  #failed-assumption
- [caveat] Not proven yet. The 3600 was observed on *historical* weeks during the offseason, and
  Sleeper demonstrably tunes TTL per endpoint — `/v1/state/nfl` carries `s-maxage=60` — so the
  origin may well emit a much shorter TTL for the in-progress week. #open-question
- [method] This reframes Tier 3's most important test from an expensive one to a trivial one.
  Instead of a 60s polling loop measuring time-to-appear, the gating question is answered by a
  single `curl -I` against the **current week** during a live game: read `s-maxage`, `age`,
  `cf-cache-status`. Do the polling loop only after that passes. #method
- [correction] The Tier 3 protocol claimed Sleeper exposes no timestamp, so lag could only be
  measured by diffing successive polls. Wrong — the `age` header plus a weak `ETag` gives
  cache-level staleness for free. Better still, it **decomposes** the measurement: `age` is edge
  staleness, poll-diffing gives the remainder, which is Sleeper's own aggregation lag. #method
- [finding] Sleeper sources NFL data from **Sportradar**, per the community catalogue of its
  undocumented endpoints — corroborated by the 99.5% `sportradar_id` coverage found in Tier 4.
  Sportradar's own NFL API documents a **2-second TTL** once a game goes `inprogress`. So the
  upstream data is fast; any lag we observe is Sleeper's aggregation layer plus its CDN, not the
  underlying source. Mildly encouraging for a short in-season origin TTL. #sleeper
- [gotcha] Web search on this topic is actively misleading. The top results (sportsfirst.net,
  zuplo, sportsapis.dev) are generated API "guides" that assert "real-time updates" and "poll
  every 5–10 minutes" with no evidence and no measurement. Give them zero weight. Reading the
  response headers took less time than reading the articles and produced the only real data.
  #gotcha #method
- [anecdote] The one human data point located is a complaint on Sleeper's own forum that
  "scoring and stats updates take much longer than other platforms." Unverified, single source,
  but directionally consistent with the 3600 TTL.

## Consequence for the provider decision

Freshness was ranked first in [[0002-live-scoreboard-backend]]'s selection criteria, then
demoted by the probe on the grounds that both sources would clear a loose 5-minute bar. That
demotion now looks premature. Freshness should be treated as a **gate, not a tiebreaker**: if
Sleeper's in-season TTL is long, Sleeper-alone is disqualified no matter how well it scores on
rule coverage — and it currently wins on every other axis, so the collision would be sharp.

Note this stacks with the *other* known Sleeper gap, the 4-point FF turnover rule from
[[2026-08-09-sleeper-idp-ff-is-raw-idp-fum-rec-is-turnover-qualified]]. Both gaps point at the
same remedy — a narrow ESPN supplement — which strengthens the case for that hybrid, though on
different grounds than ADR 0002's rejected ID-bridge hybrid.

## Relations

- part_of [[2026-08-08-espn-vs-sleeper-stat-source-probe]]
- relates_to [[2026-08-09-sleeper-idp-ff-is-raw-idp-fum-rec-is-turnover-qualified]]
- supersedes_assumption_in [[0002-live-scoreboard-backend]]
- risk_accepted_by [[0003-sleeper-as-initial-stat-provider]]

## Outcome (2026-08-09)

The gate was **not** passed — it was consciously accepted.
[[0003-sleeper-as-initial-stat-provider]] chose Sleeper anyway, in order to ship a working
single-backend MVP, and downgraded ADR 0002's ~5-minute cache TTL from a design constraint to a
tuning knob (behind a long edge cache, a local 5-min cache is decoration). The served output
carries an as-of timestamp rather than implying live data.

Two follow-ups survive, in priority order: **find out whether a fresher Sleeper surface exists**
(the Sleeper app shows live scoring, so *some* endpoint is fresh — likely a per-league scores or
matchup surface rather than the global weekly dump; cache-busting is also untested), and
re-measure the in-progress-week TTL on the first live Sunday.

---

Informative, not authoritative. The offseason TTL is not the in-season TTL; confirm on the first
live Sunday before treating this as decided.
