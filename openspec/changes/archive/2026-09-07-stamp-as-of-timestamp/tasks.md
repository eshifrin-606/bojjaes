Each behaviour below is one red-green pair. A RED task is not done until the test has been run and
has failed **for the expected behavioural reason** — a wrong string, a wrong instant, a missing
element — not a build error and not a missing symbol. Section 1 exists to get the two packages
compiling against the new interface shape with a deliberately wrong timestamp, so every RED after it
fails on a mismatch.

The wrong implementation is the point: first `WeekStatsAsOf` returns `time.Time{}` (the zero
instant), then it returns `c.now()` at read time, then it returns the entry's stored `fetchedAt`.
Each is what the next test fails against.

Tests use the existing fakes and injected clocks. No `httptest` reaching a network, no real Sleeper.

## 1. The interface shape, compiling and wrong

- [x] 1.1 In `internal/statscache`, add `WeekStatsAsOf(ctx, season, week) (score.WeekStats,
      time.Time, error)` that calls the wrapped source and returns `(stats, time.Time{}, err)` —
      the zero instant, deliberately. Change `WeekStats` to delegate: `stats, _, err :=
      c.WeekStatsAsOf(ctx, season, week)`. Confirm the package builds and the existing
      `weekly-stats-cache` tests still pass unchanged (they only call `WeekStats`).
- [x] 1.2 Update the compile-time interface assertions in the `statscache` test file to also assert
      `*Cache` satisfies a local copy of `web`'s new `WeekStatsAsOf` interface shape, alongside the
      existing `api`/`web` `WeekStats` one.
- [x] 1.3 In `internal/web`, change the local `StatsSource` interface to the single method
      `WeekStatsAsOf(ctx, season, week) (score.WeekStats, time.Time, error)`. Update the handler to
      call it, take the `time.Time`, and drop it on the floor for now. Update `unusedSource` and
      `fakeSource` in the test file to implement `WeekStatsAsOf` (fake returns a settable
      `fetchedAt` field, zero by default). Confirm `internal/web` builds and every existing web
      test passes with no assertion changes — the page does not render the timestamp yet.
- [x] 1.4 Confirm `go build ./...` and `go vet ./...` are clean and `cmd/server/main.go` is
      unchanged — the one wrapped value still satisfies both `api.StatsSource` and the new
      `web.StatsSource`.

## 2. The cache returns the fetch instant, not the read instant

- [x] 2.1 RED: in `internal/statscache`, test that `WeekStatsAsOf` on a fresh miss returns a
      non-zero instant equal to the clock at fetch completion. Run; confirm it fails reporting the
      zero instant, against 1.1.
- [x] 2.2 GREEN: return `entry.fetchedAt` from the miss path. Re-run; confirm pass and confirm the
      `weekly-stats-cache` suite still passes.
- [x] 2.3 RED: test that a cache hit returns the **original** fetch instant — fetch, advance the
      injected clock three minutes, read again through `WeekStatsAsOf`, assert the returned instant
      did not move. Run; confirm it fails only if the hit path recomputes `now()`; if it passes
      immediately because 2.2 already reads the stored field, keep it as the regression test that
      pins hit-path attribution and say so, then break the hit path to `c.now()`, watch it fail,
      and restore.
- [x] 2.4 RED: test that every waiter on one in-flight fetch receives the same completion instant —
      block the fetch, launch N callers on `WeekStatsAsOf`, release, assert all N got identical
      non-zero instants and identical stats. Run; confirm it fails if the waiter path returns
      `time.Time{}` or a per-waiter `now()`.
- [x] 2.5 GREEN: return `cached.fetchedAt` from the waiter path (it is set under `c.mu` before
      `close(flight.done)`, so a released waiter reads a stable value). Re-run 2.1–2.4; confirm all
      pass.
- [x] 2.6 RED: regression — test that `WeekStats` (no `AsOf`) still returns exactly `(stats, err)`
      and makes the same one upstream call for two reads within the TTL, i.e. the delegation did
      not change caching behaviour. Break the delegation (e.g. have `WeekStats` fetch directly),
      watch it fail, restore.

## 3. The page renders the instant, machine-readable

