## Context

A successful miss logs one line today: `statscache miss: … fetched in …`. Within `internal/sleeper`, a cache miss runs the aggregate fetch, the recent-plays poll, an optional background whole-week fetch and an optional cold-week wait. On success, none of those steps log anything.

## Goals / Non-Goals

**Goals:** one log line per upstream call on the success path, plus a line for whole-week fetch starts. Together these show the sequence of aggregate and play-by-play activity within a miss.

**Non-Goals:** merge counts (new/updated), attribution outcomes or reasons, cold-wait outcome, cache-hit logging, structured logging (`slog`).

## Decisions

- **Emit each line where the fact is known**, through the existing `logf` seam. `Client` gets a `logf` field for the aggregate line. Returning timings up to the cache was rejected because the cache must not know provider steps.
- **Reason is computed in `ForcedFumbles`** from the existing predicates: `cold` (nothing held), then `gap` (`needsWholeFetch` with plays held), then `postgame`. It is passed to `startWholeFetch`, which logs only when it actually starts a flight. Joining a running fetch stays silent.
- **Timing uses `PlayStore.now` for plays**, so tests can pin durations. The `Client` uses `time.Now` because tests assert only the line's shape, not the duration value.
- **Line format** is `sleeper {stats|plays poll|whole-week} {season} w{week}: …`. Season and week are the correlation key; single-flight allows at most one miss per week at a time.

## Risks / Trade-offs

- [Each miss logs about 3–4 extra lines] → bounded by the TTL, the same reasoning the miss line already uses.
- [A nil `logf` on `Client` would panic] → treat nil as silent, so existing test constructions keep working.
