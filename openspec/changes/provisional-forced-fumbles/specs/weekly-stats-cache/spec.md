## MODIFIED Requirements

### Requirement: A fetched week is served from cache for a short TTL

A weekly stat fetch SHALL be cached by the season and week it was fetched for, and a subsequent
request for the same season and week within the time-to-live SHALL be answered from the cache
without any upstream call. The time-to-live SHALL be approximately five minutes, except for a
fetched week whose stats are marked provisional, whose time-to-live SHALL be approximately fifteen
seconds.

Upstream request volume is thereby bounded by the TTL and the number of distinct weeks being read,
rather than by the number of readers or how often their pages refresh. The short provisional TTL
adds at most a few upstream calls per week. It lasts only while a provisional week's background
fill runs, and that fill has a timeout.

An entry SHALL expire strictly on its age: once its TTL has elapsed since the fetch completed, the
next request for that key SHALL fetch afresh.

#### Scenario: A second request within the TTL makes no upstream call

- **WHEN** the same season and week are requested twice, the second request one minute after the
  first
- **THEN** the upstream is called once and both callers receive the same week's stats

#### Scenario: A request after the TTL fetches again

- **WHEN** the same season and week are requested twice, the second request six minutes after the
  first
- **THEN** the upstream is called twice and the second caller receives the second fetch's stats

#### Scenario: An entry is live up to its expiry and stale after it

- **WHEN** a request arrives just before the TTL has elapsed and another just after
- **THEN** the first is served from the cache and the second causes an upstream call

#### Scenario: Different weeks do not share an entry

- **WHEN** week 15 is fetched and then week 16 is requested
- **THEN** week 16 causes its own upstream call and week 15's cached entry is unaffected

#### Scenario: A provisional week expires quickly

- **WHEN** a fetch returns provisional stats and the same season and week are requested again twenty
  seconds later
- **THEN** the upstream is called again

#### Scenario: A provisional week is still cached briefly

- **WHEN** a fetch returns provisional stats and the same season and week are requested again five
  seconds later
- **THEN** the second request is served from the cache with no upstream call

#### Scenario: A non-provisional refetch restores the full TTL

- **WHEN** a provisional entry expires and the next fetch returns stats that are not provisional
- **THEN** that entry is served from the cache for approximately five minutes
