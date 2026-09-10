# Backlog

What's left to get the web scoreboard of
[docs/adr/0004-web-frontend-stack.md](docs/adr/0004-web-frontend-stack.md) from "runs on my laptop"
to "a URL three people open on Sunday". Completed work has been dropped; the archived changes under
`openspec/changes/archive/` are the record of it.

The page itself is done — `GET /{season}/{week}` renders both columns, the totals, the as-of
timestamp, and the visibility-gated refresh, all behind a single-flight TTL cache. Everything below
is either a prerequisite of running it somewhere else or a thing we can only learn once it's there.

One line each, roughly in dependency order. Not sized, not scheduled.

## Blocking the first deploy

- [ ] Teach `main.go` to read `PORT`, serve from its own `http.Server` with read/write/idle timeouts
  instead of the package-level `DefaultServeMux`, and shut down gracefully on `SIGTERM` — which is
  how Fly stops a machine.
- [ ] Correct the `fly launch`-generated `Dockerfile`: build `./cmd/server`, not `.` (the generated
  `go build .` fails — there is no main package at the repo root); `CGO_ENABLED=0` for a static
  binary; scratch or distroless final stage in place of `debian:bookworm`; copy `go.sum` alongside
  `go.mod` so the build stays correct once there is a dependency. The lineups are embedded, so
  nothing is read off the working directory.
- [ ] `fly.toml` is generated and mostly right — `ord`, `auto_stop_machines = 'stop'`,
  `min_machines_running = 0`, 256mb. What is missing is an `[[http_service.checks]]` the platform
  can hit that doesn't cost a Sleeper fetch.

## Getting it in front of people

- [ ] Decide what `/` and an unknown or malformed `/{season}/{week}` do. Today a week directory that
  isn't exactly two lineups is an error; deployed, that error is what a reader sees, so it needs to
  read as a page rather than a stack trace.
- [ ] First deploy, then open it on a phone: this is the first time the page is read on the device
  it was designed for.
- [ ] Work through the browser observations the refresh change deferred —
  `openspec/changes/refresh-page-while-visible/tasks.md` §5, which are exactly the six checks that
  need a real browser and a server whose log you can watch.

## Once it's up

- [ ] Probe Sleeper's rate-limit tolerance at the cadence we're actually deployed at, now that
  "our cadence" is a real number rather than a guess. Still open from ADR 0003.
- [ ] Watch on a live Sunday how far our fetch instant drifts from when the stats actually moved
  upstream — procedure in `openspec/changes/archive/2026-09-07-stamp-as-of-timestamp/notes.md`. If the gap misleads,
  the fix is the GraphQL shape's `updated_at`.
- [ ] Grow the lineup CSV to carry position and team as display-only labels, and update `scores.sh`
  so the scripts and the page agree on the format. Position is the player's listed position, not
  the lineup slot (ADR 0004 decision 9) — slot stays implied by file order. Field order is the one
  piece still open, and it has to be settled in one place because the parser and the script both
  read it. Not deploy-blocking; the page is honest without these labels.

## Later, deliberately

- [ ] Revisit the margin once we have per-player game state — it's a presentation flip, not a redesign.
- [ ] Season-long stats and lineup submission are the two extensions we're keeping the door open for; neither is in scope and neither should force a framework.
