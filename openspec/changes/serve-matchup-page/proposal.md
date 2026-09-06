## Why

The scoreboard is still a terminal artifact: `scripts/fantasycast.sh` pads two `scripts/scores.sh`
runs into side-by-side columns on one machine. [ADR 0004](../../../docs/adr/0004-web-frontend-stack.md)
settled the destination — server-rendered HTML from the existing Go binary at `/{season}/{week}` —
and the roster domain needed to get there now exists in Go (`internal/roster` parses a roster and
resolves a week directory to its two teams). This change is the first page: the same two columns,
served over HTTP.

## What Changes

- Add `GET /{season}/{week}` to the server, rendering an HTML page for that week's matchup.
- The page shows two equal columns of scored starters — the Bojjaes always on the left, the
  opponent on the right — each column a team name, its nine starters with points, and that
  lineup's total.
- Rendering is Go `html/template`, parsed from a template file embedded in its own package. No
  JavaScript, no build step.
- Wire the existing pieces behind the handler: `roster.Tree.Matchup` picks the two teams,
  `roster.Tree.Read` and `Roster.Starters` give the lineups, one `WeekSource.Week` fetch scores
  both columns, and a starter the provider has no entry for renders as a dash contributing nothing.
- `main.go` constructs the lineup tree and the provider client, and registers the route alongside
  `POST /scores`, which is unchanged.

Out of scope, each its own backlog line: the as-of timestamp, the ~5 minute client refresh, the
TTL/single-flight cache over the Sleeper fetch, embedding the lineup tree and CSS, the unguessable
path prefix, and the roster CSV's position/team display fields. The page ships with no margin, win
probability, progress bar, or leader highlight — that is a constraint on what this change may add,
not a feature it delivers.

## Capabilities

### New Capabilities

- `matchup-page`: the served HTML matchup page — the URL, how a season and week resolve to two
  columns, what each column contains, and how a week that is not a matchup is refused.

### Modified Capabilities

None. `matchup-report` governs the terminal report and is untouched; `roster-source` and
`player-week-score` are consumed as they stand.

## Impact

- `cmd/server/main.go`: a second route, plus the lineup tree root the handler needs.
- New package for the page handler and its template; it depends on `internal/roster` and on the
  scoring path in `internal/score`.
- `internal/score` and `internal/sleeper`: consumed as `split-score-packages` leaves them. That
  change is a prerequisite — it built the `score.Week` seam this page scores through.
- No change to `scripts/*.sh`, the roster CSV format, or `POST /scores`.
