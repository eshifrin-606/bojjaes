## Context

Everything the page needs already exists, in pieces: `internal/roster` resolves a week directory to
two teams and parses a roster into starters and bench; `internal/score` fetches Sleeper's weekly
aggregate once and scores any player out of it. What is missing is the handler that joins them and
the template that renders the result.

Constraints that shape the design, all from
[ADR 0004](../../../docs/adr/0004-web-frontend-stack.md):

- `html/template`, no JS framework, no build step, one binary, one origin.
- The page must imply no winner. This is a design constraint on the markup and CSS, not a
  preference.
- The neighbouring backlog items — as-of timestamp, client refresh, TTL cache, `//go:embed` of the
  lineup tree, the unguessable path prefix, CSV display fields — are deliberately separate. This
  design should leave each of them an additive step rather than a rework.

The scoring seam this page needs was built by `split-score-packages`, which lands first:
`internal/score` is the vendor-neutral domain (`StatLine`, `Points`, `Week`), `internal/sleeper` is
the adapter, `internal/api` is the JSON layer, and `cmd/server` is the only package that names a
provider.

## Goals / Non-Goals

**Goals:**

- `GET /{season}/{week}` renders the matchup for that week: two equal columns of scored starters
  and their two totals.
- One upstream fetch per request, scoring both columns.
- Errors are distinguishable: a bad URL, a missing week, a malformed lineup tree, and a failed
  upstream fetch produce four different responses.
- Score a set of players from one fetch without going through HTTP, using the seam
  `split-score-packages` provides.

**Non-Goals:**

- Caching, single-flight, or any change to upstream request volume. One request, one fetch.
- The as-of timestamp and the client refresh. Both land on this page next; neither is here.
- Embedding the lineup tree or a CSS file. The template is embedded because it lives inside the new
  package; the lineup tree still comes from a path on disk.
- Bench rendering, season-long views, lineup submission.

## Decisions

**A new `internal/web` package owns the page.** It holds the handler, the view model it builds, and
the template. `internal/score` stays about scoring and `internal/roster` about roster files;
neither learns about HTML. The alternative — putting the handler in `internal/score` next to the
JSON handlers — would make the scoring package depend on the roster package and on a template, for
one page.

**`internal/web` declares the stats source it needs; `main` supplies it.** `score.WeekStats` and
`sleeper.FetchWeekStats(ctx, baseURL, season, week) (score.WeekStats, error)` already exist. This
package does not import `internal/sleeper`; like `internal/api` it declares the one method it uses —

```go
type StatsSource interface {
    WeekStats(ctx context.Context, season, week int) (score.WeekStats, error)
}
```

— and `main` hands it a `sleeper.Client`. That keeps the provider out of the page's tests, which
build a `score.WeekStats` with `score.NewWeekStats` rather than standing up an `httptest` server,
and it means the coming TTL cache is a wrapping `StatsSource` wired in `main` with nothing here
edited.

**The view model carries strings, not floats, for points.** The template renders what the handler
decided: a scored starter carries its formatted points, an absent one carries the literal
`no stats`. Deciding "is this a zero or an absence" in the handler rather than in the template
keeps the distinction in Go, where it is testable, and keeps the template from needing conditionals
that could be got subtly wrong. `no stats` is the same wording `scripts/scores.sh` already prints,
so the two UIs read alike while both exist.

The total is a `float64` formatted once, and it sums only the starters that had stats — an absent
starter is skipped, not added as zero, which is the same arithmetic by a different name but the
right description of it.

**Route pattern `GET /{season}/{week}` on the existing `http.ServeMux`.** Go's method- and
wildcard-aware patterns already do the parsing split: `/score` is one segment and cannot collide
with a two-segment pattern, and anything but GET on the page path gets a 405 from the mux. The
handler converts the two segments with `strconv.Atoi` and range-checks them against the same
season and week bounds the batch endpoint uses, so a typo cannot reach the lineup tree or Sleeper.
The bounds constants move to a shared place in `score` rather than being duplicated as new
literals.

**Errors map by cause, using the sentinel errors `roster` already exports:**

| Cause | Status |
| --- | --- |
| Non-integer or out-of-range segment | 400 |
| `roster.ErrNoWeek` | 404 |
| `ErrTooFewRosters`, `ErrTooManyRosters`, `ErrNotOurMatchup`, unparseable roster | 500 |
| Fetch or upstream status failure | 502 |

The 404/500 line is drawn at "did the reader ask for something that exists": a week we never played
is a 404, while a week directory that exists and is malformed is our lineup tree being wrong about a
well-formed request. Every one of these logs server-side and returns a plain-text error body; there
is no styled error page in this change.

**Render into a buffer, then write.** `template.Execute` writing straight to the `ResponseWriter`
commits a 200 and half a page before it can fail. The handler executes into a `bytes.Buffer`, and
only a successful execution is written out; a failure is a 500. The page is a few kilobytes, so
buffering costs nothing.

**The template is embedded and parsed once, at construction.** `//go:embed matchup.html` plus
`template.Must(template.ParseFS(...))` at package level means a broken template fails the process
at startup rather than the first request. The file lives in `internal/web/`, so this does not touch
the lineup-tree embed problem — that tree sits outside any package and is the separate backlog
item's job.

**CSS is a `<style>` block in the template.** Two equal columns are a two-column grid and a handful
of rules; a separate embedded stylesheet is the other backlog item, and splitting one screen of CSS
into its own file now buys nothing. The rules that matter are the ones that are absent: no
per-column colour, no `.leading`/`.winner` class, nothing keyed on which total is larger. Both
columns share one class and one width.

**The lineup tree root is a constant in `main.go`** (`scripts/lineups`, relative to the working
directory), passed to `roster.New` and handed to the handler. This is exactly today's arrangement
for the scripts and is knowingly interim: it becomes an embedded FS when the tree moves.

**Testing follows the existing handler tests:** an `httptest.Server` standing in for Sleeper, a
`t.TempDir()` lineup tree written per test, and assertions on status codes and on substrings of the
rendered HTML. The no-winner requirement is tested as an absence — the body must not contain the
difference of the two totals — which is a weak test on its own and is backed by the CSS having no
leader class to assert against.

## Risks / Trade-offs

- **Asserting on rendered HTML substrings is brittle to markup changes** → Assert on the values that
  carry meaning (a player's name, a total, `no stats`, the absence of a margin) rather than on tags
  or structure, so a restyle does not fail the suite.

- **The no-margin rule is enforced by review, not by the type system** → The spec states it as a
  requirement, the tests check for the obvious violation, and the CSS deliberately contains no
  class that could express a leader. A future contributor adding one has to add it from scratch.

- **Every page load is a full Sleeper fetch until the cache lands** → Accepted for this change and
  called out in the proposal: three viewers refreshing by hand is within what one laptop already
  does, and the TTL cache is the next item in the backlog rather than a distant one.

- **The page has no as-of timestamp yet, so a stale tab looks live** → Real, and the reason the
  timestamp item sits immediately after this one. Nothing here makes it harder to add: it is one
  more field on the view model.

- **A two-segment route is greedy** — any future `/{a}/{b}` path has to be registered as a more
  specific pattern → Go's mux prefers the more specific pattern, so `GET /api/health` would still
  win over `/{season}/{week}`. Worth remembering when the path prefix item lands, since it changes
  this pattern's shape anyway.
