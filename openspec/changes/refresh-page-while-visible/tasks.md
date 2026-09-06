Each behaviour below is one red-green pair. A RED task is not done until the test has been run and
has failed **for the expected behavioural reason** — a wrong value in the rendered body, not a build
error and not a missing symbol. So the first GREEN deliberately ships a *wrong* script rather than
no script, and each later RED then fails on a mismatch against it.

Everything here is rendered-body assertion. The tests use the existing `internal/web` harness —
`writeWeek` into a `t.TempDir()`, a `fakeSource` built with `score.NewWeekStats`, no `httptest`
upstream and never the real `scripts/lineups`. What a browser does with the timer cannot be tested
in `go test`; section 5 is the hand verification that covers it, and it is not optional.

The as-of timestamp and the TTL cache are out of scope, per the proposal. If a task below seems to
want either, it is the wrong task.

## 1. A place to assert on the script

- [x] 1.1 Add a test helper to `internal/web/matchup_test.go` that returns the contents of the
      page's `<script>` element from a rendered body, failing the test if there is not exactly one.
      Every assertion in sections 2-4 goes through it, so no test accidentally matches the string
      it is looking for somewhere else in the document. Confirm it builds; it has no callers yet.

## 2. Refreshing on an interval

- [x] 2.1 RED: test that a rendered page contains exactly one `<script>` element. Run; confirm it
      fails because the page has none.
- [x] 2.2 GREEN: add a script to the end of `matchup.html` that reloads on a `setInterval`, with a
      deliberately wrong one-second interval and no visibility guard. Re-run; confirm pass. The
      wrong value is the point: it is what the next test fails against.
- [x] 2.3 RED: test that the script's interval is five minutes in milliseconds. Run; confirm it
      fails on the one-second value rather than on the script being absent.
- [x] 2.4 GREEN: change the interval to `5 * 60 * 1000`. Re-run; confirm pass. Comment the value
      with what it is for, not with what it says.
- [x] 2.5 RED: test that the script only reloads while the tab is visible — assert it consults
      `document.visibilityState` and compares against `"visible"`. Run; confirm it fails against
      the unguarded reload from 2.2.
- [x] 2.6 GREEN: guard the reload with the visibility check. Re-run; confirm pass. Comment why the
      guard is there — a hidden tab must not turn into upstream Sleeper volume — rather than
      restating the condition.
- [x] 2.7 Read the script back and confirm it does not clear or restart the interval on
      visibility change. The design chose the fire-time guard over start/stop bookkeeping; if the
      implementation drifted into the other shape, bring it back or change the design, not both
      silently.

## 3. Returning to a stale tab

- [x] 3.1 RED: test that the script registers a `visibilitychange` listener. Run; confirm it fails
      against the interval-only script.
- [x] 3.2 GREEN: add the listener, reloading unconditionally whenever the tab becomes visible.
      Re-run; confirm pass. This is wrong on purpose — it refreshes on every tab switch — and 3.3
      is what fails against it.
- [x] 3.3 RED: test that the return path is conditioned on elapsed time and not on the visibility
      change alone — assert the script captures a load time from `Date.now()` and compares an
      elapsed value against the same interval constant used by the timer. Run; confirm it fails
      against the unconditional reload.
- [x] 3.4 GREEN: capture `Date.now()` when the script first runs and reload on return only when at
      least the interval has passed. Re-run; confirm pass. Comment that a brief switch away must
      not cost a fetch.
- [x] 3.5 Confirm by reading that the load time comes from the client clock and not from anything
      the server rendered. The script interpolates nothing; a server-stamped time here would be the
      as-of timestamp arriving through the back door, and it is a separate change.

## 4. The script says nothing about the page

- [x] 4.1 RED: render a week whose two teams and eighteen starters have distinctive names, and test
      that none of those names appears inside the script element. Run; confirm — if it passes
      immediately, keep it as the regression test that pins the no-interpolation rule and say so in
      the commit.
- [x] 4.2 RED: test that two pages rendered from different weeks, different teams, and different
      stat payloads produce a byte-identical script. Run; confirm. This is the strongest form of
      the rule and the one that would catch an interpolation added later.
- [x] 4.3 RED: extend the existing no-winner assertions to the script — test that it contains no
      comparison of the two totals and nothing from a leader/winner vocabulary, matching how
      `TestTheTwoColumnsCarryTheSameMarkup` and `TestThePageShowsNoMargin` already screen the
      markup. Run; confirm.
- [x] 4.4 GREEN: make 4.1-4.3 pass if any of them did not. Re-read the script and confirm it does
      nothing with the previous page's values — no stashed total, no diff, no highlight of what
      moved. A refresh that marks a changed number is the no-winner rule broken by another route.
- [x] 4.5 Confirm `TestRosterTextIsEscaped` still passes and still exercises the HTML text context.
      The script must not have moved any rendered name into a JavaScript context.

## 5. What only a browser can tell us

> **Deferred to a live Sunday, not done.** These six are the only coverage the browser-behaviour
> scenarios have; see `notes.md`. Leave them unticked until they have actually been observed.


Run the server against the real lineup tree for these. Each is a behaviour section 4 cannot reach,
and none is done until it has actually been observed.

- [ ] 5.1 Load a week page, leave the tab focused, and confirm from the server log that it
      re-requests itself about five minutes later. Confirm the scores on screen are from the new
      response.
- [ ] 5.2 Switch to another tab for fifteen minutes. Confirm the server log shows **no** requests
      for the page during that time.
- [ ] 5.3 Return to the tab from 5.2 and confirm it refreshes at once, and that the server log
      shows exactly one request on return — not a burst of the intervals that were skipped.
- [ ] 5.4 Switch away and back within about thirty seconds. Confirm no request is made on return
      and the page does not flash a reload.
- [ ] 5.5 Stop the network path to Sleeper (or point the client at an unreachable host) and let a
      refresh fire. Confirm the reader gets the `502` page in place of their scores, as the design
      predicts. Record what it actually looks like in the change notes — this is the risk the
      design flagged, and one live observation is worth more than the paragraph about it.
- [ ] 5.6 Confirm on a phone browser, which is where the page will actually be read: background the
      browser entirely, return after ten minutes, and confirm the page is current rather than
      showing the pre-background scores.

## 6. Close out

- [x] 6.1 Re-read the script as a whole. It should be a handful of lines; if it is not, say why in
      the change notes or cut it back. Delete any comment that restates the code and keep the two
      carrying rules: why the guard exists, and why a brief switch away does not refresh.
- [x] 6.2 Run `gofmt -l ./...`, `go vet ./...`, and `go test ./...`; confirm clean, and confirm
      `matchup.go`, `cmd/server/main.go`, `scripts/**`, and the roster CSV format are untouched by
      this change.
- [x] 6.3 Walk the `matchup-page` delta spec scenario by scenario and confirm each is covered by
      either a test from sections 2-4 or an observation from section 5, and note which. A scenario
      covered by neither is a gap, not a formality.
- [x] 6.4 Tick the backlog's "Add the ~5 minute client refresh…" line. Leave the as-of timestamp and
      TTL cache lines unticked, and consider whether 5.5's result argues for taking one of them
      next — record that judgement, do not act on it here.
