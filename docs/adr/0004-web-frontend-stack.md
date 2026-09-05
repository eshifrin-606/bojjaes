# ADR: Web Frontend Stack for the Fantasy Cast Scoreboard

**Status:** Accepted 2026-09-05.

Settles decision #7 of [ADR 0002](0002-live-scoreboard-backend.md) ("Hosting: local first,
cloud-ready") and refines its decision #4 (freshness). Everything else in ADR 0002 and
[ADR 0003](0003-sleeper-as-initial-stat-provider.md) — the provider interface, game-granular
fetch, hardcoded scoring, static lineup config, Sleeper as sole provider — remains in force.

## Context

The scoreboard works, but only as a terminal artifact on one machine:
`scripts/fantasycast.sh` pads two `scripts/scores.sh` reports into side-by-side columns against a
server running on `localhost:8080`. Per the standing project convention, `scripts/*.sh` are an
interim UI, not the destination.

What we want instead:

- A URL three people can open from a phone or laptop on Sunday.
- Scores that are live-ish without hammering Sleeper.
- No sign-in.
- A stack we would not regret if this later grows lineup submission or season-long roster stats.

Traffic is negligible: three regular viewers, on the order of 100 page loads a week, peaking maybe
10 a minute when a game is close.

Two things constrain the design more than the numbers do:

- **Sleeper is an undocumented API with unprobed rate limiting** (ADR 0003). Viewer-count and
  refresh-mashing must not translate into upstream request volume.
- **`fantasycast.sh` deliberately computes no margin.** A starter whose game has not kicked off is
  indistinguishable from one who was inactive, so a difference shown on Sunday morning would read
  as a settled result. Nothing in this change adds the per-player game state that would make a
  margin honest.

## Decision

Serve the scoreboard as **server-rendered HTML from the existing Go binary**, on a single origin,
hosted on Fly.io.

1. **Rendering: Go `html/template`.** No JavaScript framework, no build step, no bundler. The page
   is a template; the small amount of client-side behaviour it needs (a refresh timer) is a few
   lines of vanilla JS.

2. **Topology: one binary, one origin.** The same process serves the HTML page and the existing
   JSON endpoints (`GET /score`, `POST /scores`). No separate static host, no CORS, no per-
   environment API base URL. `scripts/scores.sh` and `scripts/fantasycast.sh` keep working
   unchanged against the same server.

3. **Assets embedded.** Templates, CSS, and `scripts/lineups/**` are pulled into the binary with
   `//go:embed`. A deploy is therefore also the lineup update, which is what we want while lineups
   remain hand-edited files in git.

4. **URL: `/{season}/{week}`,** e.g. `/2025/15`. The week directory
   `scripts/lineups/<season>/<week>/` holds exactly two rosters — `bojjaes.csv` and the opponent's
   — and the opponent is resolved as "the file that is not `bojjaes.csv`". The Bojjaes are always
   the left column. A directory that does not hold exactly two rosters is an error, not a guess.

5. **Freshness: client polls, server caches.** The page re-fetches roughly every 5 minutes, and
   only while the tab is visible. The server holds a short TTL cache (~5 min) over the Sleeper
   fetch, guarded by single-flight so concurrent misses collapse into one upstream call.

   No scheduler, no game-day window configuration, no background poller. Freshness is pull-driven:
   upstream load is bounded by the cache TTL rather than by viewer count, and drops to zero when
   nobody is looking.

6. **Hosting: Fly.io.** One binary, one Dockerfile, one region.

7. **Access: none.** No auth, no accounts. The scoreboard is served under an unguessable path
   prefix and is otherwise open. **Reads are public and writes do not exist.**

8. **Presentation carries the honesty constraint.** The page shows two equal columns of scored
   starters and their two totals, plus a visible **as-of timestamp**. The timestamp is **our
   Sleeper fetch time, labelled as such** — the REST weekly aggregate the web path reads carries
   no `updated_at`; only the GraphQL shapes do (ADR 0003), and changing fetch shape is not worth
   it for this alone. What it claims is "when we last asked," which is honest and is what the
   reader needs. Moving it to upstream `updated_at` later is an upgrade, not a correction. The
   page must not show a margin, a win probability, a progress bar, a leader highlight, or any
   winner-implying styling.

9. **Lineup files gain display fields.** The roster CSV format grows position and team alongside
   the existing id and name. These are **local labels only** — the Sleeper player ID remains the
   sole identity key, exactly as in `scripts/scores.sh`. They are rendered; they are never used to
   resolve a player.

## Rationale

