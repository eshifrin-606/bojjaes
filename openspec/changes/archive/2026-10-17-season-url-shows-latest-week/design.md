## Context

`GET /{season}` answers `302` to `/{season}/{latest}` (`internal/web/season.go`). The week bar builds
its `href`s in the template as `/{{$.Season}}/{{.}}`. The refresh script checks
`location.pathname + "/version"`, so on a page at `/2026` it would ask `/2026/version`, which the mux
currently routes to `GET /{season}/{week}` with week `version`. That answers `400`, and the script
treats it as a failed check and keeps polling, so version-driven reloads would stop without any
visible error.

## Goals / Non-Goals

**Goals:**
- `/{season}` is the current week's address, and an open tab on it moves to a new week by itself.
- Numbered URLs keep working unchanged.

**Non-Goals:**
- A route for `/` (latest season).
- Deciding "current" from the NFL calendar. The newest week directory decides it, as today.

## Decisions

**An open `/{season}` tab follows the latest week.** `/{season}/version` resolves the latest week
on every request. When week 7's directory appears, the answer is week 7's version (empty if it
hasn't been fetched), which differs from the page's, so the page reloads `/2026` and shows week 7.
*Alternative:* render a pinned check URL (`/2026/6/version`) into the page. Rejected: the tab would
stay on week 6 until the 5-minute timer reloaded it anyway, so pinning only delays the switch.

**The season route reuses the week render; it doesn't redirect internally.** Split
`Handler`'s body into a render keyed on `(season, week)`. The week route calls it after
`seasonWeek`, and the season route calls it after `LatestWeek`. Status codes and refusals for the
latest week then match its numbered URL. *Alternative:* rewrite `r.URL` and call the week handler.
Rejected because it depends on how path values get populated, which is a mux detail.

**Links to the latest week point to `/{season}`, and nothing redirects.** The handler computes each
neighbour's `href` and puts it in the view, using `tree.LatestWeek`, which is a directory listing
just like `HasWeek`. The template stops building URLs itself. *Alternative:* `302` from
`/{season}/{latest}` to `/{season}`. Rejected: whether that redirect happens would depend on whether
a higher week exists yet, and every click on a "next" link would cost an extra hop.

**The season form of the version endpoint gets a latest-week resolver, not the tree.**
`VersionHandler` currently takes no lineup tree so it *cannot* read a roster. Give the season form
a one-method interface (`LatestWeek(season) (int, bool)`) that `*lineup.Tree` satisfies. That keeps
the guarantee enforced by the type rather than by convention. In Go 1.22's mux, `GET
/{season}/version` is strictly more specific than `GET /{season}/{week}`, so registering both is
not a conflict, and the literal route wins.

**The refresh script is unchanged.** `location.pathname + "/version"` is right for both URL forms
once the season form exists.

## Risks / Trade-offs

- [A week directory that exists before both rosters are in it] → `/2026` and every tab open on it
  reload into a `500` until the second roster lands. Today only the numbered URL would `500`. To
  avoid that, write both rosters for a new week together. See the open question below.
- [Latest week changes between a check and the reload] → harmless: the reload shows whichever week
  is latest then.
- [Something outside the app relied on the `302`] → none known. The page's own links were always
  numbered.

## Open Questions

- How do new weeks' lineups reach the volume (ADR 0005)? If the two rosters can land separately,
  should the latest week only count once it is a well-formed matchup? That rule would change
  `lineup-source`'s latest-week answer, so it is out of scope here unless it turns out to happen in
  practice.
