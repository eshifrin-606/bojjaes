## Why

The page never says which week it shows. The season and week appear only in the URL and the tab
title, so on a phone the only indication is the address bar. To reach another week, the reader has
to edit the URL. With a second week in the 2026 season (commit 2471d43), moving between weeks is
now something readers need to do.

## What Changes

- A **week bar** above the two cards: a centred label naming the season and week (`2026 · Week 2`),
  a **previous** link at the left, and a **next** link at the right.
- The previous link points to week − 1 and the next link to week + 1, **in the same season only**.
  Week 1 has no previous link. The last week of a season has no next link, even if the following
  season has lineups.
- A link is shown **only if its week exists**, meaning the lineup tree has a directory for it.
  Whether that week is a well-formed matchup is not checked: a week that would be a 500 still gets a
  link, so a mistake in the lineup tree stays visible rather than hidden.
- When a link is absent, its slot stays empty. The label does not move as the reader clicks through
  the weeks.
- The links are plain `<a href>` with `rel="prev"` / `rel="next"`. They need no script and leave the
  refresh script as it is.
- Deciding which links to show reads only the lineup tree. It never calls the stats provider, and
  the page still fetches only the week it shows.

Out of scope:

- Navigation across seasons.
- A `/` landing page, or friendlier 404/500 pages (the backlog item "Decide what `/` … do").
- Links on error responses: they stay plain `http.Error` text.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `lineup-source`: adds *A week's existence can be asked without resolving it*. The tree answers
  whether `<season>/<week>` exists, without reading its rosters and without deciding whether it is
  a matchup.
- `matchup-page`: adds *The page names its week and links to the adjacent weeks of its season*.
  *Nothing sits above the heading* is not modified: it concerns elements inside a column, and the
  week bar sits outside both columns.

## Impact

- `internal/lineup`: a new `Tree` method that reports whether a week exists, built on the existing
  `weekDir`, so the tree's layout is still stated in one place. It has tests over `fstest.MapFS`.
- `internal/web/matchup.go`: the view model gains the adjacent weeks, which the handler fills in
  after resolving the matchup and before the stats fetch.
- `internal/web/matchup.html`: a `<nav>` above `<main>`, and its CSS.
- `internal/web/matchup_test.go` and `layout_test.go`: markup tests for the links, and a
  rendered-layout test that checks the bar fits on one line at 390px and the label does not move.
- `cmd/server` is unchanged: `newMux` already passes the tree to the handler.
- No change to scoring, the stats cache, the provider, or deploys. The in-flight
  `refresh-page-while-visible` change modifies only *The page implies no winner*, which this change
  does not touch.