- [x] 3.1 Add `America/Chicago` loading to `internal/web`: `time.LoadLocation` at package init
      next to the template parse, stored in a package var, with `import _ "time/tzdata"` so the
      binary carries its own zone database. Add a test that the location loaded (non-nil, name
      `America/Chicago`). Run; confirm pass.
- [x] 3.2 RED: test that a served page contains exactly one `<time>` element whose `datetime`
      attribute is the fake's `fetchedAt` formatted as `time.RFC3339`. Use a fixed non-zero
      `fetchedAt` on the fake. Run; confirm it fails — no `<time>` element in the current template.
- [x] 3.3 GREEN: add `FetchedAtRFC3339 string` to the view model, set it from the handler's
      `time.Time` via `Format(time.RFC3339)`, render it as `<time datetime="…">` in the template
      outside `<main class="matchup">`. Re-run; confirm pass.
- [x] 3.4 RED: test that the RFC 3339 attribute denotes the same instant regardless of the fetch's
      wall-clock zone — set the fake's `fetchedAt` to a known UTC instant, assert the attribute
      parses back (`time.Parse(time.RFC3339, …)`) to an equal instant. Run; confirm it fails if the
      handler formatted in Chicago local time and dropped the offset. Settle on formatting the
      original instant, not `.In(loc)`, for the attribute.

## 4. The page renders the instant, human-readable and labelled

- [x] 4.1 RED: test that the page's visible text (tags stripped) contains the `fetchedAt` instant
      rendered as an `America/Chicago` wall-clock time including the zone abbreviation — pick a
      `fetchedAt` that is a known Chicago local time (e.g. a summer instant → `CDT`) and assert the
      formatted substring appears. Run; confirm it fails — only the RFC 3339 string renders so far.
- [x] 4.2 GREEN: add `FetchedAtText string` to the view model, formatted from
      `fetchedAt.In(chicagoLoc)` with a layout that shows date, time, and zone (settle the exact
      layout here against the assertion). Render it as the visible content of the `<time>` element.
      Re-run; confirm pass.
- [x] 4.3 RED: test the label — the visible text around the `<time>` element identifies the instant
      as when the data was **fetched** (assert the word, e.g. "fetched", is adjacent), and the page
      nowhere contains "live" or "current" (case-insensitive). Run; confirm it fails against the
      bare timestamp from 4.2.
- [x] 4.4 GREEN: add the label text in the template (e.g. `Sleeper stats fetched <time …>…</time>`).
      Re-run; confirm pass.
- [x] 4.5 RED: test the no-winner constraint for the new element — render the spec's 56 vs 42
      example, assert the as-of line's text contains neither total, no "-" between numbers, and
      that the `<time>` element is not inside any `class="column"` section. Run; confirm pass (or
      fix placement until it does).

## 5. Cache-hit attribution end to end

- [x] 5.1 RED: a `internal/web` handler test wired to a real `statscache.Cache` (injected clock)
      over a counting fake: request the page, advance the clock three minutes, request again,
      assert the second response's `<time datetime>` equals the first's — the served instant is the
      fetch, not either request. Run; confirm it passes on the machinery from sections 2–4, or
      shows exactly which path recomputed the time.
- [x] 5.2 RED: test that the second request in 5.1 made **no** second upstream call (fake's count
      is 1) — the as-of path did not bypass the cache. Run; confirm.

## 6. Docs and close-out

- [x] 6.1 Update `docs/package-dependencies.md` if it states the `web`→`statscache` interface
      shape or method name; the dependency direction and arrows are unchanged.
- [x] 6.2 Tick the backlog line "Stamp the as-of from our own Sleeper fetch time for now, labelled
      as such…" and its sibling "Keep the page honest: as-of timestamp visible…" if this satisfies
      it. Leave the "watch on a live Sunday how far it drifts" half unticked with a pointer to
      `notes.md`.
- [x] 6.3 Write `notes.md`: the deferred live-Sunday drift observation (procedure, what it feeds),
      why the stamp is the fetch time and not the stat-movement time, and any layout decision made
      in 4.2 that the spec left open.
- [x] 6.4 `go test ./...` clean; `openspec validate stamp-as-of-timestamp --strict` clean.
