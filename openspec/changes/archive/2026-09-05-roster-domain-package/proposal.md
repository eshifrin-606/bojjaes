## Why

Roster knowledge — that a lineup file is `id,name` per line, that `#` and blank lines are not
records, that the file lives at `<lineups>/<season>/<week>/<team>.csv`, and that the first nine
records are the starting lineup — exists only inside `scripts/scores.sh`. Per the standing project
convention the shell scripts are an interim UI, and
[ADR 0004](../../../docs/adr/0004-web-frontend-stack.md) commits to serving `/{season}/{week}` from
the Go binary. That page needs the same three facts, and the only place to read them today is a
bash while-loop. Giving them a Go home is the first step of the backlog's "Roster domain in Go"
section and unblocks matchup resolution and the served page.

## What Changes

- Add an `internal/roster` package that is the Go home for roster and lineup knowledge:
  - **Parsing**: read an `id,name` roster file into ordered records, skipping blank and `#` comment
    lines, trimming surrounding spaces, and yielding a final record that has no trailing newline.
    A file with no records is an error, not an empty roster.
  - **Location**: resolve a season, week, and team name to a path under a configurable lineup-tree
    root, so the tree layout is stated once rather than rebuilt by each caller.
  - **Lineup shape**: split a parsed roster into the first nine records as starters and the
    remainder as bench, in file order, with no positional validation and no reordering.
- No change to scoring, to the HTTP endpoints, or to `cmd/server`.
- `scripts/scores.sh` and `scripts/fantasycast.sh` are deliberately left untouched and keep their
  own bash parsing. The duplication is accepted and time-boxed: it ends when the served page
  replaces the scripts, not by adding a CLI shim we would then delete.

## Capabilities

### New Capabilities

- `roster-source`: how a roster file is read and where it is found — the record format and what is
  not a record, the season/week/team layout of the lineup tree, and the positional rule that makes
  the first nine records the starting lineup. This is the source side of a roster; the
  `roster-score-report` capability remains the reading side.

### Modified Capabilities

<!-- None. roster-score-report and matchup-report describe terminal output that is unchanged by
     this change; their behaviour is restated, not revised, by roster-source. -->

## Impact

- **New code**: `internal/roster` (parse, path resolution, lineup split) with table-driven tests and
  `testdata` roster fixtures.
- **Unchanged**: `internal/score`, `cmd/server`, all HTTP endpoints, both shell scripts, and the
  on-disk format and location of `scripts/lineups/**`.
- **Dependencies**: none beyond the standard library.
- **Follow-on**: unblocks the backlog's matchup resolution from a week directory and the
  `GET /{season}/{week}` page. The `//go:embed` constraint — embed cannot cross `..` — will
  eventually force the lineup tree to move under a package; this change keeps the root a
  constructor parameter so that move is a caller change, not a rewrite.
