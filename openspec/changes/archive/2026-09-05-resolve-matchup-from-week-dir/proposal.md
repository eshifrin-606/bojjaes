## Why

Every consumer of a matchup has to be told who is playing. `scripts/fantasycast.sh` takes the
opponent as an argument and defaults the left column to `bojjaes`; the backlog's
`GET /{season}/{week}` page has no argument to take one from — the URL carries a season and a week
and nothing else. The week directory already holds the answer: `scripts/lineups/2025/14/` contains
exactly `bojjaes.csv` and `wood.csv`, and has for every week in the tree. Reading the matchup out of
the directory is what lets a URL with two path segments render a two-column page.

The reason to do it now is that `internal/roster` just became the one place that knows the lineup
tree's layout. A week directory holding exactly two rosters is a fact about that layout, and it
belongs next to the layout rather than in the first handler that needs it.

## What Changes

- Add matchup resolution to `internal/roster`: given a season and a week, list that week's
  directory and return the two team names — ours first, then the opponent.
- Define what a well-formed week directory is, and make every departure from it an error that names
  what it found:
  - exactly two roster files, one of which is the Bojjaes'
  - three or more files is an error, not a guess at which pair was meant
  - one file, an empty directory, or a missing directory is an error
  - two files with no `bojjaes.csv` is an error — this resolver answers "who are we playing",
    which is not a question a directory of two other teams can answer
- Resolution returns names only. Reading and validating those rosters stays `Read`'s job, so the
  directory-shaped failures above stay separate from the file-shaped ones (`bad line`,
  `duplicate id`, `no records`) that `Read` already reports.
- No change to scoring, to the HTTP endpoints, to `cmd/server`, or to either shell script.

## Capabilities

### New Capabilities

<!-- None. -->

### Modified Capabilities

- `roster-source`: add the week directory as a locatable thing in its own right. The capability
  currently resolves a roster only when the caller already names the team; this adds resolving the
  pair of teams from the directory itself, and states what makes a week directory well-formed.

<!-- matchup-report is NOT modified. It governs the terminal report's arguments, columns, and
     alignment, none of which change: the report still takes one or two team names. Whether a
     caller may omit them entirely is a question for the page, and the page is a later change. -->

## Impact

- **New code**: matchup resolution in `internal/roster`, with tests over `t.TempDir()` week
  directories.
- **Unchanged**: `internal/score`, `cmd/server`, every HTTP endpoint, `scripts/scores.sh`,
  `scripts/fantasycast.sh`, and the on-disk contents of `scripts/lineups/**`.
- **Dependencies**: none beyond the standard library.
- **Constraint introduced**: the tree's weeks must each hold exactly two rosters. That is already
  true of all three weeks in the tree, and this change makes it a checked rule rather than a
  coincidence. A bye week, or a week staged early with only our own roster in it, will now fail
  loudly where nothing looked at the directory before.
- **Follow-on**: unblocks `GET /{season}/{week}`, which is the reason names-only is enough — the
  handler reads both rosters itself and needs the names anyway for the column headings that
  `scores.sh` never printed.
