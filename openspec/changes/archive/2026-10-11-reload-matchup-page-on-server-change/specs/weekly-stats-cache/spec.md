## ADDED Requirements

### Requirement: A week's fetch instant can be read without fetching

The cache SHALL let a caller ask for the fetch-completion instant of the entry it holds for one
season and week. Asking SHALL NOT make an upstream call, SHALL NOT start or wait on a fetch, and
SHALL NOT change what a later request for that week does.

The cache SHALL report that it holds no instant when the week has never been fetched, when its entry
has been invalidated, and while a fetch for it is still in flight.

An entry older than the TTL that has not been replaced SHALL still report its instant: expiry decides
when the next request fetches, not whether the last fetch happened.

Asking SHALL NOT write a log line. It is called far more often than a week is fetched, and the log is
a record of upstream calls.

#### Scenario: A fetched week reports its fetch instant

- **WHEN** week 4 is fetched and its fetch instant is then asked for
- **THEN** the instant returned is the one that fetch completed at, the same one a freshness-reporting
  read of that entry returns

#### Scenario: Asking makes no upstream call

- **WHEN** week 4 is fetched once and its fetch instant is then asked for many times
- **THEN** exactly one upstream call has been made

#### Scenario: A never-fetched week has no instant

- **WHEN** the fetch instant of a week that has never been requested is asked for
- **THEN** the cache reports that it holds none, and makes no upstream call

#### Scenario: An invalidated week has no instant

- **WHEN** week 4 is fetched, then invalidated, and its fetch instant is asked for
- **THEN** the cache reports that it holds none

#### Scenario: A week in flight has no instant yet

- **WHEN** a fetch for week 4 is in flight and its fetch instant is asked for
- **THEN** the cache reports that it holds none, without waiting for the fetch

#### Scenario: An expired entry still reports its instant

- **WHEN** week 4 is fetched and its fetch instant is asked for six minutes later, with no request for
  week 4 in between
- **THEN** the instant returned is that fetch's, and no upstream call is made
