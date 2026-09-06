# Notes

## Scenario coverage (task 6.3)

Each scenario in `specs/matchup-page/spec.md`, and what covers it. Go tests can only assert on the
rendered body; the three behaviours that exist only in a browser are covered by section 5's hand
verification and by nothing else.

| Scenario | Covered by |
| --- | --- |
| A visible tab refreshes on the interval | `TestThePageCarriesARefreshScript`, `TestTheRefreshIntervalIsFiveMinutes`, `TestTheRefreshIsGuardedByTabVisibility` pin the rendered contract; **task 5.1** observes the refresh actually happening. |
| A hidden tab makes no request | `TestTheRefreshIsGuardedByTabVisibility` pins the guard; **task 5.2** observes the silence in the server log. |
| An hour hidden is not an hour of requests | The fire-time guard (task 2.7 confirmed no start/stop bookkeeping, so throttled background ticks find `hidden` and do nothing); **tasks 5.2 and 5.3** observe that a long absence produces no burst on return. |
| A tab hidden past the interval refreshes on return | `TestTheScriptListensForTheTabComingBack`, `TestReturningToTheTabRefreshesOnlyWhenStale`; **tasks 5.3 and 5.6** observe it, 5.6 on a phone. |
| A brief switch away does not refresh | `TestReturningToTheTabRefreshesOnlyWhenStale` (the elapsed-time condition); **task 5.4** observes no reload on a thirty-second round trip. |
| A roster name never reaches the script | `TestNoRosterTextReachesTheScript`, `TestTheScriptIsIdenticalAcrossWeeks`. Both were checked against a deliberately injected `{{.Team}}` and `{{.Total}}` and both failed on it, so neither is vacuous. |
| No margin appears | `TestThePageShowsNoMargin` (pre-existing, unchanged). |
| The leading column is styled like the trailing one | `TestTheTwoColumnsCarryTheSameMarkup` (pre-existing, unchanged). |
| A refresh marks no change | `TestTheScriptImpliesNoWinner` — no leader vocabulary in the script, and no `localStorage`/`sessionStorage`/`document.cookie` for a previous total to survive the reload in. Also verified against the injected mutation above. |

No scenario is left to a test that would pass whatever the script said.

## Section 5 is outstanding (deferred to a live Sunday)

Sections 1-4 and 6 are done; section 5 is not, and was deliberately deferred rather than dropped.
Every task in it is a browser observation against a running server — a five-minute focused wait, a
fifteen-minute hidden tab, a thirty-second round trip, a severed Sleeper path, and a phone
backgrounded for ten minutes. None of them can run in `go test`, and setting them up on a quiet
weekday buys less than doing them on the Sunday the page is actually being read, when the scores
are moving and a stale page is visible as stale.

So what has been verified is the rendered contract: the script is present, its interval is five
minutes, the timer is guarded by `document.visibilityState`, the return path is conditioned on
elapsed time from the client clock, and nothing from the page is interpolated into it. What has
**not** been verified is that a browser does the corresponding things. The gap is real:

- The reload could fire and the page still look stale if a tab is restored from bfcache in a state
  where the script never re-runs. Nothing in the Go tests would notice.
- The elapsed-time comparison is pinned only as text. That the arithmetic is right in a running
  browser is untested.
- 5.5's `502` — the design's self-declared sharpest edge — has never been seen.

Carry all six tasks to the live Sunday and tick them there. They are the only coverage those
scenarios have.

## The backlog judgement (task 6.4)

The "Add the ~5 minute client refresh" line is ticked, with a pointer to the caveat above. The as-of
timestamp and TTL cache lines are left unticked, as the proposal requires.

Which to take next: **the as-of timestamp**, and deferring section 5 is what decides it. 6.4 asked
whether 5.5's result argues for one of them, and 5.5 did not run — so the argument has to come from
what deferring costs rather than from what was observed.

It costs this: the page is now refreshing on a mechanism nobody has watched work, and it looks
exactly the same whether the mechanism works or not. The design already flagged that as a risk. The
timestamp is what converts it from invisible to obvious, and it is what makes the Sunday session
worth anything — walking through 5.1-5.6 against a page with a visible as-of is a matter of reading
it, where against this page it means tailing the server log and inferring. The timestamp is also
the smaller change and the one that makes the other observations legible.

The TTL cache stays second. Its argument is upstream volume, which at three viewers is about one
call a minute at peak, and the visibility guard already stops it scaling with tabs left open. But
that ordering is contingent on staying private: if the page is deployed anywhere public before
either lands, the cache goes first, as the design says. And 5.5 remains worth running when the
Sunday comes — a live look at what the 502 does to a reader mid-game could still reorder these.
