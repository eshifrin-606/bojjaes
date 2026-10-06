## Why

When a stats fetch succeeds, the only log line is the cache miss with its total duration. That total includes the aggregate fetch, the play-by-play poll and any cold-week wait, so you can't tell which one took the time. It also doesn't show whether a whole-week play fetch ran, why it ran, or when it finished. This change adds one more level of detail so that "why is a forced fumble missing or late?" can be traced from the logs.

## What Changes

- Log each successful weekly aggregate fetch with its row count and duration.
- Log each successful recent-plays poll with its play count and duration.
- Log when a whole-week play fetch starts, and the reason: `cold`, `gap` or `postgame`.
- Log each successful whole-week play fetch with its play count and duration.
- Failure logging, merge, attribution and the cold-week wait are unchanged.

## Capabilities

### New Capabilities
- `upstream-call-logging`: success-path log lines for Sleeper aggregate and play-by-play calls.

### Modified Capabilities

## Impact

- `internal/sleeper`: `Client` gains a `logf` field. `PlayStore.ForcedFumbles`, `startWholeFetch` and `fetchWholeWeek` emit the new lines.
- `cmd/server/main.go` passes `log.Printf` into `Client`.
- No API or behavior change.
