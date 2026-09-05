## Context

Roster knowledge lives in `scripts/scores.sh`: a `while IFS=, read` loop, a `lineup_size=9`
variable, and a `case` that builds `lineups/<season>/<week>/<team>.csv`. The Go side of the
repo — `internal/score`, `cmd/server` — knows only about player IDs arriving in a JSON body.
[ADR 0004](../../../docs/adr/0004-web-frontend-stack.md) commits to serving `/{season}/{week}` as
server-rendered HTML from the same binary, which means the page will need exactly the three facts
the bash loop currently holds.

Constraints that shape the design:

- **Scripts are an interim UI, and they stay as they are this change.** `scores.sh` and
  `fantasycast.sh` keep their bash parsing. The package is the Go home for roster knowledge; it is
  not yet the only home.
- **The roster file is a hand-edited lineup card.** File order is meaningful, the file carries no
  positions, and nobody validates it against a real lineup. Anything the package does silently to
  a record changes who starts.
- **`//go:embed` cannot cross `..`.** The lineup tree at `scripts/lineups/**` will eventually have
  to move under a package. Nothing in this change should make that move expensive.
- **The CSV format is going to grow.** The backlog carries a position and team column, and an
  undecided question about whether position means lineup slot or listed position. This change must
  not settle that question by accident.

## Goals / Non-Goals

**Goals:**

- One Go package that reads a roster file, resolves where roster files live, and splits a roster
  into starters and bench.
- Behaviour driven out test-first, one behaviour at a time, per the repo's TDD convention.
- A shape the served page and the matchup resolver can both build on without reinterpreting the
  file format.

**Non-Goals:**

- Changing `scripts/scores.sh` or `scripts/fantasycast.sh`. Their bash parsing stays; the
  duplication ends when the page replaces them.
- Scoring, HTTP handlers, templates, or anything touching `internal/score`.
- Matchup resolution from a week directory ("exactly two rosters, opponent is the file that isn't
  `bojjaes.csv`"). That is the next backlog item and belongs on top of this package.
- Position and team columns, and the lineup-slot-versus-listed-position question.
- Moving `scripts/lineups/**` for `//go:embed`, or reading rosters from anything but the filesystem.

## Decisions

### A new `internal/roster` package, not an addition to `internal/score`

`internal/score` is about turning provider stats into points; it has no notion of a team, a week
directory, or a lineup. A roster is a different subject with a different source of truth (a file in
git, edited by hand) and a different failure mode (a typo shifts the lineup). Keeping them apart
means the page can depend on both without either depending on the other.

*Alternative considered:* a `roster.go` inside `internal/score`. Rejected — it would make the
scoring package the importer of filesystem paths and team names, and there is no call either
package would make on the other.

### Reading returns a whole roster, not a stream

`Read` (or `Open`) parses the entire file, validates it, and returns a value holding the ordered
records — or an error and nothing else. Rosters are nine to fifteen lines; there is no reason to
stream, and a partially-consumed roster is a lineup card with a hole in it.

Validation is therefore whole-file: the duplicate-id check needs every record before it can pass,
so it happens on the way out rather than per line.

*Alternative considered:* an iterator yielding records with errors interleaved. Rejected — it moves
the "is this roster usable" question to every caller, and there will be more than one caller.

### A malformed line is an error, where the script skipped it

`scores.sh` skips any line whose first field is empty. The package refuses the file instead, naming
the line number. This is a deliberate divergence from the script, and the reason is in the spec: a
skipped line moves every record after it up one position, so a mistyped line quietly promotes a
bench player into the starting nine and demotes a starter, with nothing in the report showing it.
Refusing the file is loud and immediately fixable.

The divergence is safe today because no roster file in the tree contains such a line, and the
script is not being changed to share this code.

### The lineup-tree root is a constructor parameter

The package holds the *layout* — `<root>/<season>/<week>/<team>.csv` — and the caller holds the
*root*. `cmd/server` passes `scripts/lineups`; tests pass `t.TempDir()`.

This is what makes the eventual `//go:embed` move cheap: relocating the tree changes the string one
caller passes, not the package. It also keeps tests off real roster files, so a lineup edit for a
real week cannot break the suite.

*Alternative considered:* a package-level default root constant. Rejected — it invites tests to
read the live tree, and it hardcodes a path we already know is going to move.

### Team names are single path segments, checked before use

A team name reaches path resolution from a URL segment once the page exists. Rejecting any name
containing a separator or a parent reference keeps the check next to the layout it protects rather
than in each future caller. The season and week are integers, so they need no equivalent check.

### The starters/bench split is a view over the records, not a second parse

The split takes a parsed roster and returns two ordered slices. Nine is a named constant with the
league rule as its comment, not a literal in a loop. A roster shorter than nine yields an empty —
not nil-versus-empty-ambiguous — bench, matching the `roster-score-report` requirement that the
bench heading prints regardless.

Splitting stays separate from parsing so that the position/team columns, when they arrive, change
the record shape without touching the lineup rule.

### Record carries `ID` and `Name`, and nothing else yet

Two fields, both strings. No position, no team, no points. The name is documented as display-only
at the type, because "the server never sees it, so a wrong name pairs silently with the wrong
stats" is exactly the kind of non-obvious constraint a comment should carry.

## Risks / Trade-offs

- **Duplicated roster knowledge in bash and Go until the page lands** → Accepted deliberately and
  scoped in the proposal. The alternative — a `go run` shim inside `scores.sh` — adds a build step
  to the interim UI and a CLI surface we would delete. If the two ever disagree, the Go package is
  the one that is specified and tested.
- **The package refuses files the script accepts** → Only for lines that yield no id, which no
  roster in the tree has. Should one appear, the script silently mis-lineups and the package says
  which line and why; that is the right asymmetry.
- **Refusing duplicate ids is stricter than anything the script does** → A duplicate is
  unambiguously wrong (double-counted in the starters' total, undisclosed), so refusing costs
  nothing real. If a legitimate reason to repeat an id ever appears, the check is one function.
- **Nine is hardcoded, and league lineup size could change** → It is a named constant with the rule
  written next to it, so a change is a one-line edit with an obvious blast radius. Making it
  configurable now would be a parameter no caller has a reason to vary.
- **Tests could drift from the real roster files** → Fixtures live in `internal/roster/testdata`
  and are written to mirror the real files' shape (leading comment line, `id,name` records). The
  real tree stays out of the suite on purpose.

## Migration Plan

Additive: a new package, no existing caller. Nothing to deploy, nothing to roll back. The follow-on
backlog items (matchup resolution, the served page) are the first consumers.

## Open Questions

- Does the eventual `//go:embed` move put the lineup tree under `internal/roster` or somewhere the
  page package owns? Not decided here; the constructor parameter keeps both open.
- When the CSV grows a position column, does the starters/bench split stay purely positional or
  start reading the label? The backlog's "Decide first" item settles that; this change assumes
  positional and says so in the spec.
