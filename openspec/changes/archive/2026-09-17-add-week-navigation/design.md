## Context

`GET /{season}/{week}` is served by `web.Handler(tree, source)`. The handler validates the path,
calls `tree.Matchup` (which reads one week directory), reads both rosters, fetches that week's
stats, and renders `matchup.html`. It is careful about the order: nothing reaches the provider until
the lineup tree has vouched for the request.

`lineup.Tree` wraps an `fs.FS` rooted at the tree. In production that is the embedded
`lineup.Embedded`; in tests it is an `fstest.MapFS`. The layout `<season>/<week>/<team>.csv` is
stated in one place: `weekDir` builds the directory and `Path` builds on it. Nothing today lists or
enumerates weeks.

The tree today: 2025 weeks 14–16 and 2026 weeks 1–2. The seasons are contiguous, but only by
accident: the Bojjaes could miss a playoff week and leave a gap.

Constraints still in force:

- *The page implies no winner*: nothing on the bar can depend on the points.
- *Nothing sits above the heading* is about elements inside a column. The bar sits outside both.
- The 33rem media query and the 12.5rem container threshold size the cards. The bar must not add
  width to either card.
- The layout tests run in headless Chrome when it is available and skip otherwise.

## Goals / Non-Goals

**Goals:**

- A label naming the season and week, and previous/next links to week ± 1 in the same season,
  each shown only if that week's directory exists.
- Existence is decided by the lineup package, which is the only code that knows the layout.
- Deciding which links to show adds no provider fetches.

**Non-Goals:**

- Crossing seasons, or skipping over gaps to the nearest existing week.
- Hiding links to weeks that exist but would fail as a matchup.
- A landing page at `/`, or HTML error pages.

## Decisions

### 1. `Tree.HasWeek(season, week) bool`, checked per request with `fs.Stat`

```go
func (t *Tree) HasWeek(season, week int) bool {
	info, err := fs.Stat(t.fsys, t.weekDir(season, week))
	return err == nil && info.IsDir()
}
```

It reuses `weekDir`, so `HasWeek` and `Matchup` cannot disagree about where a week lives.

**Alternative considered: an index of all weeks built once when the `Tree` is constructed.** We
leaned toward this in exploration because "next existing week" needed a sorted list across seasons.
Restricting navigation to week ± 1 in the same season removes that need: each link is one yes/no
question about one directory. An index would bring:

- an error path in `New`, which cannot fail today and is called in `main` and many tests,
- a copy of the tree's contents that could disagree with the filesystem it was built from,
- directory-name parsing (skipping non-numeric names, sorting numerically) that nothing else needs.

Two `Stat` calls on an in-memory embedded filesystem cost nothing next to the roster reads the
handler already does. If the directory rule ever has to consider contents, it changes in one method.

### 2. Existence means "the directory is there", not "the week resolves"

`HasWeek` does not call `Matchup`. A link to a broken week leads to the 500 that exposes the broken
week. Hiding the link would hide the mistake, and would make whether a link appears depend on
roster contents in ways that are hard to predict. It also keeps a link's meaning simple: it
appears exactly when following it would not be a 404.

A regular file named like a week is not a week. `fs.Stat` succeeds on it, so `IsDir` has to be
checked. `Matchup` already treats such a path as `ErrNoWeek` (its `ReadDir` fails), so the two
agree.

### 3. Adjacent weeks are computed in `web`, bounded by `score.ValidateSeasonWeek`

"Week − 1 / week + 1, same season" is a page rule, so it lives in the handler. Each candidate is
passed to `score.ValidateSeasonWeek` before `HasWeek`, so week 0 and week 19 are never asked about
and never linked. Checking the page's own URL bounds means a link is never generated that the
page itself would answer with a 400.

Only the week + 1 side of the check is load-bearing. Week − 1 of week 1 is 0, which decision 4
already renders as no link, so the week − 1 side is kept for symmetry and no test can catch its
removal. The tree could not hold a week 0 anyway.

