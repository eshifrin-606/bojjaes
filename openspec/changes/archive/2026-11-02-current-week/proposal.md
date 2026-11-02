## Why

The lineup form (a later change) must know which week it is setting a lineup for. Today nothing in
the server knows the current NFL week; the backlog's "Current NFL week" row asks for it from
Sleeper's state endpoint, cached, with the off/preseason and postseason meaning there is no settable week. When the league's
own season ends (week 17) is a league rule, not an NFL fact, and is left to the lineup form.

## What Changes

- Fetch the current NFL season, week, and season phase from Sleeper
  `GET https://api.sleeper.app/v1/state/nfl`, translated inside `internal/sleeper` to a
  provider-neutral domain value.
- Add a domain value that answers one question: is there a settable week, and if so which season
  and week. Any regular-season week is settable.
- Cache the answer for an hour, reusing `internal/weekcache`'s single-flight and failure handling.
- Log each successful state fetch and each state cache hit, alongside the existing upstream lines.
- No route, page, form, or `cmd/server` wiring: the lineup form change wires and consumes it.

## Capabilities

### New Capabilities
- `current-nfl-week`: reading the current NFL season, week, and phase from the provider, deciding
  whether a week is settable, and caching the answer.

### Modified Capabilities
- `upstream-call-logging`: adds a line for each successful state fetch and each state cache hit.

## Impact

- New package `internal/nflweek` (domain type, no imports beyond the standard library).
- `internal/sleeper`: a second host constant and `Client` field for the v1 API, a new
  `state_endpoint.go`, and a `Client.CurrentWeek` method.
- `internal/weekcache`: an unkeyed wrapper over `Cache` and a one-hour TTL constant; the log line
  format of keyed caches is unchanged.
- `docs/package-dependencies.md`: add `internal/nflweek` to the map.
- One more Sleeper call, at most once an hour per process.
