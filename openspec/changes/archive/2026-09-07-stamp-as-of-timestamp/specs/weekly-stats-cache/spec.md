## MODIFIED Requirements

### Requirement: A cached entry records when its fetch completed

Each cached entry SHALL record the time its upstream fetch completed, and the cache SHALL make that
time available to a caller alongside the stats it returns, so freshness can be reported as the time
the data was fetched rather than the time the page was requested.

The recorded time SHALL be that of the fetch, not of any later read. Every caller served from one
entry — the caller whose request triggered the fetch, a caller served a cache hit, and a caller
that waited on an in-flight fetch — SHALL receive the same recorded time for that entry.

The plain stats-only read path SHALL remain available unchanged for callers that do not report
freshness.

#### Scenario: A cache hit is attributable to its fetch, not its read

- **WHEN** an entry is fetched and then read three minutes later through the freshness-reporting
  path
- **THEN** the time returned is the time of the fetch, not of the read

#### Scenario: Waiters on one fetch all get its completion time

- **WHEN** several requests wait on a single in-flight fetch and it completes
- **THEN** each waiter receives the stats and the same fetch-completion time

#### Scenario: The stats-only path is unaffected

- **WHEN** a caller reads a week through the path that returns only stats and error
- **THEN** it receives the same stats it would have before this change, with no freshness value and
  no change to when an upstream call is made
