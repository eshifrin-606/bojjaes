## Context

`GET /{season}/{week}` is the only route (`cmd/server/mux.go`). The week bar (`add-week-navigation`)
already needs to ask the tree "does this week exist" without resolving it (`Tree.HasWeek`, per
`lineup-source`'s existence requirement) and follows the same "exists, not resolved" rule this
change reuses. `internal/lineup.Tree` locates lineup files against an `fs.FS` rooted at the tree,
with the `<season>/<week>/<team>.csv` layout encoded once, in `weekDir`.

## Goals / Non-Goals

**Goals:**
- `GET /{season}` redirects to the season's highest existing week.
- Resolving that redirect never opens a roster file or calls the stats provider.

**Non-Goals:**
- Changing `GET /{season}/{week}`, the week bar, or any other existing route.
- A true `/` landing page across seasons.

## Decisions

- **New `Tree.LatestWeek(season int) (week int, ok bool)`**, implemented with `fs.ReadDir` on the
  season directory, parsing each directory entry's name as an integer and keeping the maximum.
  Alternative considered: loop `HasWeek(season, w)` over `score.minWeek..maxWeek` (the pattern the
  week bar's `linkable` closure uses) — rejected because it makes `internal/lineup` depend on
  `internal/score`'s bounds for something the tree can already answer by listing its own
  directory, and because it silently caps at `maxWeek` rather than reporting whatever is actually
  there.
- **A separate handler, not a modification to the existing one.** The existing handler resolves a
  matchup and fetches stats; this one only redirects. Keeping them separate keeps the existing
  handler's contract (validate → resolve → fetch → render) unchanged.
- **`http.StatusFound` (302).** The target is a fact about the tree's current contents (this
  season's latest week), not a permanent identity for `/{season}`, so a permanent redirect would be
  wrong once a new week is added.
- **Season-only validation via the existing bounds.** `score.ValidateSeasonWeek(season, 1)` checks
  the season range without adding a new exported season-only validator, since week 1 is always
  in range.
- **Route registration.** `mux.Handle("GET /{season}", ...)` alongside the existing
  `GET /{season}/{week}`; `net/http`'s `ServeMux` (Go 1.22+) prefers the more specific pattern, so
  a request with both segments still reaches the existing handler.

## Risks / Trade-offs

- [A client that caches redirects across a season boundary sees last week's page] → 302 is
  explicitly non-cacheable by default in the absence of caching headers this handler does not set,
  and the risk is no different from the same client caching `/{season}/{week}` itself.

## Open Questions

None.
