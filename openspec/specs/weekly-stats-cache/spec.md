# weekly-stats-cache Specification

## Purpose

Bound how often a weekly stat feed is fetched from upstream: a fetched week is served from memory
for a short time-to-live, and concurrent misses for the same week collapse into one upstream call.

This capability exists because readers stopped deciding when a fetch happens. A page that refreshes
itself on a timer makes upstream volume scale with the number of open tabs, so the bound has to come
from the server. It governs only *when an upstream call is made* — the TTL, the collapse of
concurrent misses, what happens to a failure, and whose cancellation may stop a fetch. It does not
govern what is fetched, how a stat line is transformed, or how anything is displayed: the stats are
`player-week-score`'s and the page is `matchup-page`'s, and both are unchanged by a cache sitting
behind them.

Freshness here is pull-driven and unannounced. Nothing refreshes ahead of expiry, so with no readers
there is no upstream load; and a cached page carries no indication of its age, which is what makes
reporting the fetch time the natural next capability rather than a decoration.
## Requirements
### Requirement: A fetched week is served from cache for a short TTL

A weekly stat fetch SHALL be cached by the season and week it was fetched for, and a subsequent
request for the same season and week within the time-to-live SHALL be answered from the cache
without any upstream call. The time-to-live SHALL be approximately five minutes.

Upstream request volume is thereby bounded by the TTL and the number of distinct weeks being read,
rather than by the number of readers or how often their pages refresh.

An entry SHALL expire strictly on its age: once the TTL has elapsed since the fetch completed, the
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

### Requirement: Concurrent misses for one week collapse into one upstream call

When several requests for the same season and week arrive while no cached entry exists and a fetch
for that key is already in flight, exactly one upstream call SHALL be made. Every waiting caller
SHALL receive the result of that single call.

A request for a *different* season and week SHALL NOT be blocked by a fetch in flight for another
key.

#### Scenario: Simultaneous misses share one fetch

- **WHEN** ten requests for the same season and week arrive while the upstream is still responding
  to the first
- **THEN** the upstream is called exactly once and all ten callers receive that call's stats

#### Scenario: A slow fetch does not block another week

- **WHEN** a fetch for week 15 is in flight and a request for week 16 arrives
- **THEN** week 16's fetch begins without waiting for week 15's to finish

#### Scenario: A later request after the flight completes is a cache hit

- **WHEN** a request arrives for a season and week whose in-flight fetch has just completed
- **THEN** it is served from the cached entry with no further upstream call

### Requirement: A failed fetch is not cached

If the upstream call fails, the failure SHALL be returned to every caller waiting on that fetch and
SHALL NOT be stored. The next request for that season and week SHALL attempt a fresh fetch.

Remembering a failure would turn a momentary upstream problem into a fixed window of errors on a
page that refreshes itself, long after the upstream had recovered.

#### Scenario: A failure is returned, not stored

- **WHEN** a fetch fails and the same season and week is requested again immediately
- **THEN** the second request attempts a new upstream call rather than returning the stored failure

#### Scenario: A recovered upstream serves the next request

- **WHEN** a fetch fails and the next request for the same season and week succeeds
- **THEN** that caller receives stats and the successful result is cached

#### Scenario: A failed flight fails all its waiters

- **WHEN** several requests wait on one in-flight fetch and that fetch fails
- **THEN** every waiter receives an error and none receives stats

### Requirement: The cache never fetches on its own

The cache SHALL initiate an upstream call only in service of a caller's request. It SHALL NOT poll,
schedule, refresh ahead of expiry, or pre-warm any key.

Freshness is pull-driven: with nobody reading, upstream load is zero.

#### Scenario: An idle cache makes no calls

- **WHEN** an entry is fetched and then no further request is made for an hour
- **THEN** exactly one upstream call has been made and the entry is not refreshed on expiry

### Requirement: A caller's cancellation does not cancel the shared fetch

The upstream call SHALL run under a context detached from the requesting caller's, bounded by the
cache's own timeout. A caller abandoning its request SHALL NOT cancel a fetch that other callers are
waiting on.

A caller whose own context is cancelled while waiting SHALL observe that cancellation and stop
waiting, without affecting the fetch or the other waiters.

#### Scenario: The first caller leaving does not fail the others

- **WHEN** the caller that started a fetch cancels its context while other callers wait on the same
  fetch
- **THEN** the fetch completes and the remaining callers receive its stats

#### Scenario: A waiter's cancellation is its own

- **WHEN** one waiting caller's context is cancelled
- **THEN** that caller returns a cancellation error and the other waiters still receive the fetch's
  stats

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

