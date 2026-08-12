# ADR: Sleeper as the Initial Stat Provider

**Status:** Accepted 2026-08-09. **Amended 2026-08-12 — the accepted costs below are obsolete.**

Supersedes decision #2 ("Data source: deferred") of
[ADR 0002](0002-live-scoreboard-backend.md). Everything else in ADR 0002 — the provider
interface, game-granular fetch, on-demand serving, hardcoded scoring, static lineup config,
local-first hosting — remains in force.

> **Read this first.** The core decision — *Sleeper, sole provider, behind the ADR 0002 provider
> interface* — is unchanged and strengthened. But this ADR was written on the belief that Sleeper
> had no play-by-play and a one-hour cache. **Both were wrong.** Sleeper's GraphQL API is uncached
> and carries full PBP, which dissolves every trade-off in "Accepted inaccuracies" and the entire
> freshness risk. Jump to
> [Amendment 2026-08-12](#amendment-2026-08-12--the-accepted-costs-are-recoverable) for what we
> actually build. The original reasoning is kept below because the reversal is instructive, but
> **do not implement from the Decision section without reading the amendment.**

## Context

ADR 0002 deliberately declined to name a stat vendor, deferring to a game-day probe. That probe
ran on 2026-08-08/09 against ESPN's unofficial API and Sleeper's public API and is written up in
[docs/probe-espn-sleeper.md](../probe-espn-sleeper.md). Its findings, in brief:

- **Sleeper covers 19/19 raw stats we need from aggregates alone**, in a single ~570 KB request
  per week for the entire league. It pre-computes distance-bucketed bonus stats at exactly our
  thresholds (`pass_td_40p`, `rec_td_40p`, `rush_td_40p`, `fgm_50p`), handles half-sacks, and
  carries full-league IDP coverage.
- **ESPN's box score covers ~14/19** and needs its play-by-play endpoint for five rules, at
  roughly 26 calls per Sunday slate.
- The **hybrid plan (Sleeper for ID mapping, ESPN for stats) is not viable** — Sleeper's
  `espn_id` is populated for only 37% of productive players.
- Two problems survive in Sleeper: it **cannot express the 4-point forced-fumble rule** (its
  `idp_ff` is a raw count; paying off it overpays 44% of forced fumbles, ~162 a season), and its
  stats endpoint carried a **one-hour Cloudflare edge TTL** when measured in the offseason —
  12× the ~5-minute freshness budget in ADR 0002.

The freshness measurement is unproven for live games (it was taken on historical weeks in the
offseason, and Sleeper demonstrably tunes TTL per endpoint). It remains the largest open risk.

> **2026-08-12:** that hedge was right, and it broke in our favour. Sleeper tunes TTL *by week
> recency* — 3600 s is the completed-week policy; the current week gets 30 s. Both "problems that
> survive" above are now solved. See the
> [amendment](#amendment-2026-08-12--the-accepted-costs-are-recoverable).

## Decision

**Use Sleeper as the sole stat provider for the MVP**, behind ADR 0002's provider interface.

We are explicitly choosing a working single-backend implementation over an optimal one. One
backend is simpler than two, and Sleeper wins on every axis the probe measured except forced
fumbles. The specific trade-offs we are accepting:

### Accepted inaccuracies

> ⛔ **Superseded 2026-08-12.** All three rows are fixable from Sleeper's GraphQL PBP. See the
> [amendment](#amendment-2026-08-12--the-accepted-costs-are-recoverable). Retained as the record
> of what we were prepared to give up.

| # | Rule | Sleeper's behavior | Direction | Decision |
| --- | --- | --- | --- | --- |
| 1 | Forced fumble resulting in turnover (4) | `idp_ff` is a raw count | **Over**pays ~44% of FFs | Award it, but **display it as provisional** (see below) |
| 2 | 40+ yard bonus on defensive / return TDs (1) | No distance bucket known for these | **Under**pays 1 pt | Accept. Rare, low value |
| 3 | Safety, solo credit only (2) | `idp_safe` may include shared credit | Would **over**pay | **Exclude safeties from scoring entirely.** Prefer a known omission to a possible wrong award |

Rules 2 and 3 are rare and cheap; correctness of the common path matters more than completeness
of the tail. Rule 3 is a one-line flip if we later confirm `idp_safe` semantics.

### Forced fumbles are surfaced as provisional

> ⛔ **Superseded 2026-08-12.** We can compute the rule correctly, so there is no error to flag.
> Do not build the provisional-display path.

The FF error is **systematic and one-directional** — always an overpay, never an underpay, on
plays that are memorable and argued about. Silently folding it into a total would make our
scoreboard consistently disagree with the league's spreadsheet in the same direction, which
erodes trust in the tool faster than random error of the same size.

So: when a scored starter has `idp_ff > 0`, the 4 points are awarded but **flagged in the served
output as provisional** rather than absorbed into the total without comment.

### Freshness risk is accepted, with one cheap investigation first

> ⛔ **Superseded 2026-08-12.** The investigation ran and the risk evaporated. Note the one
> conclusion below that *survives*: ADR 0002's ~5-minute cache TTL is still not load-bearing, and
> the served output should still carry an as-of timestamp — now sourced from Sleeper's own
> `updated_at` rather than our fetch time.

We accept that the served scoreboard may lag real time by up to an hour if Sleeper's in-season
edge TTL matches the offseason value. Consequences:

- **ADR 0002's ~5-minute cache TTL is not load-bearing** and should not be defended as a design
  constraint. Behind a long edge cache, our local cache is decoration. Cache design is free to
  change; treat the TTL as a tuning knob, not a contract.
- The served output should carry an **as-of timestamp** rather than implying live data.

Before accepting this permanently, we will investigate whether a **fresher Sleeper surface
exists** — the Sleeper app shows live scoring, so some Sleeper endpoint is fresh, likely a
per-league matchup or scores surface rather than the global weekly stats dump. This stays within
"one backend." Cache-busting against the existing endpoint is also untested.

### Live-data semantics

- **A missing stat key means zero.** This is correct for completed weeks and merely lossy
  mid-game (a player whose data has not landed reads as 0). Accepted for the MVP. The known
  failure mode — overwriting real accumulated points with 0 because the feed broke — is deferred;
  fixing it means adding a store that refuses to regress a nonzero score, which is the DB tier
  ADR 0002 already names as an escape hatch.
- **Scores are allowed to decrease.** Stat corrections (sack reassignments, reversed fumble
  rulings) will move totals down. We want the most accurate figure available at any instant, not a
  monotonic one.

### Player-ID mapping

The lineup config is keyed on **first name + last name + team + position**, not on Sleeper IDs.
Resolution to `sleeper_id` happens in a mapping layer at the provider boundary.

This costs a little more than storing Sleeper IDs directly, and buys the thing that matters: the
config does not become Sleeper-shaped. Sleeper IDs are fine to use freely *inside* the Sleeper
provider, where they are already the natural key; they must not leak into config, scoring, or the
served output.

The mapping layer must handle the hazards the probe found: Sleeper **strips name suffixes**
(stores `Marvin Harrison`, not `Marvin Harrison Jr.`), 445 names carry apostrophes/hyphens/periods
requiring punctuation normalization, and there are **83 exact full-name collisions** among active
players — which is precisely why team and position are part of the key rather than name alone.

### Scope

- **Regular season only.** No postseason support; the Sleeper path is `/regular/`.
- **ESPN play-by-play stays on the shelf as a verification tool**, not a second provider.
  *(Amended 2026-08-12: still true, but weaker — the FF check it was reserved for is no longer
  needed. ESPN is now an optional independent oracle, with no planned use.)* Later,
  we may use it to check the FF rule against a small number of fumble plays (~20 a week
  league-wide, far fewer for our starters). If it graduates from verification to live use, it is a
  narrow enrichment step behind the provider interface — not a second backend.

## Amendment 2026-08-12 — the accepted costs are recoverable

The top follow-up below ("investigate whether a fresher Sleeper surface exists") ran on
2026-08-12. It found one, and the finding is larger than the question: **Sleeper has an
undocumented GraphQL API at `POST https://api.sleeper.app/graphql` — no auth, introspection
enabled — that is uncached and carries full play-by-play.** Evidence and measurements are in
[docs/probe-espn-sleeper.md](../probe-espn-sleeper.md) Tier 5.

This does not change the decision. It changes its price. Everything this ADR bought with an
accepted inaccuracy is now available for free.

### What we build instead

| Original accepted cost | Amended position |
| --- | --- |
| #1 FF overpay (~162/season), displayed as provisional | **Compute it correctly.** The fumbler's PBP row carries `fum_lost: 1.0` exactly when the fumble was lost to the defense. All 20 FF plays in 2025 wk 1 classify correctly against the gamebook text (13 turnover-qualified); own-team recoveries and out-of-bounds fumbles read as non-turnovers. **No provisional flag.** |
| #2 No 40+ bonus on defensive / return TDs | **Award it.** `idp_int_ret_yd` / `idp_fum_ret_yd` on the scorer's PBP row give raw return distance. Sleeper's own buckets are 50+ only (`bonus_def_int_td_50p`, `bonus_def_fum_td_50p`) — wrong threshold, so compute ours from the raw yardage. |
| #3 Safeties excluded entirely | **Award them.** `idp_safe` appears on the play alongside `idp_tkl_solo` rather than `idp_tkl_ast`, so solo credit is decidable per play. Caveat: only 2 instances observed — safeties are rare. Verify against a shared-credit safety before trusting the distinction. |
| Freshness: up to 1 h stale, accepted | **Gone.** GraphQL is uncached (`max-age=0, private, must-revalidate`, `cf-cache-status: DYNAMIC`). Separately, the REST 3600 s TTL turned out to be a *completed-week* policy — the current week is served at `s-maxage=30` — and cache-busting the REST endpoint reaches origin anyway (`MISS`). Three independent answers. |

### Attributing the forced fumble

The one genuine wrinkle. PBP gives the forcer no stat of their own, but names them in
`metadata.description` (`"forced by E.Oliver"`) — and the forcer is almost always already among
that same play's `play_stats` rows under a tackle or sack key. Matching the parsed name against
**only that play's 2–6 rows** resolved 20/20 to a native Sleeper `player_id`. This is a bounded
match, not the roster-wide fuzzy join the probe's Tier 4 warned about. Volume is ~20 FF plays per
week league-wide.

### ⚠️ `idp_ff` means opposite things in the two feeds

The most dangerous implementation detail in this ADR. In the REST weekly aggregate, `idp_ff`
credits the **forcer**. In GraphQL `play_stats`, it sits on the **fumbler**. Verified on
2025 wk 1 BAL@BUF: the aggregate gives Ed Oliver `idp_ff=1` and Derrick Henry nothing; PBP gives
Henry `idp_ff=1` and Oliver only tackle keys.

Any code that reuses aggregate-derived parsing against the PBP feed **will credit forced fumbles
to the offense.** Read PBP `idp_ff` as *"fumble forced against this player."*

### Consequences of the amendment

- **The ESPN PBP supplement is no longer needed.** It was scoped for the FF rule alone; that rule
  is now sourceable from Sleeper. ESPN drops from "planned enrichment" to "available independent
  oracle if we ever want one" — a want, not a need. This removes the last argument for a second
  backend.
- **Player-ID mapping gets easier on the PBP path.** `play_stats[].player` embeds the full player
  object (id, name, team, position), so PBP needs no join against the 14.6 MB player dump and no
  name matching at all. The Tier 4 hazards — suffix stripping, 83 full-name collisions,
  punctuation — still apply to the **config → Sleeper** boundary, which is unchanged, but not to
  anything downstream of it.
- **Two viable fetch shapes**, both uncached, both carrying `updated_at` (epoch ms) for a real
  as-of timestamp:
  - `stats_for_players_in_week(player_ids: [...])` — 1.9 KB, ~206 ms for 3 players. The natural
    primary poll for a ~20-starter lineup. Aggregates only; sufficient for every rule except FF.
  - `plays(sport, season, season_type, week)` trimmed to ids + stats — 574 KB, ~1.2 s. Same size
    as the REST weekly dump but play-level. Needed for FF, safeties, and def/return TD distance.
  - ⚠️ `plays`' `game_id` argument is **silently ignored** — you always get the whole week
    (2,966 plays / 16 games for 2025 wk 1). Filter client-side on the returned `game_id`.
- **"A missing stat key means zero" still holds**, and so does "scores are allowed to decrease."
  Neither depended on the freshness or FF findings.

### Risks this amendment introduces

- **GraphQL is *more* undocumented than the REST stats endpoint.** It is the mobile app's private
  API; its 240 root fields span betting and social features, which suggests active churn. The
  ADR 0002 provider interface remains the mitigation, and the REST weekly dump is a working
  fallback at 30 s current-week TTL if GraphQL closes or starts requiring auth.
- **Rate limiting is unprobed.** Do not poll a 574 KB uncached call aggressively without testing
  tolerance — there is no edge cache absorbing that load, which is precisely why it is fresh.
- **All PBP validation used completed 2025 games.** Live behavior is unmeasured: whether plays
  land promptly mid-game, how `updated_at` moves, and whether the 30 s current-week TTL holds once
  the week actually has data (it was read while 2026 preseason week 1 was still empty).

### This is now testable three weeks early

`/v1/state/nfl` reports the current season as **2026 preseason, week 1**, with games on
2026-08-14/15/16. The live-Sunday protocol in
[docs/probe-espn-sleeper.md](../probe-espn-sleeper.md) Tier 3 can run against the preseason slate
instead of waiting for September.

## Rationale

- **Shipping beats optimizing.** A scoreboard that is right on 4.5 of 5 hard rules and honest
  about the fifth is worth more than an unbuilt one that is right on all five.
- **Sleeper's aggregate coverage was the surprise of the probe.** The expectation going in was
  that pre-bucketed bonus stats would sit at the wrong thresholds and be useless. They sit at
  exactly ours, separately for passer and receiver, which matches our clarified rule that a 40+
  TD pass pays both.
- **One call per week for the whole league** is a genuine simplicity advantage over ESPN's ~26
  calls per slate, independent of freshness.
- **Provisional display over silent error** keeps the tool's disagreement with the spreadsheet
  legible instead of mysterious.
- **Excluding safeties over guessing at them** follows the same principle: a visible gap is
  cheaper to live with than an invisible wrong answer.
- **Name-keyed config over ID-keyed config** preserves ADR 0002's central bet — that the vendor is
  the least certain and most reversible decision — at the one place where a shortcut would have
  quietly welded us to Sleeper.

## Consequences

> The first two bullets are **superseded 2026-08-12** — struck through below. The rest stand.

- ~~Our scoreboard will **overpay forced fumbles** relative to the league's official scoring until
  the ESPN supplement lands. The provisional flag makes this visible rather than silent.~~
- ~~Our scoreboard will **not award safeties at all**, and will miss the 1-point bonus on 40+ yard
  defensive and return TDs.~~
- **nflverse validation is weakened but not blocked.** Sleeper's `gsis_id` covers only ~31% of
  players, so validation joins fall back to name + team + position. That is acceptable because
  validation is an offline batch activity, not a live path — but it means we cannot cheaply
  auto-validate scoring correctness.
- Depending on an **undocumented endpoint** (Sleeper's stats surface is not part of its documented
  public API) means it can change without notice. The provider interface localizes the blast
  radius. On ADR 0002's documentation-quality criterion, Sleeper is no safer than ESPN unofficial;
  the probe recommends that criterion carry near-zero weight, and this ADR adopts that.
- Sleeper's feed mixes **team aggregate rows** (`TEAM_BUF`, `TEAM_LV`) in with player rows. The
  provider must filter them.

## Follow-ups

Updated 2026-08-12. Done:

- ~~**Investigate a fresher Sleeper surface.**~~ **Done** — GraphQL, uncached, with PBP.
- ~~Confirm whether Sleeper exposes a **40+ distance bucket for defensive / return TDs**.~~
  **Answered** — no bucket at 40+, but raw return yardage makes it computable.
- ~~Confirm **`idp_safe` solo-credit semantics**.~~ **Answered from PBP**, on 2 observations.
- ~~Build the **ESPN PBP verification harness** for the FF rule.~~ **Dropped** — not needed.

Open:

- **Measure live behavior on the 2026-08-14 preseason slate.** Do plays land promptly mid-game,
  how does `updated_at` move, and does the current-week `s-maxage=30` hold once the week has data?
  Protocol in [docs/probe-espn-sleeper.md](../probe-espn-sleeper.md) Tier 3.
- **Probe GraphQL rate limiting / auth tolerance** under a realistic polling cadence.
- **Verify `idp_safe` against a shared-credit safety** — the solo/assisted distinction rests on
  2 clean observations.
- Settle the **Troy Dye ruling** — own-team fumble recovery after an interception return, which
  Sleeper credits as `idp_fum_rec`. A rules question for the league, not a data question.
- Verify Sleeper's `*_td_40p` bucket boundary is **40+ inclusive** — or sidestep it by computing
  the threshold from raw PBP yardage, which is now an option.
