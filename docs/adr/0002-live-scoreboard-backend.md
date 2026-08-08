# ADR: Live Scoreboard Backend (Bojjaes vs. Opponent)

**Status:** Accepted

## Context

The first feature in this project is a backend that serves **live fantasy scores** for a
head-to-head matchup: the Bojjaes' starters versus the opponent's starters for the current
week. It is a personal tool for the co-managers, not a public service.

Product constraints:

- Data no more than ~5 minutes stale is acceptable ("somewhat realtime").
- Near-term scope is only the two starting lineups (~18 players), which change week to week.
- The served output is **fantasy points only**, computed under HMFFL's custom scoring from
  raw NFL stats pulled from a third-party source.
- Traffic is tiny (a couple of managers on game days).

Key facts that shaped the design:

- The natural fetch unit of NFL stat sources is a **game** (a full box score), not a single
  player. The ~18 relevant players span most of a given Sunday's games, so fetching at game
  granularity does not add meaningful upstream load versus fetching per-player.
- No stat source is obviously correct yet. The live-serving shortlist is the ESPN unofficial
  API, the Sleeper public API, and MySportsFeeds (a documented, developer-oriented freemium
  API); nflverse is a candidate validation oracle, not a live-serving source. None has been
  probed against live data.
- Player-ID mapping (matching an upstream source's player identifiers to the names in our
  lineup config) is expected to be the fiddliest part of the build.

## Decision

Build a small **Go** backend that serves the live matchup scoreboard, structured around a
**swappable data-source provider interface**. Specifically:

1. **Provider interface.** Define an internal interface for fetching raw NFL stats. All
   source-specific code (HTTP shape, player IDs, parsing) lives behind it. The scoring and
   serving layers depend only on the interface, never on a concrete vendor.

2. **Data source: deferred.** Do **not** canonize a vendor in this ADR. Start with a single
   free provider chosen after a game-day probe of the candidates. Selection criteria, roughly
   in priority order: live freshness within budget, player-ID mapping ergonomics, raw-stat
   coverage for our scoring, and developer-documentation quality (a real but secondary factor).

3. **Fetch at game granularity, serve narrowly.** Fetch whole games (all players in the
   relevant games) and scope the served output to the ~18 configured starters. This decouples
   upstream fetching from lineup decisions and leaves room for future features (bench,
   other matchups, waiver watch) at negligible extra upstream cost.

4. **Freshness: on-demand fetch + short TTL cache (~5 min).** Reads trigger a fetch when the
   cache is cold/expired; results are cached in-process. No database in the MVP.

5. **Scoring: hardcoded (MVP).** Encode HMFFL scoring directly in code for now; extract to a
   config format later if the rules churn.

6. **Lineups: static weekly config file.** A hand-edited file names the Bojjaes and opponent
   starters for the current week. No integration with the existing spreadsheet yet.

7. **Hosting: local first, cloud-ready.** Run on a manager's machine to start, but avoid any
   design choice that blocks a later move to a small cloud service (e.g., no reliance on local
   filesystem state beyond the config file, keep the cache abstraction replaceable).

## Rationale

- **Provider interface over vendor choice.** The source is the least certain and most
  reversible decision. An interface makes it cheap to switch — or to run a hybrid (e.g., one
  source for ID mapping, another for live stats) — and lets the ADR stay honest about the fact
  that nothing has been probed live.
- **Game-granular fetch.** Because upstream cost is per-game, narrowing the *served* players
  saves response size and compute (negligible here) but not calls. Fetching games and
  filtering at read time is simpler and more extensible for no real cost.
- **On-demand + TTL over background polling.** Simplest thing that meets the freshness budget
  at this traffic level; no scheduler or always-on process required. A DB/background poll is a
  known escape hatch (see Consequences), not an upfront cost.
- **Hardcoded scoring / static lineups.** These are the fastest paths to a working feature and
  each has an obvious later refactor. They keep the MVP focused on the live-scoring core.
- **Go.** Single-binary builds and a cheap cloud footprint suit an always-available-later
  personal service; the data munging here is light enough that Go's ergonomics are not a
  bottleneck.

## Consequences

- The concrete data source is an open decision to be settled by a **game-day probe** of a
  3-way live shortlist — **ESPN unofficial, Sleeper, and MySportsFeeds** — compared on
  freshness, ID mapping, and stat coverage, with **nflverse** used as a validation oracle to
  check our scoring. SportsData.io is the documented fallback if MySportsFeeds' free tier
  disappoints. Until the probe settles it, no code depends on a specific vendor.
- **Player-ID mapping** is the primary implementation risk. Sources with a published
  cross-ID dictionary (e.g., Sleeper) may be valuable as a mapping layer even if live stats
  come from elsewhere.
- **On-demand fetch has a known failure mode:** if per-read latency is poor or upstream call
  volume grows uncomfortable, introduce a small store/cache tier (DB) and/or a background
  refresh. The provider interface and cache abstraction are designed so this is an additive
  change, not a rewrite.
- **Hardcoded scoring** means rule changes require a code edit and redeploy until we extract a
  config format.
- **Static lineup config** must be updated by hand each week; a stale file yields a wrong
  scoreboard. Integrating with the spreadsheet as source of truth is a deliberate later step.
- Depending on an unofficial upstream (if chosen) carries the risk of it changing without
  notice; the provider interface localizes the blast radius of such a break.

## Follow-ups

- Probe the live shortlist (ESPN unofficial, Sleeper, MySportsFeeds) against real (last-season
  or live) data, using nflverse to validate scoring; record findings in Basic Memory and
  promote the chosen provider + rationale back into this ADR or a successor.
- Confirm HMFFL's exact scoring categories against the candidate sources' raw-stat payloads
  before finalizing the provider.
