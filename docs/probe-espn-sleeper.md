# Probe: ESPN unofficial vs. Sleeper

**Status:** Tiers 1, 2, 4 complete (2026-08-08). Tier 3 (freshness) blocked until live games.

Settles the open data-source decision in
[ADR 0002](adr/0002-live-scoreboard-backend.md) for the two free candidates. Scoring rules
under probe are [docs/scoring.md](scoring.md), including the 2026-08-08 clarifications.

MySportsFeeds is out of scope for this round; nflverse remains the validation oracle and is not
probed here.

**Data probed:** 2025 regular season. Week 1 in depth (all 16 games listed; DAL@PHI, KC@LAC,
TB@ATL, DET@GB, BAL@BUF pulled in full), plus weeks 2, 5, 9, 14, 18 for sparse-key scanning.

## The question this probe exists to answer

Freshness is *not* the deciding factor. The deciding factor is whether a source can express
rules that are not computable from box-score aggregates:

| Rule | Why aggregates are insufficient |
| --- | --- |
| Bonus: TD play of 40+ yds | Per-player "long" doesn't say whether the long *was* a TD, and can't count a second 40+ TD. Compounded by the clarification that the bonus pays *both* QB and receiver. |
| Bonus: FG 50+ | Needs per-kick distances. "FG 2/2, long 53" can't distinguish one 50+ make from two. |
| Forced fumble **that results in turnover** (4) | FF is credited even when the offense recovers it. Raw FF counts overcount. |
| Fumble recovery **that results in turnover** (2) | Needs confirmation the feed's FR excludes own-team recoveries. |
| Safety, **solo credit only** | Needs to know whether credit was shared. |

**Headline result:** the premise was half wrong. Sleeper pre-computes distance-bucketed bonus
stats at *exactly* our thresholds, so it clears four of these five from aggregates alone with no
play-by-play. ESPN needs play-by-play for all of them — and has a good one. The real
discriminator turned out to be **player-ID mapping**, where the expected answer also inverted.

---

## Tier 1 — Raw-stat coverage — COMPLETE

Sleeper keys verified present with correct granularity. ESPN column refers to the **box score**
(`summary?event=` → `boxscore.players`); items needing ESPN play-by-play are marked PBP.

| # | Stat needed | ESPN box score | Sleeper key |
| --- | --- | --- | --- |
| 1.1 | Passing yards | `passing` YDS | `pass_yd` |
| 1.2 | Passing TD | `passing` TD | `pass_td` |
| 1.3 | Interception thrown | `passing` INT | `pass_int` |
| 1.4 | 2pt pass | **absent → PBP** | `pass_2pt` |
| 1.5 | Rushing yards | `rushing` YDS | `rush_yd` |
| 1.6 | Rushing TD | `rushing` TD | `rush_td` |
| 1.7 | Receiving yards | `receiving` YDS | `rec_yd` |
| 1.8 | Receiving TD | `receiving` TD | `rec_td` |
| 1.9 | Return TD (KR/PR) | `kickReturns`/`puntReturns` TD | `kr`/`pr` family, `blk_pr_td` |
| 1.10 | Defensive / fumble-return TD | `defensive` TD, `interceptions` TD | `idp_def_td`, `def_td` (team), `pass_int_td` |
| 1.11 | Fumbles **lost** | `fumbles` LOST ✅ distinct from FUM | `fum_lost` ✅ distinct from `fum` |
| 1.12 | 2pt scored (rush/rec) | **absent → PBP** | `rush_2pt`, `rec_2pt` |
| 1.13 | XP made | `kicking` XP ("2/2") | `xpm` |
| 1.14 | FG made (count) | `kicking` FG ("2/2") | `fgm` |
| 1.15 | IDP sack, **0.5 granularity** | ✅ `defensive` SACKS carries `0.5`/`1.5` | ✅ `idp_sack` values {0.5,1.0,1.5,2.0,2.5} |
| 1.16 | IDP interception | `interceptions` INT | `idp_int` |
| 1.17 | IDP forced fumble | **absent from box score → PBP** | `idp_ff` |
| 1.18 | IDP fumble recovery | `fumbles` REC | `idp_fum_rec` |
| 1.19 | Safety credited to a player | **absent → PBP** | ✅ `idp_safe` (and team `safe`) |

