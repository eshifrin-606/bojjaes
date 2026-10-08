## ADDED Requirements

### Requirement: A week's entry can be invalidated before its TTL

The cache SHALL let a caller invalidate one season and week. After invalidation, the next request
for that season and week SHALL make an upstream call, even if the TTL has not elapsed. Invalidation
itself SHALL NOT make an upstream call, and SHALL NOT affect any other season and week.

Invalidating a season and week that has no entry SHALL do nothing.

#### Scenario: A request after invalidation fetches again

- **WHEN** week 4 is fetched, then invalidated one minute later, then requested again
- **THEN** the upstream is called twice and the second caller receives the second fetch's stats

#### Scenario: Invalidation makes no call by itself

- **WHEN** week 4 is fetched and then invalidated, and no further request is made
- **THEN** exactly one upstream call has been made

#### Scenario: Other weeks keep their entries

- **WHEN** weeks 4 and 5 are fetched and week 4 is invalidated
- **THEN** a request for week 5 within the TTL is served from the cache

#### Scenario: Invalidating an absent week is harmless

- **WHEN** week 4 has never been fetched and is invalidated
- **THEN** nothing happens, and the next request for week 4 fetches as usual

### Requirement: A fetch in flight when its week is invalidated is not cached

If a season and week is invalidated while a fetch for it is in flight, that fetch's result SHALL
still be returned to the callers waiting on it, but SHALL NOT be stored. The next request for that
season and week SHALL make a new upstream call.

That fetch finishing, whether it succeeds or fails, SHALL NOT remove or replace an entry installed
for the same season and week after the invalidation.

#### Scenario: An invalidated flight still answers its waiters

- **WHEN** a fetch for week 4 is in flight, week 4 is invalidated, and the fetch then completes
- **THEN** the callers waiting on it receive its stats

#### Scenario: An invalidated flight's result is not served later

- **WHEN** a fetch for week 4 is in flight, week 4 is invalidated, the fetch completes, and week 4 is
  requested again within the TTL
- **THEN** that request makes a new upstream call

#### Scenario: A stale flight does not clobber a newer one

- **WHEN** a fetch for week 4 is in flight, week 4 is invalidated, a second request starts a new
  fetch, and the first fetch then fails
- **THEN** the second fetch's result is still stored when it completes, and later requests within
  the TTL are served from it
