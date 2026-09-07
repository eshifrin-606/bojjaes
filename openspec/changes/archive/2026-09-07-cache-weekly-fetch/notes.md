# Notes

## Lock order and why single-flight cannot deadlock (task 4.6)

There are two things a goroutine can block on in `Cache.WeekStats`: the map mutex `c.mu`, and an
entry's `done` channel. They are always taken in that order and never held together. `c.mu` is
taken once on the way in, for the map lookup and — on a miss — the install of the in-flight entry,
and it is released before the goroutine does anything that can take time. A waiter releases the
mutex and only then blocks on `<-cached.done`; a leader releases the mutex and only then makes the
upstream call. The leader re-takes `c.mu` afterwards, to stamp the entry's fetch time and (on
failure) to remove it, and releases it again before `close(flight.done)`.

A waiter does not block on `done` alone: it selects on `done` and on its own `ctx.Done()`, so a
waiter that leaves takes only itself. That weakens nothing in the argument below — a waiter waking
early releases no lock it was holding and closes no channel anyone depends on — and it removes the
last way a caller could be pinned to a flight it no longer wants.

That gives the wait-for graph no cycle. A goroutine blocked on `done` holds no lock at all, so it
can never be the reason another goroutine fails to acquire `c.mu`, and in particular it can never
delay the one goroutine that will close the channel it is waiting on. A goroutine holding `c.mu` is
only ever doing map work — it never waits on a channel and never calls upstream — so the mutex is
held for a bounded, non-blocking span. Since the only edge from a channel wait to a mutex wait
cannot exist, and the only edge from a mutex hold to a channel wait cannot exist either, there is
nothing to close a cycle with.

The `done` channel is closed exactly once, by the leader that created it, on every path out of the
fetch including the failing one. Nothing else closes it and nothing sends on it, so a waiter's wake
depends on one goroutine that is itself blocked only on the upstream call. The leader is therefore the only caller that cannot walk
away, which is why its fetch runs under `context.WithoutCancel` and the cache's own
`fetchTimeout` rather than the caller's context: detaching it stops one reader's cancellation from
failing the others, and the timeout supplies the bound that detaching gave up. A leader whose
upstream never returns is a liveness problem for that key, not a deadlock, and one the timeout ends
after a bounded wait — readers of other keys are unaffected either way, because the mutex was
released before the fetch began.

## The failure mode section 5 fixed (task 5.1)

Of the three shapes task 5.1 anticipated — a stored error, a stored zero value, or a permanently
closed channel — section 4's single-flight produced the **stored zero value**, and it is the
quietest of the three. The leader stamped `fetchedAt` on its entry whatever the outcome and left it
in the map, so the next caller within the TTL found a complete, unexpired entry and took the hit
path, which returns `cached.stats, nil`. That caller got an empty `WeekStats` and **no error**: the
upstream failure had been converted into a plausible-looking week in which nobody scored. A stored
error would at least have been honest about having failed. The fix is the same either way — drop
the entry on failure before closing the channel — but the bug being silent is why 5.1's test
asserts the call count and the served fetch rather than only the returned error.

## Scenario coverage (task 8.3)

Every scenario in the `weekly-stats-cache` delta spec, and the test that carries it. The third
column is the strength of the claim, not a pass/fail: a test that failed for the behaviour it
names is stronger evidence than one that passed the moment it was written.

- **Genuine red** — the test failed against the previous section's implementation, for the
  behavioural reason the task predicted.
- **Verified red** — the test passed on first run (the section that would have made it fail had
  already landed). Rather than bank a possibly-vacuous pass, the production code was deliberately
  broken, the failure observed, and the code restored.

| Requirement | Scenario | Test | Claim |
| --- | --- | --- | --- |
| Short TTL | A second request within the TTL makes no upstream call | `TestSecondRequestWithinTTLIsServedFromCache` | Genuine red (2.1: reported two calls) |
| Short TTL | A request after the TTL fetches again | `TestRequestAfterTTLFetchesAgain` | Genuine red (3.2: reported one call) |
| Short TTL | An entry is live up to its expiry and stale after it | `TestEntryLivesUpToTheTTLAndNotBeyond` | Verified red (3.4) |
| Short TTL | Different weeks do not share an entry | `TestDifferentWeeksDoNotShareAnEntry` | Verified red (2.3: key narrowed to one field) |
| Single-flight | Simultaneous misses share one fetch | `TestSimultaneousMissesShareOneFetch` | Genuine red (4.1: reported N calls) |
| Single-flight | A slow fetch does not block another week | `TestASlowFetchDoesNotBlockAnotherWeek` | Verified red (4.3) |
| Single-flight | A later request after the flight completes is a cache hit | `TestARequestAfterTheFlightCompletesIsAHit` | Verified red (4.5) |
| Failures | A failure is returned, not stored | `TestAFailureIsReturnedNotStored` | Genuine red (5.1: the stored zero value, above) |
| Failures | A recovered upstream serves the next request | `TestASuccessAfterAFailureIsCached` | Verified red (5.4) |
| Failures | A failed flight fails all its waiters | `TestAFailedFlightFailsAllItsWaiters` | Verified red (5.3) |
| No self-initiated fetch | An idle cache makes no calls | `TestIdleCacheMakesNoCalls` | See below |
| Cancellation | The first caller leaving does not fail the others | `TestTheLeaderLeavingDoesNotFailTheWaiters` | Genuine red (6.1) |
| Cancellation | A waiter's cancellation is its own | `TestAWaitersCancellationIsItsOwn` | Genuine red (6.3: a hang) |
| Fetch time recorded | A cache hit is attributable to its fetch, not its read | `TestEntryRecordsItsFetchTimeNotItsReadTime` | Verified red (3.5) |