Notes on the gotchas we flagged:

- **1.15 half-sacks — both sources pass.** Verified against DET@GB: ESPN shows Rashan Gary
  `1.5`, Lukas Van Ness `0.5`; Sleeper shows `idp_sack` 1.5 / 0.5 for the same players. Also
  matched Billy Bowman 0.5 and Micah Parsons 1.0. **Zero disagreement on every value spot-checked.**
- **1.11 lost vs. total fumbles — both sources distinguish them.**
- **1.19 safeties — Sleeper has them after all.** `idp_safe` and `safe` are *sparse keys*: absent
  in weeks 1, 2, 9; present in weeks 5, 14, 18. This sparse-key behavior is general across the
  Sleeper feed and is an important implementation note — **a missing key means zero, not an
  error.** Any parser must treat absent keys as 0 rather than failing.
- Sleeper's stats feed includes **team aggregate rows** keyed `TEAM_BUF`, `TEAM_LV`, etc.
  mixed in with player rows. These must be filtered out.

**Verdict Tier 1: Sleeper 19/19 from aggregates. ESPN box score ~14/19, four rules requiring PBP.**

---

## Tier 2 — Play-level detail — COMPLETE

### ESPN

| # | Check | Result |
| --- | --- | --- |
| 2.1 | `summary?event={id}` carries boxscore **and** plays | ✅ Both. `boxscore.players` + `drives.previous[].plays` (169 plays for DAL@PHI) + a separate `scoringPlays` array (8). |
| 2.2 | `sports.core.api.espn.com/v2/.../plays` shape | ✅ 171 items, `pageCount: 1` at `limit=400` — **a whole game in one request, no pagination needed.** |
| 2.3 | Structured participant athlete IDs, or text only? | **Split.** The `summary` endpoint has **no** athlete participants — only `teamParticipants`. The **core** endpoint has full `participants[]` with athlete refs on 140/171 plays. |
| 2.4 | Scoring-play yardage as a number | ✅ `statYardage` on every play (169/169). |
| 2.5 | 40+ TD attributable to both passer and receiver? | ✅ Participants carry **typed roles**: `passer`, `receiver`, `rusher`, `scorer`, `tackler`, `assistedBy`, `sackedBy`, `fumbler`, `forcedBy`, `recoverer`, `returner`, `kicker`, `patScorer`, `passDefender`. Roles map almost 1:1 onto our rules. |
| 2.6 | Individual FG distances | ✅ Via PBP `statYardage` (41, 53, 58 yd FGs resolved individually) — and **confirmed impossible from the box score**, which shows only `FG 2/2, LONG 53` for Aubrey's 41+53. |
| 2.7 | FF correlated to recovering team | ✅ `forcedBy` + `recoverer` participants on the same play, plus a play-level `isTurnover` boolean and play type `Fumble Recovery (Opponent)`. |

Important ergonomics detail on 2.3: participants are `$ref` URLs, **but the athlete ID is
embedded in the URL path** (`.../athletes/4361579?lang=en`). It can be regexed out without
issuing a second HTTP request. The `$ref` chasing that usually makes ESPN's core API painful is
avoidable here.

One quirk: some `$ref` values come back pointing at the internal host
`sports.core.api.espn.pvt` rather than `.com`. Harmless if we only parse IDs out of them —
**dangerous if we ever try to dereference them.** Don't.

### Sleeper

| # | Check | Result |
| --- | --- | --- |
| 2.8 | Stats endpoint exists | ✅ `api.sleeper.app/v1/stats/nfl/regular/2025/{week}` → HTTP 200, ~570 KB, **1.8 s**, ~2,100–2,500 player rows/week. |
| 2.9 | IDP for all defenders or only rostered? | ✅ **All.** Week 1 has `idp_tkl` for 501 players, `idp_sack` for 64, `idp_pass_def` for 111 — full league coverage, not a rostered subset. |
| 2.10 | Pre-bucketed bonus fields? | ✅ Extensive. 228 distinct stat keys. |
| 2.11 | **Do the buckets match our thresholds?** | ✅ **Yes — better than expected.** `pass_td_40p`, `rec_td_40p`, `rush_td_40p` exist at exactly our 40+ threshold (50p variants exist too). `fgm_50p`, `fgm_50_59`, `fgm_60p` cover the FG bonus. |
| 2.12 | Any play-level endpoint? | ❌ None found. Confirms the expectation. |

