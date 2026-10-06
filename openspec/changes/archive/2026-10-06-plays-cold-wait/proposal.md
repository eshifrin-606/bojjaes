## Why

The first read of a week with an empty play store scores forced fumbles from only the latest 300
plays. The whole-week fetch that fills the store finishes 3–12 s later. By then the stats cache has
stored the partial result, so turnover forced fumbles stay missing until the cache expires 5 minutes
later. Locally, every restart hits this, which makes the feature hard to debug. On Fly, the machine
scales to zero, so the first visit after any idle stretch hits it too.

## What Changes

- When the store holds no plays for a week, the stats fetch waits up to a configured duration for the
  whole-week fetch before scoring forced fumbles. If the fetch finishes in time, the page is complete
  on first load. If the wait runs out, behavior is today's: the poll's plays are scored and the
  whole-week fetch keeps running in the background.
- Only a cold week waits. Gap fills and post-game refreshes stay in the background.
- The wait comes from the `PLAYS_COLD_WAIT` environment variable as a Go duration (`30s`, `0`). Unset
  or empty means `30s`, the local default. `fly.toml` sets `0`, so the deployed app keeps today's
  behavior. A malformed or negative value stops the server at startup.
- This reverses the "never wait on a whole-week fetch" decision from `score-forced-fumbles` (D4), but
  only for a cold week and only up to a bound.

Out of scope: invalidating the stats cache when a background whole-week fetch completes. That is the
follow-up for fixing the 5-minute staleness when the wait runs out or is `0`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `forced-fumble-attribution`: a cold week's whole-week fetch may be waited on up to a configured
  bound. The "never blocks" requirement becomes "never blocks beyond that bound". A new requirement
  covers where the bound comes from.

## Impact

- `internal/sleeper/playstore.go`: `NewPlayStore` takes the cold wait. `ForcedFumbles` waits on the
  in-flight whole-week fetch when the week was cold.
- `cmd/server`: resolves `PLAYS_COLD_WAIT` (new file beside `addr.go` and `volume.go`) and passes it
  to `NewPlayStore`. Startup fails on a bad value.
- `fly.toml`: `PLAYS_COLD_WAIT = '0'` under `[env]`.
- Request latency: a cold local read can take up to ~30 s. Fly is unchanged.
- Depends on `score-forced-fumbles` being archived first, so that `forced-fumble-attribution` exists
  in `openspec/specs/` for this delta to modify.