No scenario is uncovered. Two claims are weaker than the table alone suggests, and are worth
naming:

**"An idle cache makes no calls" is the weak one.** It constructs a cache, advances the clock an
hour, and asserts the call count is what it was. That is an assertion about an *absence*, and the
only implementation it can fail against is one that already polls — so it is a regression guard for
a sweeper or a pre-warmer someone adds later, not evidence about the code as written. The real
evidence for that requirement is task 3.6: reading the package back and confirming there is no
goroutine at all. A test cannot prove that; only the read can.

**The timeout clause of the cancellation requirement has one test, added last.**
`TestAFetchThatOutlastsTheTimeoutFailsAndIsNotCached` (task 6.5) covers prose in the requirement
body rather than a numbered scenario: a fetch outliving `fetchTimeout` fails its callers and leaves
nothing cached, and the next call retries. It passed on first run, because the timeout arrived in
6.2 alongside the detachment it exists to bound. Replacing `context.WithTimeout` with a plain
`context.WithCancel` was confirmed to hang the test until the harness deadline fired — which is
also the clearest statement of why the bound is there: a detached fetch has nothing else left to
stop it.

## What this opens up next (task 8.5)

**The as-of timestamp is now the strongest next line.** Before this change a page was fetched when
it was requested, so "now" was an honest enough answer to *when is this from*. A cache is precisely
what makes a page able to be quietly stale: a reader can be served data up to five minutes old with
nothing on the page saying so, and the refresh timer makes that the common case rather than the
edge one. The hook already exists — `entry.fetchedAt` is stored and expiry is measured from it —
and this change deliberately renders nothing from it. Threading it out to the handler and onto the
page is the work; the decision it depends on was made here.

**The rate-limit probe is still open**, and this change did not settle it. It bounds volume without
knowing what the bound needs to be: the TTL was chosen for how stale a live week's scores may
acceptably be, not from any measurement of what Sleeper tolerates. The two are independent — a
probe could show five minutes is far more conservative than it needs to be, or that the ceiling is
about concurrent weeks rather than rate. Either would be a reason to revisit the number, not the
mechanism.

## Live verification (tasks 7.3, 7.4)

Measured against the real Sleeper API with the server running locally. All figures are
`curl -s -o /dev/null -w '%{time_total}'` against `GET /{season}/{week}`.

**7.3 — the TTL holds, and the miss is much cheaper than the task assumed.**

| | week 15 | week 16 |
| --- | --- | --- |
| cold (fetch) | 35.5 ms | 32.4 ms |
| second (hit) | 0.50 ms | 0.35 ms |
| third (hit) | 0.34 ms | 0.33 ms |

Expiry, on week 16: a hit at 0.54 ms, then nothing for five minutes and twenty seconds, then
**132 ms** — refetched — and 0.60 ms on the request after it. The 132 ms is higher than the 32 ms
cold figure because the idle connection needed a fresh TLS handshake; it matches the ~115 ms a bare
`curl` to `api.sleeper.app` costs, which is what a genuinely cold fetch is worth.

The task's original wording — "first load slow, the rest instant" — was a bad instrument and is
corrected in `tasks.md`. A ~35 ms miss is well under a page paint. The cache is not detectable by
eye and was never going to be; the first attempt at this verification concluded "it's fast from the
beginning" on a week the browser had already warmed. The signal is the ~100× ratio, not the
sensation.

**7.4 — the two transports share one cache.** With week 14 cold, `scripts/scores.sh 2025 14`
(`POST /scores`) ran first at 50 ms, and the *page* for the same week — which had fetched nothing
itself — returned in **0.58 ms**. A second `scores.sh` took 30 ms, essentially all bash and curl
startup. One wrap in `main`, one upstream budget, whichever transport arrives first.
