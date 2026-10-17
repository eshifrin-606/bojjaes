## ADDED Requirements

### Requirement: A week's games are cached under the same rules as its stats

A fetch of a season and week's NFL games SHALL be cached by that season and week under every rule
this capability gives a stat fetch: the time-to-live, the collapse of concurrent misses into one
upstream call, a failure not being stored, no fetch without a request, a caller's cancellation not
cancelling the shared fetch, and the recorded fetch instant.

A week's games and a week's stats SHALL be separate entries. Fetching, expiring, or invalidating one
SHALL NOT fetch, expire, or invalidate the other.

#### Scenario: A second request for a week's games makes no upstream call

- **WHEN** the same season and week's games are requested twice, one minute apart
- **THEN** the provider's game feed is called once and both callers receive the same games

#### Scenario: Games expire on the same time-to-live

- **WHEN** the same season and week's games are requested twice, six minutes apart
- **THEN** the provider's game feed is called twice

#### Scenario: A failed game fetch is not cached

- **WHEN** a game fetch fails and the same season and week's games are requested again at once
- **THEN** the second request makes its own upstream call

#### Scenario: Invalidating stats leaves games alone

- **WHEN** a week's stats and games are both cached and the week's stats are invalidated
- **THEN** the next request for that week's games is served from the cache with no upstream call
