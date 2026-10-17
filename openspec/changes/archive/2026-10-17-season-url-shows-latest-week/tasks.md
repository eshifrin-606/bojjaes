Each task is one red-green behavior unless marked otherwise. When a test needs a new symbol, first
add it as a stub that compiles and returns a wrong value. That way the red shows a value mismatch,
not a build error.

## 1. `web`: the season route renders the latest week

- [x] 1.1 Refactor (tests stay green): split `Handler`'s body into a render keyed on
      `(season, week)` that the week route calls after `seasonWeek`.
- [x] 1.2 Red-green: with 2026 weeks 1 and 2, `GET /2026` answers `200`, has no `Location` header,
      and is labelled `2026 · Week 2`. `SeasonHandler` takes the stats and games sources and calls
      the shared render.
- [x] 1.3 Red-green: the stats provider is asked for 2026 week 2 exactly once.
- [x] 1.4 Confirm green: a gap (weeks 1 and 3) shows week 3. A three-roster latest week answers
      `500`. The existing `400`/`404` season tests still pass, with redirect assertions replaced.

## 2. `web`: neighbour links to the latest week use the season URL

- [x] 2.1 Red-green: with weeks 1–3, `GET /2026/2` has `rel="next"` pointing to `/2026`, with text
      `Week 3 ›`. Move `href` construction from the template into the view.
- [x] 2.2 Confirm green: with weeks 1–4, `/2026/2` links to `/2026/1` and `/2026/3`. `GET /2026`
      shows the same bar as `/2026/{latest}`. Update existing week-bar tests whose next week is the
      latest.

## 3. `web`: the season form of the version endpoint

- [x] 3.1 Red-green: register `GET /{season}/version` in `newMux`. `GET /2026/version` answers the
      version a fresh `GET /2026` page carries (not `400`). Its handler takes a one-method
      `LatestWeek` interface, not `*lineup.Tree`.
- [x] 3.2 Red-green: after a week 3 directory appears, `GET /2026/version` reports week 3's held
      version (empty when unfetched).
- [x] 3.3 Red-green: a season with no weeks answers `404`. `GET /twentytwentysix/version` answers
      `400` without reaching the resolver.
- [x] 3.4 Confirm green: `Cache-Control: no-store`, no stats-provider call, and
      `GET /2026/5/version` still routes to the week form.

## 4. Page script on `/{season}`

- [x] 4.1 Confirm green in the Chrome script test: a page loaded from `/2026` sends its checks to
      `/2026/version`, and reloads `/2026` when the reported version differs.

## 5. Wrap-up

- [x] 5.1 Run `go test ./...` with `-race`. Check `/2026` and `/2026/5` by hand in the running app.
- [x] 5.2 Check whether `slow-stats-polling-when-idle` needs to mention `/{season}/version`.
