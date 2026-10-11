Each task is one red-green behavior unless marked otherwise. When a test needs a new symbol, first
add it as a stub that compiles and returns a wrong value. That way the red shows a value mismatch,
not a build error. "Confirm green" tasks pin behavior the previous step already gives; break the code
briefly to see each one fail.

## 1. `statscache`: read a week's fetch instant (D5, D10)

- [x] 1.1 Stub `func (c *Cache) FetchedAt(season, week int) (time.Time, bool)` returning the zero
      time and `false`.
- [x] 1.2 Red-green: fetch week 4 under the fake clock, then ask. Expect `true` and the same instant
      `WeekStatsAsOf` returned.
- [x] 1.3 Red-green: block the fake source, start a week 4 fetch, ask while it is in flight. Expect
      `false`, returned without waiting.
- [x] 1.4 Confirm green: a never-fetched week reports `false`. An invalidated week reports `false`.
      An entry six minutes old with no later request still reports its instant.
- [x] 1.5 Confirm green: asking many times makes no upstream call and writes no log line.
- [x] 1.6 Run the whole `statscache` suite with `-race`.

## 2. `web`: the page carries its stats version and first delay (D2, D7)

- [x] 2.1 Red-green: a page served from a known fetch instant has `data-version` on the as-of
      paragraph equal to that instant as Unix nanoseconds in decimal. Exactly one `data-version` on
      the page.
- [x] 2.2 Red-green: two instants within the same second give different `data-version` values.
- [x] 2.3 Red-green: the as-of paragraph has `data-next-check` equal to the default delay in seconds,
      taken from one exported or package-level constant.
- [x] 2.4 Confirm green: over a real `statscache.Cache`, two pages served from one cached fetch carry
      the same version.

## 3. `web`: the version endpoint (D3, D4, D8)

- [x] 3.1 Declare the interface the handler consumes (one method, `FetchedAt`) and stub
      `VersionHandler` to answer 200 with `{}`. Add a fake for the tests.
- [x] 3.2 Red-green: with a fetch instant held, the JSON body's `version` is that instant's token,
      produced by the same function the page uses.
- [x] 3.3 Red-green: with nothing held, `version` is the empty string.
- [x] 3.4 Red-green: `nextCheckSeconds` is the default delay.
- [x] 3.5 Red-green: `Content-Type` is `application/json` and `Cache-Control` is `no-store`.
- [x] 3.6 Red-green: a non-numeric season or week is refused with 400.
- [x] 3.7 Red-green: an out-of-range season or week is refused with 400, using
      `score.ValidateSeasonWeek`.
- [x] 3.8 Confirm green, over a real `statscache.Cache` and a counting upstream: the endpoint's
      version equals a freshly served page's; after `Invalidate` it is empty; after a refetch it is
      the new page's and differs from the first; the endpoint alone never calls upstream.

## 4. `cmd/server`: route and wiring (D10)

- [x] 4.1 Give `stubStats` in `mux_test.go` a `FetchedAt` method and widen what `newMux` accepts, with
      no new route yet. The suite stays green.
- [x] 4.2 Red-green: `GET /2025/15/version` is answered by the version handler, not a 404.
- [x] 4.3 Red-green: `POST /2025/15/version` is 405.
- [x] 4.4 Confirm green: `GET /2025/15` and `GET /2025` still reach their handlers; `main.go` compiles
      passing the one `stats` cache for both roles.

## 5. `web`: the check script (D6, D9)

Script tests read the script's text through `renderedScript`, as the existing refresh tests do.

- [x] 5.1 Red-green: the script reads the version and the first delay from `dataset` and builds the
      check URL from `location.pathname` plus `/version`.
- [x] 5.2 Red-green: the check is guarded by `document.visibilityState === "visible"`, separately from
      the five-minute timer's guard.
- [x] 5.3 Red-green: the script reloads with `location.reload()` when the reported version is not
      equal to its own, and compares by string equality only.
- [x] 5.4 Red-green: the next check is scheduled with `setTimeout` from the response's
      `nextCheckSeconds`, and a value of zero schedules nothing.
- [x] 5.5 Red-green: a failed or unreadable response does not reload and reschedules at the previous
      delay.
- [x] 5.6 Confirm green: the existing refresh tests still pass unchanged, including the five-minute
      interval, the stale-tab return, `TestTheScriptImpliesNoWinner`, and the test that no roster
      name reaches the script. Add the version value to that last test's forbidden list.

## 6. Run the script for real

- [x] 6.1 Headless-Chrome test, skipped under `-short` or without Chrome like the layout tests: an
      `httptest` server serves a page with a one-second first delay whose version endpoint reports a
      different version, and serves a marked second render on reload. Run Chrome with `--dump-dom`
      and a virtual time budget; expect the marked render. If virtual time cannot drive the fetch,
      record that here and rely on 6.2.
      Result: virtual time does drive the fetch. `TestTheCheckScriptInChrome` runs nine cases
      (reload on a later or empty version, keep checking on a match, stop on zero, follow a longer
      delay, no reload on an error status, a missing route, or an unknown shape).
- [x] 6.2 Manual check against a local server with a cold week: the page reloads on its own after the
      whole-week fetch lands; the server log shows version requests stop while the tab is hidden and
      resume when it is shown.
      Server half checked 2026-10-11 with curl against a local server and a cold 2026 week 5: the
      endpoint went empty when the whole-week fetch landed, and matched the page again after a
      reload. Still to do by hand in a real browser: watch the reload happen, and hide and show
      the tab.

## 7. Wrap up

- [x] 7.1 Run `go vet ./...` and `go test -race ./...`.
- [x] 7.2 Update `docs/architecture.md` where it describes the page's refresh and the server's
      routes.
- [x] 7.3 Run `openspec validate reload-matchup-page-on-server-change`.
