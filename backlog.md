# Backlog

Coarse requirements to get from "scoring engine + shell scripts" to the web scoreboard in
[docs/adr/0004-web-frontend-stack.md](docs/adr/0004-web-frontend-stack.md).
One line each, roughly in dependency order. Not sized, not scheduled.

## Decide first

- [ ] Decide whether the roster CSV's new position field means lineup slot or the player's listed position, since a slot label that disagrees with file order is worse than no label at all.

## Roster domain in Go

- [x] Move roster/lineup knowledge out of `scripts/scores.sh` and into a Go package: parse the CSV,
  know where the lineup tree lives, know that the first nine records are the starters. Done as
  `internal/roster`. The bash parsing in `scores.sh` deliberately stays until the served page
  replaces the scripts.
- [x] Resolve a matchup from a week directory: exactly two rosters, opponent is the file that isn't `bojjaes.csv`, three files is an error rather than a guess.
- [ ] Grow the roster CSV to carry position and team as display-only labels, and update `scores.sh` so the scripts and the page agree on the format.

## Serving the page

- [x] Split `internal/score` into `internal/score` (domain), `internal/sleeper` (adapter), and
  `internal/api` (JSON), with `cmd/server` the only package naming a provider, so that a second
  consumer can score a roster from one fetch without going through HTTP.
- [x] Add `GET /{season}/{week}` rendering two equal columns of scored starters plus their two totals with `html/template`.
- [ ] Keep the page honest: as-of timestamp visible, and no margin, win probability, progress bar, or leader highlight anywhere in the markup or CSS.
- [ ] Stamp the as-of from our own Sleeper fetch time for now, labelled as such, and watch on a live Sunday how far it drifts from when the stats actually moved.
- [x] Add the ~5 minute client refresh in a few lines of vanilla JS, and only while the tab is visible.
  The rendered contract is tested; the browser behaviours are still unobserved (see
  `openspec/changes/refresh-page-while-visible/notes.md`).
- [ ] Embed templates, CSS, and the lineup tree with `//go:embed` — which means the lineup files have to move somewhere a package can reach, since embed won't cross `..` or follow symlinks.
- [ ] Pick the unguessable path prefix and decide where it lives so it doesn't end up in logs or a public README.

## Not hammering Sleeper

- [x] Put a ~5 minute TTL cache over the weekly fetch, with single-flight so concurrent misses collapse into one upstream call. `internal/statscache`, wrapped once in `main` so both handlers share one budget.
- [ ] Probe Sleeper's rate-limit tolerance at the cadence we're actually going to deploy at.
- [ ] Log cache misses in `internal/statscache` — season, week, and elapsed — so `fly logs` shows the
  actual Sleeper budget. Log the miss, not the hit: a miss is ~one line per week per TTL and stays
  readable, while hits arrive with every reader's refresh and would bury it.

## Making the binary deployable

- [ ] Teach `main.go` to read `PORT`, use its own `http.Server` with timeouts instead of `DefaultServeMux`, and shut down gracefully.
- [x] Drop the walking-skeleton `fmt.Printf` out of the `/score` handler. Done by deleting
  `GET /score` outright in the package split: the fixed player and week it proved is now a test in
  `internal/sleeper`, and `POST /scores` covers what the endpoint did.
- [ ] Dockerfile and `fly.toml`, one region, one machine.

## Later, deliberately

- [ ] Revisit the margin once we have per-player game state — it's a presentation flip, not a redesign.
- [ ] Season-long stats and lineup submission are the two extensions we're keeping the door open for; neither is in scope and neither should force a framework.