- **`html/template` over a SPA.** Both extensions we can foresee — submitting a lineup, browsing
  season stats — are a form and some tables. Neither wants a component framework, and a framework
  would cost a build step, a second deploy artifact, and a second language for the life of the
  project. If interactivity ever outgrows templates, HTMX drops in without restructuring, and a
  JSON-consuming SPA remains reachable because the JSON API is not going away.

- **One origin over Cloudflare-Pages-plus-API.** The split was the intuitive shape, but at this
  traffic level a CDN in front of three phones buys nothing while costing CORS configuration, two
  deploy pipelines, and the ability to server-render. It also does not need to be decided now:
  putting Cloudflare in front of a single Go origin later is a DNS change.

- **Client polling over a server-side scheduler.** The requirement was stated as "hit our server
  every ~5 minutes during game hours," but a TTL cache delivers the same upstream profile with no
  clock to configure and no window to keep in sync with the NFL calendar. The pages being open
  *is* the schedule. Continuous Sunday polling also keeps the machine and its cache warm, so the
  cold path is paid roughly once a day rather than once a viewer.

- **Fly over Cloud Run.** Cloud Run is the easier first deploy, but it has no persistent disk. Both
  named extensions imply storage eventually, and a Fly volume with SQLite is a short, ordinary next
  step, where Cloud Run would mean Cloud SQL or Litestream. Hosting is the one choice here that is
  genuinely annoying to reverse, so it is made on the anticipated shape rather than the current one.

- **No auth now, and no design debt from it.** Reads carry nothing sensitive — it is a fantasy
  scoreboard. Keeping the surface read-only means the eventual write path can be gated by a single
  shared passphrase when it arrives, rather than requiring an account system be designed today.

- **The no-margin rule is a design constraint, not a preference.** It is the same reasoning
  `fantasycast.sh` already encodes, and it is more load-bearing on the web than in a terminal: a
  terminal report is obviously a snapshot the reader just produced, while a page left open on a
  phone has no such anchor. The as-of timestamp is the substitute anchor — and a fetch time
  serves that purpose, because what the reader is being warned about is a page that has gone
  quiet, which "last asked at 1:07pm" tells them exactly as well as an upstream timestamp would.

## Consequences

- **A lineup change requires a commit and a deploy.** Accepted deliberately: it keeps the whole
  system one artifact with no external store, no write path, and no auth. It also inherits ADR
  0002's known failure mode — a stale lineup file yields a confidently wrong scoreboard, now with a
  wider audience than one terminal.

- **The week directory convention becomes load-bearing.** `/{season}/{week}` resolving the opponent
  by "the other file" means adding a third roster to a week directory breaks that week's page. The
  convention is currently followed by every week in the tree; it is now enforced rather than
  incidental.

- **First load after an idle period pays a full Sleeper fetch** (~200 ms – 1.2 s per ADR 0003), and
  Fly machine auto-stop discards the in-process cache. Both are acceptable at this traffic; if
  either becomes annoying, the fixes (a minimum running machine, or moving the cache out of
  process) are additive.

- **Scores may still decrease and missing stats still read as zero** (ADR 0003). Neither is changed
  here, and both are now visible to more people. The as-of timestamp is the only mitigation in
  scope.

- **A fetch-time as-of overstates freshness by whatever the upstream lag is.** Bounded in practice
  by the endpoint's 30 s current-week edge TTL plus however long Sleeper takes to post a stat, but
  unmeasured live. The label is what keeps it honest; the measurement is a follow-up.

- **No margin means the reader does the subtraction.** This is a real ergonomic cost on the feature
  people most want, accepted until per-player game state makes the number honest.

- **The CSV format change touches `scripts/scores.sh`,** which currently parses `id,name`. The
  scripts and the web page must agree on the format, or the interim UI breaks.

- **Deploying makes the endpoint publicly reachable,** which turns our Sleeper usage from one
  laptop into a hosted service. The TTL cache is what keeps that from being a change in upstream
  behaviour; it is a correctness requirement of going public, not an optimisation.

## Follow-ups

- Decide the exact roster CSV field order and whether position denotes a **lineup slot** or the
  player's listed position — `scripts/scores.sh` currently derives the starting lineup from file
  order alone, and a rendered slot label that disagrees with file order would be worse than none.
- Watch the fetch-time as-of on a live Sunday and see how far it drifts from when stats actually
  move. If the gap is big enough to mislead, the fix is the GraphQL shape's `updated_at`.
- Probe Sleeper rate-limit tolerance at the deployed polling cadence (still open from ADR 0003).
- Revisit the margin once per-player game state lands; it is a presentation flip, not a redesign.
- Choose the unguessable path prefix and record where it lives, so it is not accidentally logged or
  committed into a public README.