**2.11 is the finding that reframes the decision.** The prediction was that Sleeper's buckets
would be 50+ and therefore useless for our 40+ TD bonus. Wrong: Sleeper carries 40+ buckets
*separately for passer and receiver* (`pass_td_40p` **and** `rec_td_40p`), which is precisely
the semantics the 2026-08-08 clarification confirmed — a 45-yard TD pass pays both.

### Remaining gap for Sleeper

`idp_ff` and `idp_fum_rec` appear to be **raw counts, not turnover-qualified.** Week 1 shows 20
players with `idp_ff` and 12 with `idp_fum_rec`, which reads like unqualified FF credit. Our
rules pay 4 only for an FF *that results in a turnover* and 2 for a recovery *that results in a
turnover*. **Unverified — flagged as the top open question.** This is the one place ESPN's PBP
does something Sleeper's aggregates cannot.

### Documentation quality

Confirmed as predicted: Sleeper's documented public API covers leagues/drafts/players, **not**
stats. The stats endpoint is undocumented. On ADR 0002's documentation criterion, Sleeper is
**not** meaningfully safer than ESPN unofficial — both are unofficial surfaces. This criterion
should carry little weight in the decision.

---

## Tier 3 — Freshness — BLOCKED until live games

Cannot run on 2026-08-08 (offseason). Protocol for the first live Sunday:

- Poll both sources every 60s during a live window; log `fetch_time`, payload, and any upstream
  timestamp / ETag / cache header.
- Measure **time-to-appear** for scoring plays against a known wall-clock reference. ESPN plays
  carry a `wallclock` field (e.g. `2025-09-05T00:30:57Z`) and a `modified` timestamp — these
  give a free, precise lag measurement without an external reference. **Sleeper has no
  equivalent timestamp**, so its lag must be measured by diffing successive polls.
- Measure **PBP lag separately from box-score lag**.
- Watch for **stat corrections** (sack reassigned, fumble ruling reversed). Confirm sources
  actually correct rather than freeze.
- Record payload size and p50/p95 latency for a full slate.

Known baseline costs from this probe (offseason, cached):

| Call | Size | Time |
| --- | --- | --- |
| Sleeper weekly stats (entire league, one call) | 570 KB | 1.8 s |
| Sleeper player dump | 14.6 MB | 0.6 s |
| ESPN summary (one game) | 424 KB | — |
| ESPN core plays (one game, unpaginated) | — | — |

Note the asymmetry: **Sleeper serves the entire league's weekly stats in one request.** ESPN
requires ~13 game calls (plus ~13 more for PBP) to cover a Sunday slate. That is a real
freshness and complexity advantage for Sleeper that Tier 3 should quantify.

### Findings — Tier 3

_(blocked — rerun when the season starts)_

---

## Tier 4 — Player-ID mapping — COMPLETE

The Sleeper player dump is 12,217 entries / 8,616 Active.

### 4.6 Cross-source ID coverage — **the expected mapping bridge does not exist**

Coverage among the 8,616 Active players:

| Field | Coverage |
| --- | --- |
| `sportradar_id` | **99.5%** |
| `rotowire_id` | 86.5% |
| `fantasy_data_id` | 78.5% |
| `swish_id` | 57.4% |
| `oddsjam_id` | 47.8% |
| `yahoo_id` | 46.7% |
| **`espn_id`** | **45.3%** |
| `gsis_id` | 31.4% |
| `pandascore_id` | 0% |

Restricting to the population that actually matters — the **458 players with scoreable Week 1
production** — coverage gets *worse*:

- **`espn_id`: 37.1%** (170/458)
- `gsis_id`: 30.8% (141/458)

