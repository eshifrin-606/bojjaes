# Summary

This repo is to assist fantasy football team owners in the he man fantasy football league.

Bojjaes on top.

## Local setup

Requires Go 1.26.5 (see `go.mod`) and [direnv](https://direnv.net) to load
local env vars:

```bash
brew install direnv
printf '\neval "$(direnv hook zsh)"\n' >> ~/.zshrc   # then open a new shell
direnv allow                                   # in the repo root; again whenever .envrc changes
```

`.envrc` loads the committed `config/local.env`, then the gitignored
`.env.local` if present, so personal overrides go in `.env.local`.

Run the server:

```bash
go run ./cmd/server
```

It logs `listening on :8080` and stays in the foreground (Ctrl-C to stop).

Smoke test: with the server running, open http://localhost:8080/2025/14.

### Run with a local volume

`.env.local` sets `LINEUP_VOLUME=.volume`, so `go run ./cmd/server` from the
repo root serves lineups from `.volume/lineups` instead of the embedded tree,
and `/lineup` gets a Save button. Boot seeds `.volume/lineups` from git
(`internal/data/lineups`), and saves land there. `rm -rf .volume` resets the
store; the next boot re-seeds it.

Without direnv, set it for one run:

```bash
LINEUP_VOLUME=.volume go run ./cmd/server
```

### The matchup page

`GET /{season}/{week}` serves that week's matchup as an HTML page: the two teams
of that week's lineup directory, their nine starters with points, and each
lineup's total. The Bojjaes always hold the left column.

```
open http://localhost:8080/2025/15
```

No team appears in the URL — the week directory in the lineup tree names both.
A week holding only one side's lineup shows the other as `TBD`. Any other week
that is not one Bojjaes matchup is refused rather than guessed at: a week we
never played is a `404`, a malformed week directory is a
`500`, and a failed upstream fetch is a `502` rather than a page of zeros.

Lineups are hand-edited at `internal/data/lineups/<season>/<week>/<team>.csv`.
The tree is compiled into the binary with `//go:embed`. Without a volume the
server reads no file off the working directory, so an edited lineup reaches the
page only after a rebuild. With `LINEUP_VOLUME` set, boot seeds the volume from
that tree and lineups saved from the form are served from it (see above).

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
