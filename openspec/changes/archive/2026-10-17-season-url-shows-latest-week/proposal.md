## Why

The page people bookmark and share is the current week, but its URL names a week number, so a
bookmark of `/2026/6` stops being "this week" as soon as week 7's lineups land. `/2026` already
resolves the current week; it should be the current week's address, not a hop to another one.

## What Changes

- `GET /{season}` renders the season's latest week directly instead of redirecting to
  `/{season}/{week}`. **BREAKING** for anything relying on the `302`.
- `GET /{season}/{week}` is unchanged: every week, including the latest, still renders at its
  numbered URL. Nothing redirects to `/{season}`.
- The week bar's link to a neighbouring week that is the season's latest points to `/{season}`
  rather than `/{season}/{week}`.
- New `GET /{season}/version` reports the stats version of the season's latest week, resolved on
  each request. A tab left open on `/2026` therefore reloads into week 7 once week 7 exists.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `matchup-page`: the bare-season route renders rather than redirects; neighbour links to the
  latest week use the season URL; the version endpoint gains a season-only form that follows the
  latest week.

## Impact

- `internal/web/season.go` — `SeasonHandler` renders via the matchup handler's path instead of
  `http.Redirect`.
- `internal/web/matchup.go` — neighbour link targets; the week-keyed render shared with the season
  route.
- `internal/web/version.go`, `cmd/server/mux.go` — the `/{season}/version` route.
- `internal/web/matchup.html` — link `href`s come from the view rather than being built in the
  template. The refresh script's `location.pathname + "/version"` is unchanged and becomes
  correct on `/{season}`.
- The `slow-stats-polling-when-idle` change, which also touches the version endpoint, should cover
  both route forms.
