## Why

Reading a head-to-head matchup today means running `scripts/scores.sh` twice and holding both
lineups in your head, or scrolling between two blocks of output. The question a manager actually
asks on Sunday — "how am I doing against them" — is a comparison, and the tool only answers half of
it at a time.

## What Changes

- New `scripts/fantasycast.sh <season> <week> <team> [team2]`. One team argument means that team is
  the Bojjaes' opponent, and the report shows Bojjaes against them; two arguments show those two
  teams. Bojjaes, or the first named team, is the left column.
- The two rosters print **side by side** — each column is a full `scores.sh` report (starters, total,
  bench) for one team, with the season and week printed once above both.
- `scripts/fantasycast.sh` composes the report by invoking `scripts/scores.sh` once per team and
  laying the two outputs out in columns. No roster parsing, scoring, or report logic is duplicated.
- **BREAKING** `scripts/scores.sh` no longer prints its `season <n> week <n>` header line. The line
  moves to `fantasycast.sh`, which prints it once for the matchup. A standalone `scores.sh` run now
  begins at the `STARTERS` heading.
- `scripts/scores.sh` prints each player's points in a fixed-width field, so the second column
  starts at the same offset on every line and the pasted columns align.
- No scoreboard, margin, or winner line. The two `TOTAL` lines sit on the same output line, which is
  the comparison; a computed margin would read as authoritative while starters are still missing
  stats.
- README gains a matchup section and its `scores.sh` example output loses the header line.

## Capabilities

### New Capabilities
- `matchup-report`: how two rosters are selected, arranged into columns, and aligned for a single
  season and week — including the one-argument opponent form and what happens when the two rosters
  are different lengths.

### Modified Capabilities
- `roster-score-report`: the report no longer carries a season/week header of its own, and its
  player lines use a fixed-width points column so the report can be composed into a wider layout.

## Impact

- `scripts/fantasycast.sh` (new), `scripts/scores.sh` (header removed, points field padded).
- `README.md` — `scores.sh` example output and a new matchup section.
- No Go changes. Each team is scored by its own `POST /scores` request, one per `scores.sh`
  invocation, so the 26-player-ID cap in `internal/score/batch.go` applies per roster rather than
  per matchup.
- No new dependencies; still `curl` and `jq`. The column layout is done with `printf` in bash 3.2,
  which is what macOS ships.
