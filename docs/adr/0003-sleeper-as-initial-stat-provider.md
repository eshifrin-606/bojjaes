# ADR: Sleeper as the Initial Stat Provider

**Status:** Accepted

Supersedes decision #2 ("Data source: deferred") of
[ADR 0002](0002-live-scoreboard-backend.md). Everything else in ADR 0002 — the provider
interface, game-granular fetch, on-demand serving, hardcoded scoring, static lineup config,
local-first hosting — remains in force.

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

## Decision

**Use Sleeper as the sole stat provider for the MVP**, behind ADR 0002's provider interface.

We are explicitly choosing a working single-backend implementation over an optimal one. One
backend is simpler than two, and Sleeper wins on every axis the probe measured except forced
fumbles. The specific trade-offs we are accepting:

### Accepted inaccuracies

| # | Rule | Sleeper's behavior | Direction | Decision |
| --- | --- | --- | --- | --- |
| 1 | Forced fumble resulting in turnover (4) | `idp_ff` is a raw count | **Over**pays ~44% of FFs | Award it, but **display it as provisional** (see below) |
| 2 | 40+ yard bonus on defensive / return TDs (1) | No distance bucket known for these | **Under**pays 1 pt | Accept. Rare, low value |
| 3 | Safety, solo credit only (2) | `idp_safe` may include shared credit | Would **over**pay | **Exclude safeties from scoring entirely.** Prefer a known omission to a possible wrong award |

Rules 2 and 3 are rare and cheap; correctness of the common path matters more than completeness
of the tail. Rule 3 is a one-line flip if we later confirm `idp_safe` semantics.

### Forced fumbles are surfaced as provisional

The FF error is **systematic and one-directional** — always an overpay, never an underpay, on
plays that are memorable and argued about. Silently folding it into a total would make our
scoreboard consistently disagree with the league's spreadsheet in the same direction, which
erodes trust in the tool faster than random error of the same size.

So: when a scored starter has `idp_ff > 0`, the 4 points are awarded but **flagged in the served
output as provisional** rather than absorbed into the total without comment.

### Freshness risk is accepted, with one cheap investigation first

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
- **ESPN play-by-play stays on the shelf as a verification tool**, not a second provider. Later,
  we may use it to check the FF rule against a small number of fumble plays (~20 a week
  league-wide, far fewer for our starters). If it graduates from verification to live use, it is a
  narrow enrichment step behind the provider interface — not a second backend.

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

- Our scoreboard will **overpay forced fumbles** relative to the league's official scoring until
  the ESPN supplement lands. The provisional flag makes this visible rather than silent.
- Our scoreboard will **not award safeties at all**, and will miss the 1-point bonus on 40+ yard
  defensive and return TDs.
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

- **Investigate a fresher Sleeper surface** (per-league scores / matchup endpoints, GraphQL,
  cache-busting). Highest-value open item.
- **Re-measure freshness on the first live Sunday.** One `curl -I` against the in-progress week
  reads `s-maxage` / `age` / `cf-cache-status` and settles the gate. Protocol is in
  [docs/probe-espn-sleeper.md](../probe-espn-sleeper.md) Tier 3.
- Confirm whether Sleeper exposes a **40+ distance bucket for defensive / return TDs**; if it
  does, accepted inaccuracy #2 disappears for free.
- Confirm **`idp_safe` solo-credit semantics**; if solo-only, re-enable safeties.
- Settle the **Troy Dye ruling** — own-team fumble recovery after an interception return, which
  Sleeper credits as `idp_fum_rec`. A rules question for the league, not a data question.
- Build the **ESPN PBP verification harness** for the FF rule.