And the missing ones are not fringe players. Productive Week 1 players with **no `espn_id`**
include: **Brandon Aubrey, James Cook, Kyle Pitts, Khalil Shakir, Bucky Irving, Omarion
Hampton, Colston Loveland, Jahan Dotson, Kayshon Boutte, Noah Gray.**

**Consequence:** ADR 0002 anticipated Sleeper as a mapping layer via its cross-ID dictionary.
For an ESPN bridge specifically, **that plan is not viable** — `espn_id` is missing for roughly
two-thirds of relevant players, including likely starters. `sportradar_id` at 99.5% is the only
credible bridge, but ESPN does not expose Sportradar IDs, so it doesn't help this pairing.

Also relevant: `gsis_id` at ~31% weakens the nflverse validation-oracle plan, since nflverse is
keyed on GSIS. Validation will need name-based joins too.

### 4.1–4.4 Name-matching hazards

- **Sleeper strips name suffixes.** Only 6 entries in 12,217 carry a suffix. Sleeper stores
  `Marvin Harrison`, `Brian Thomas`, `Michael Penix`, `Travis Etienne` — no "Jr."
- **ESPN keeps suffixes.** ESPN box scores show `Kenneth Murray Jr.`, `Billy Bowman Jr.`
- ⇒ **Any ESPN↔Sleeper or config↔Sleeper name join must normalize suffixes.** A lineup file
  written as "Marvin Harrison Jr." will not match Sleeper.
- **83 exact `full_name` collisions** among active roster-eligible players. Suffix stripping
  actively *causes* some of them — e.g. two `Kenneth Walker` (a WR with `espn_id` 2971595, and
  the KC RB with **no** `espn_id`). Name-only matching is unsafe; joins need **name + position +
  team**.
- **445 names** contain an apostrophe, hyphen, or period (`Le'Veon Bell`, `Henry To'oTo'o`,
  `Mo Alie-Cox`, `T.J. Vasher`). Punctuation normalization required on both sides.

---

## Decision

ADR 0002 §2 criteria, priority order: live freshness → ID-mapping ergonomics → raw-stat
coverage → documentation quality. This probe amends that: play-level coverage is a **gate**, and
documentation quality should be **dropped to near-zero weight** (both sources are unofficial).

| Option | Verdict |
| --- | --- |
| **Sleeper alone** | **Leading candidate.** Covers 19/19 raw stats and 4 of the 5 hard rules from aggregates, one call per week for the whole league, half-sacks correct, full IDP coverage. Single open gap: FF/FR turnover qualification. |
| ESPN alone | **Viable but more work.** Requires the core PBP endpoint for 5 rules; ~26 calls per slate; typed participants are genuinely good and it is the *only* source that can settle FF-turnover. |
| Hybrid (Sleeper mapping + ESPN stats) | **Rejected as originally conceived.** `espn_id` at 37% makes the ID bridge unworkable; joining would fall back to fuzzy name+team+position matching, which is exactly the fragility the hybrid was meant to avoid. |
| Neither — escalate | Not indicated. |

**Chosen:** _(pending — see open questions below and Tier 3)_

**Leaning:** Sleeper as the primary provider, with the FF/FR turnover question resolved either
by (a) validating `idp_ff` semantics against nflverse, or (b) accepting a known-wrong edge case
on two low-frequency rules, or (c) a narrow ESPN PBP supplement used *only* for fumble plays,
where the join volume is small enough that name matching is tractable.

### Open questions

1. **Are Sleeper's `idp_ff` / `idp_fum_rec` turnover-qualified?** Top blocker. Validate against
   nflverse PBP for a known week.
2. **Does Sleeper's `idp_safe` reflect solo credit only?** Low frequency, low stakes.
3. **Sleeper live-update behavior** — the whole Tier 3 question. Does the weekly stats endpoint
   update mid-game, and at what lag?
4. Do Sleeper's `*_td_40p` buckets use **40+ inclusive**? Our rule is "40+ yards". Verify a
   known 40-yard TD lands in the bucket.

Promote the outcome into ADR 0002 (or a successor) and journal working notes in Basic Memory
per CLAUDE.md.
