# Backlog

Coarse requirements to get from "scoring engine + shell scripts" to the web scoreboard in
[docs/adr/0004-web-frontend-stack.md](docs/adr/0004-web-frontend-stack.md).
One line each, roughly in dependency order. Not sized, not scheduled.

## Decide first

- Decide whether the roster CSV's new position field means lineup slot or the player's listed position, since a slot label that disagrees with file order is worse than no label at all.

## Roster domain in Go

- Move roster/lineup knowledge out of `scripts/scores.sh` and into a Go package: parse the CSV, know where the lineup tree lives, know that the first nine records are the starters.
- Resolve a matchup from a week directory: exactly two rosters, opponent is the file that isn't `bojjaes.csv`, three files is an error rather than a guess.
- Grow the roster CSV to carry position and team as display-only labels, and update `scores.sh` so the scripts and the page agree on the format.

## Serving the page

- Add `GET /{season}/{week}` rendering two equal columns of scored starters plus their two totals with `html/template`.
- Keep the page honest: as-of timestamp visible, and no margin, win probability, progress bar, or leader highlight anywhere in the markup or CSS.
- Stamp the as-of from our own Sleeper fetch time for now, labelled as such, and watch on a live Sunday how far it drifts from when the stats actually moved.
- Add the ~5 minute client refresh in a few lines of vanilla JS, and only while the tab is visible.
- Embed templates, CSS, and the lineup tree with `//go:embed` — which means the lineup files have to move somewhere a package can reach, since embed won't cross `..` or follow symlinks.
- Pick the unguessable path prefix and decide where it lives so it doesn't end up in logs or a public README.

## Not hammering Sleeper

- Put a ~5 minute TTL cache over the weekly fetch, with single-flight so concurrent misses collapse into one upstream call.
- Probe Sleeper's rate-limit tolerance at the cadence we're actually going to deploy at.

## Making the binary deployable

- Teach `main.go` to read `PORT`, use its own `http.Server` with timeouts instead of `DefaultServeMux`, and shut down gracefully.
- Drop the walking-skeleton `fmt.Printf` out of the `/score` handler.
- Dockerfile and `fly.toml`, one region, one machine.

## Later, deliberately

- Revisit the margin once we have per-player game state — it's a presentation flip, not a redesign.
- Season-long stats and lineup submission are the two extensions we're keeping the door open for; neither is in scope and neither should force a framework.