The handler computes the neighbours **after** `Matchup` succeeds and **before** the stats fetch. A
404 or 500 week renders no page, so it needs no links. Doing it before the fetch keeps all tree work
ahead of the provider, as the handler already does.

### 4. View model: `PrevWeek, NextWeek int`, zero meaning absent

```go
type matchup struct {
	Season, Week       int
	PrevWeek, NextWeek int // 0 when that week does not exist
	...
}
```

Weeks start at 1, so 0 is never a real week, and `{{with .PrevWeek}}` in the template treats it as
absent without a separate flag. The href is built in the template from `.Season` and the week, so
the season cannot differ between the label and the links.

Alternative: `*int` or a `weekLink{Href, Label}` struct. These are more explicit but add pointer or
type plumbing for a value the template already handles with `with`.

### 5. Markup: a `<nav>` above `<main>`, three fixed slots

```html
<nav class="weeks">
  <span class="prev">{{with .PrevWeek}}<a href="/{{$.Season}}/{{.}}" rel="prev">‹ Week {{.}}</a>{{end}}</span>
  <h1>{{.Season}} · Week {{.Week}}</h1>
  <span class="next">{{with .NextWeek}}<a href="/{{$.Season}}/{{.}}" rel="next">Week {{.}} ›</a>{{end}}</span>
</nav>
```

```
┌──────────────────────────────────────────────┐
│ ‹ Week 1        2026 · Week 2                │  ← .next slot is empty but still
├──────────────────────┬───────────────────────┤    takes its 1fr
│ BOJJAES        56    │ RENEGADES       48    │
```

- `.weeks` is `display: grid; grid-template-columns: 1fr auto 1fr`. The side slots always render,
  empty or not, and they share the remaining width equally, so the label stays centred whichever
  links exist. `.next` is `text-align: right`.
- The label is the page's `h1`. The page has no `h1` today, and the team names are `h2`s under it.
- The links are plain anchors. The refresh script reloads the current URL, so it behaves the same
  on any week the reader navigates to.
- Link text omits the season: links never cross seasons, so the season in the label applies to the
  links too.
- At 390px the bar needs about `‹ Week 1` + `2026 · Week 2` + `Week 3 ›`, well under the width, so
  no narrow-viewport variant is expected. The rendered-layout test confirms it.

`html/template` escapes `.Season` and the week inside `href` as URL content. Both are `int`s, so
there is nothing to escape, but it costs nothing and stays safe if the types ever change.

### 6. Tests

- `internal/lineup`: `HasWeek` table over `fstest.MapFS`, with a present directory, a missing
  week, a broken week (three rosters), a week holding an unparseable roster, and a file where the
  directory should be.
- `internal/web`: a multi-week fixture (merge several `weekFS` values into one `MapFS`) for markup
  tests covering both links, first week, latest week, the season boundary, a gap, and a broken
  neighbour. A fake source that records the season and week it was asked for, to pin that only the
  shown week is fetched.
- `layout_test.go`: the label's left edge is the same with and without a previous link, and at
  390px the three items share a line, stay inside the bar, and neither link wraps.

## Risks / Trade-offs

- **[A lineup committed ahead of kickoff gets a next link to a page of `--`.]** → This is the
  intended reading: the week exists and has no stats yet. The page already renders it honestly.
- **[A stray empty directory makes a link to a 500.]** → Intended by decision 2. The 500 is the
  signal to fix the tree.
- **[No link from 2026 week 1 back to 2025.]** → Chosen by the user. The URL still works directly.
  Crossing seasons can be added later, since it only changes how the neighbours are computed.
- **[The `h1` shifts the vertical position of the rendered-layout fixtures' cards.]** → The
  existing layout tests compare elements with each other, not with absolute page coordinates. Any
  test that turns out to depend on absolute positions is fixed as it fails.
