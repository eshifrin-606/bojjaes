## Context

The matchup page (`GET /{season}/{week}`) is deployed and is now the UI. Two things built for the
earlier terminal UI are still here:

```
scripts/scores.sh ◀── scripts/fantasycast.sh
      │ curl
      ▼
POST /scores ── internal/api ──┐
                               ├── cmd/server/mux.go  (statsSource = api.StatsSource + web.StatsSource)
GET /{season}/{week} ── web ───┘                │
                                   internal/statscache.Cache
                                     WeekStats      ← only internal/api
                                     WeekStatsAsOf  ← internal/web
```

Afterwards:

```
GET /{season}/{week} ── internal/web ── mux.go (web.StatsSource) ── statscache.Cache.WeekStatsAsOf
```

Almost all of this is deletion. The work that needs thought is (a) keeping any rule that is still
live but was only stated on something being deleted, and (b) editing docs so nothing describes the
scripts as a current constraint.

## Goals / Non-Goals

**Goals:**
- No shell scripts, no JSON scoring endpoint, and no production code whose only caller was either.
- Every rule the page still depends on stays normative in a spec that remains.
- Docs describe one UI.

**Non-Goals:**
- Changing the roster CSV format. The backlog item gets simpler (one reader) but stays open.
- Changing `internal/score`, `internal/sleeper`, `internal/lineup`, or `internal/web` behaviour.
- Replacing the endpoint with another machine-readable surface.

## Decisions

### Remove `Cache.WeekStats` rather than keep it as a delegate

Its doc comment names `internal/api` as its only caller. Keeping it would leave an untested-in-prod
path that future cache changes would still have to preserve. Its tests, about 40 call sites in
`cache_test.go`, move to `WeekStatsAsOf` and ignore the instant with `_`. This is a mechanical
rewrite that keeps each test's assertion. Every behaviour those tests cover (single-flight, TTL,
error handling, context detachment) lives in `WeekStatsAsOf` already, because `WeekStats` is a thin
delegate.

Alternative considered: keep `WeekStats` for test ergonomics. Rejected, because test convenience is
not a reason to keep production API.

`statscache.StatsSource` (the upstream interface `sleeper.Client` satisfies) keeps its own
`WeekStats` method. That is a different interface and is not affected.

### `mux.go` depends on `web.StatsSource` directly

The combined local `statsSource` interface existed only to join two consumers' interfaces. With one
consumer, `newMux` takes `web.StatsSource`. The `stubStats` in `mux_test.go` drops its `WeekStats`
method.

### Move the regular-season rule; don't delete it

"Season and week are interpreted as the regular season" was stated only inside the removed
"Multi-player score endpoint" requirement. The page depends on it through `sleeper.FetchWeekStats`
(the `/v1/stats/nfl/regular/` URL). It is re-added in `player-week-score` as its own requirement,
attached to the stat fetch rather than to any route.

### Retire whole capabilities instead of rewriting them for the page

`lineup-score-report` and `matchup-report` describe terminal output (fixed-width columns, BENCH
heading, argument order). The rules that matter beyond the terminal are already in `lineup-source`
(first nine are starters) and `matchup-page` (no-stats shows no number and adds nothing, one total,
no margin). Each REMOVED entry names where its surviving rule lives. After archive, the two spec
directories are deleted rather than left with empty requirement lists.

### The 405 scenario moves to the matchup route

The mux requirement was really about method-qualified patterns and the owned mux. `POST
/2025/15` exercises the same property. Go's mux answers a GET-only pattern's wrong method with 405 and
`Allow: GET, HEAD`. The test pins the 405 and that the handler was not invoked. It does not pin the
exact `Allow` string, which is the standard library's formatting. A separate scenario pins that
`POST /scores` no longer returns a score. Its expected status is 404, because `/scores` is one
segment.

### Purpose sections are edited by hand

Delta specs cannot change a `## Purpose`. Four of them mention retired things:
- `player-week-score`: "through to an HTTP response"
- `matchup-page`: the `matchup-report` sentence, and "scoring endpoints" in the range rule
- `lineup-source`: "belongs to `lineup-score-report`"
- `http-server-lifecycle`: "the scores are `player-week-score`'s"

These are a task after the specs sync.

### Docs are amended in place, ADR included

Per the project's doc-consolidation preference. ADR 0004's constraints that read as live ("scripts
keep working", "the CSV change touches `scores.sh`", the follow-up naming both readers) are edited
so they no longer bind. Where the scripts serve as historical context ("the scoreboard worked only
as a terminal artifact"), past tense is enough. The no-margin rationale stays and is attributed to
the page's own reasoning rather than to `fantasycast.sh`.

## Risks / Trade-offs

- [Something outside the repo calls `POST /scores` on the deployed app] → Only the scripts are known
  to call it, and it was documented as not a public API. A caller would get a 404 with no fallback.
  Accepted.
- [A rule silently disappears with a removed spec] → Every REMOVED entry names its surviving home or
  says "None" with a reason. The regular-season rule was found this way and moved.
- [Rewriting ~40 cache test call sites hides a behaviour change] → The rewrite is mechanical (`x,
  err :=` becomes `x, _, err :=`). Run the full suite before and after the rewrite with `WeekStats`
  still present, then delete `WeekStats` and confirm it still compiles and passes.
- [Stale `scripts/**` references in the active `refresh-page-while-visible` change] → Harmless; that
  change's claim ("untouched") stays true. No edit.

## Migration Plan

This is a single deploy. Rollback is a revert of the merge commit, which restores the endpoint and
scripts together.
