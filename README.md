# Summary

This repo is to assist fantasy football team owners in the he man fantasy football league.

Bojjaes on top.

## Local setup

Requires Go 1.26.5 (see `go.mod`). No other dependencies, env vars, or config.

Run the server:

```bash
go run ./cmd/server
```

It logs `listening on :8080` and stays in the foreground (Ctrl-C to stop).

Smoke test: with the server running, open http://localhost:8080/2025/14.

### The matchup page

`GET /{season}/{week}` serves that week's matchup as an HTML page: the two teams
of that week's lineup directory, their nine starters with points, and each
lineup's total. The Bojjaes always hold the left column.

```
open http://localhost:8080/2025/15
```

No team appears in the URL — the week directory in the lineup tree names both.
A week that is not exactly one Bojjaes matchup is refused rather than guessed
at: a week we never played is a `404`, a malformed week directory is a
`500`, and a failed upstream fetch is a `502` rather than a page of zeros.

Lineups are hand-edited at `internal/lineup/data/<season>/<week>/<team>.csv`.
The tree is compiled into the binary with `//go:embed`, so the server reads no
file off the working directory — and an edited lineup reaches the page only
after a rebuild.

The page shows both totals and nowhere shows their difference. A starter whose
game has not kicked off is indistinguishable from one who was inactive, so it
carries no margin, win probability, or leader highlight.

A starter Sleeper has no stats for shows no number, not a zero — it adds
nothing to the total, and the absence is not the same claim as a real `0`.

Tests and compile check:

```bash
go test ./...
go build ./...
```
## Design docs

[docs/package-dependencies.md](docs/package-dependencies.md) maps how the Go packages relate — a
diagram, each package's imports, and where the next dependency is likely to land. Start there
before deciding where new code belongs. Decisions behind the current shape are in
[docs/adr/](docs/adr/), and the league's scoring rules in [docs/scoring.md](docs/scoring.md).
