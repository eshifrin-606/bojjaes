# Probe: ESPN unofficial vs. Sleeper

**Status:** Tiers 1, 2, 4 complete (2026-08-08). Tier 3 (freshness) **partial** — cache headers
measured 2026-08-09 (§3.0); live time-to-appear still blocked until games start.
Open question 1 (FF/FR turnover qualification) resolved 2026-08-09 — see "Resolved: FF/FR
turnover qualification" below.

**Tier 5 (Sleeper GraphQL) added 2026-08-12 — this is the most consequential section in the
document and it invalidates finding 2.12 and open questions 2, 3 and 5.** Sleeper has an
undocumented GraphQL API that is uncached and carries full play-by-play. See
[Tier 5](#tier-5--sleeper-graphql--the-fresh-surface-exists--complete-2026-08-12).

**Outcome: Sleeper chosen** on 2026-08-09, recorded in
[ADR 0003](adr/0003-sleeper-as-initial-stat-provider.md). This document is now the evidence
behind that ADR; the ADR is authoritative for what we build.

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
| Forced fumble **that results in turnover** (4) | FF is credited even when the offense recovers it. Raw FF counts overcount. **Confirmed 2026-08-09: Sleeper's `idp_ff` is a raw count.** |
| Fumble recovery **that results in turnover** (2) | Needs confirmation the feed's FR excludes own-team recoveries. **Resolved 2026-08-09: it does.** |
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
| 2.12 | Any play-level endpoint? | ❌ None found on the REST surface. **Superseded 2026-08-12 — see [Tier 5](#tier-5--sleeper-graphql--the-fresh-surface-exists--complete-2026-08-12). Sleeper's GraphQL API has a `plays` query with full PBP.** This row was scoped to `api.sleeper.app/v1/*` and drew a conclusion about Sleeper as a whole; that inference was wrong. |

**2.11 is the finding that reframes the decision.** The prediction was that Sleeper's buckets
would be 50+ and therefore useless for our 40+ TD bonus. Wrong: Sleeper carries 40+ buckets
*separately for passer and receiver* (`pass_td_40p` **and** `rec_td_40p`), which is precisely
the semantics the 2026-08-08 clarification confirmed — a 45-yard TD pass pays both.

### Remaining gap for Sleeper

`idp_ff` and `idp_fum_rec` appear to be **raw counts, not turnover-qualified.** Week 1 shows 20
players with `idp_ff` and 12 with `idp_fum_rec`, which reads like unqualified FF credit. Our
rules pay 4 only for an FF *that results in a turnover* and 2 for a recovery *that results in a
turnover*. This is the one place ESPN's PBP does something Sleeper's aggregates cannot.

### Resolved: FF/FR turnover qualification (2026-08-09)

Validated against nflverse play-by-play for the **full 2025 regular season**, restricted to
players whose name maps to exactly one Sleeper entry so roster collisions cannot contaminate the
comparison. Turnover qualification is taken from nflverse `fumble_lost` (for FF) and
`fumble_recovery_N_team != fumbled_1_team` (for FR).

**`idp_ff` is a raw count — NOT turnover-qualified. Confirmed.** Across 203 unambiguous
player-weeks carrying an FF event:

| Hypothesis | Agreement |
| --- | --- |
| `idp_ff` == all FF events | **201/203 (99%)** |
| `idp_ff` == turnover-qualified FF only | 104/203 (51%) |

Season totals: **366 forced fumbles, 204 of which produced a turnover.** Paying the 4-point rule
straight off `idp_ff` overpays **162 forced fumbles a season — 44% of them.** Not an edge case.

**`idp_fum_rec` IS effectively turnover-qualified — the suspicion above was wrong.** It counts
only recoveries by the non-fumbling team. Across 269 unambiguous player-weeks:

| Hypothesis | Agreement |
| --- | --- |
| `idp_fum_rec` == all recoveries | 112/269 (42%) |
| `idp_fum_rec` == turnover-qualified | 250/269 (93%) |
| **`idp_fum_rec` + `st_fum_rec` + `def_st_fum_rec` == turnover-qualified** | **268/269 (99.6%)** |

All 19 apparent misses were *undercounts*, not overcounts: special-teams recoveries that Sleeper
books to `st_fum_rec` / `def_st_fum_rec` rather than the IDP key. Summing the three keys
reproduces the turnover-qualified set almost exactly. The single residual (K. Kelly, wk 3) looks
like a player-ID edge case rather than a semantic one.

Own-team recoveries land in the non-IDP `fum_rec` key, which is why the IDP key is clean.

**Consequence: the gap is half the size this probe assumed.** The 2-point recovery rule is
implementable from Sleeper aggregates today via the three-key sum. Only the 4-point FF rule is
not — Sleeper cannot express it at any accuracy, because turnover qualification is a property of
the play, not of the player's stat line.

**Rules question surfaced, not a data question:** wk 14, Troy Dye (LAC) recovered his own team's
fumble *after an interception return* and Sleeper credited `idp_fum_rec=1`. nflverse's
`fumble_lost` says that play was not a fumble-turnover — the turnover was the INT. Whether HMFFL
pays the 2 there needs a ruling; low frequency. Settle alongside open question 2 (`idp_safe`).

### Documentation quality

Confirmed as predicted: Sleeper's documented public API covers leagues/drafts/players, **not**
stats. The stats endpoint is undocumented. On ADR 0002's documentation criterion, Sleeper is
**not** meaningfully safer than ESPN unofficial — both are unofficial surfaces. This criterion
should carry little weight in the decision.

---

## Tier 3 — Freshness — PARTIAL (cache headers measured 2026-08-09; live lag still blocked)

Time-to-appear cannot be measured in the offseason. But **HTTP cache headers can**, and they
turned out to carry most of the signal.

### 3.0 Cache-header evidence — measured 2026-08-09 (offseason)

| Endpoint | `cache-control` | Observed |
| --- | --- | --- |
| Sleeper `/v1/stats/nfl/regular/2025/{wk}` | `public, s-maxage=3600, stale-while-revalidate=300, stale-if-error=600` | `cf-cache-status: HIT`, `age: 1739` |
| Sleeper `/projections/nfl/2025/1` | `s-maxage=3600, swr=600` | HIT, `age: 3369` |
| Sleeper `/v1/state/nfl` | `s-maxage=60, swr=180` | HIT, `age: 51` |
| ESPN `site.api…/nfl/scoreboard` | **`max-age=8`** | — |
| ESPN `site.api…/nfl/summary?event=` | **`max-age=1`** | — |
| ESPN `sports.core.api…/plays?limit=400` | `max-age=900, swr=7200` (Varnish) | — |

**This puts the structural "Sleeper is fresher" inference below in doubt.** Sleeper's stats
endpoint sits behind Cloudflare with a **one-hour edge TTL**. If that policy holds during live
games, the aggregate feed can be up to 60 minutes stale — 12× the ~5-minute budget in
[ADR 0002](adr/0002-live-scoreboard-backend.md) — regardless of how often we poll. ESPN's
`max-age=1` / `max-age=8` are the values of a genuinely live surface.

Not yet proven, for two reasons: the 3600 was observed on *historical* weeks in the offseason,
and Sleeper demonstrably tunes TTL per endpoint (`state/nfl` gets 60s), so the origin may emit a
much shorter `s-maxage` for the in-progress week. Likewise ESPN's `max-age=900` on core plays is
probably a *completed-game* policy and should be re-measured live.

**This is now the cheapest, highest-value Tier 3 test:** a single `curl -I` against the current
week during a live game answers it. No polling loop required.

### 3.1 Correction to the protocol below

The protocol originally claimed Sleeper offers no timestamp, so lag could only be measured by
diffing successive polls. Not quite — the `age` header plus a weak `ETag` gives cache-level
staleness for free. It does not reveal upstream stat lag, but it **decomposes** the measurement:
`age` = edge staleness, poll-diffing = the remainder (Sleeper's own aggregation lag).

### 3.2 Upstream context

Sleeper sources NFL data from **Sportradar** (stated in the community catalogue of Sleeper's
undocumented endpoints, and corroborated by the 99.5% `sportradar_id` coverage found in Tier 4).
Sportradar's own NFL API documents a **2-second TTL once a game goes `inprogress`**. So the data
upstream of Sleeper is fast; any observed lag is Sleeper's aggregation layer plus its CDN, not
the underlying source. That is mildly encouraging for the origin emitting a short TTL in-season.

Third-party "guides" to both APIs (sportsfirst.net, zuplo, sportsapis.dev) assert "real-time
updates" and "poll every 5–10 minutes" with no evidence and read as generated content — **give
them zero weight.** The one human anecdote located is a complaint on Sleeper's own forum that
"scoring and stats updates take much longer than other platforms"; single unverified data point,
but directionally consistent with the 3600 TTL.

### Protocol for the first live Sunday

- **First, and cheapest:** `curl -I` the current-week Sleeper stats endpoint mid-game and read
  `s-maxage` / `age` / `cf-cache-status`. Re-measure ESPN core `plays` on an in-progress game.
- Poll both sources every 60s during a live window; log `fetch_time`, payload, and any upstream
  timestamp / ETag / cache header.
- Measure **time-to-appear** for scoring plays against a known wall-clock reference. ESPN plays
  carry a `wallclock` field (e.g. `2025-09-05T00:30:57Z`) and a `modified` timestamp — these
  give a free, precise lag measurement without an external reference. For Sleeper, subtract the
  `age` header to separate edge staleness from aggregation lag (see 3.1).
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

Note the call-count asymmetry: **Sleeper serves the entire league's weekly stats in one
request.** ESPN requires ~13 game calls (plus ~13 more for PBP) to cover a Sunday slate. This is
a real *complexity* advantage for Sleeper — but per 3.0 it can no longer be assumed to be a
*freshness* advantage. Fewer round trips does not beat a one-hour edge cache.

### Findings — Tier 3

Cache-header findings recorded in 3.0. Live time-to-appear measurement still blocked — rerun
when the season starts.

---

## Tier 5 — Sleeper GraphQL — the fresh surface exists — COMPLETE (2026-08-12)

Run against ADR 0003's top follow-up: *"investigate whether a fresher Sleeper surface exists."*
It does, and it is better than the follow-up anticipated — the win is not just freshness, it is
**play-level data with native Sleeper player IDs.**

Endpoint: `POST https://api.sleeper.app/graphql`, JSON body `{"query": "..."}`. **No
authentication required** for any query used here. Introspection is enabled, but the schema uses
snake_case meta-fields (`query_type`, `of_type`, not `queryType`/`ofType`). 240 root query fields;
most are social/betting/league features irrelevant to us.

### 5.1 The freshness answer — two independent findings

**(a) The REST stats TTL is a function of week recency, not a fixed policy.** ADR 0003's accepted
risk was built on a 3600 s TTL measured against *historical* weeks. Re-measured 2026-08-12, with
`/v1/state/nfl` reporting the current week as `pre` / 2026 / week 1:

| Endpoint | `s-maxage` |
| --- | --- |
| `/v1/stats/nfl/pre/2026/1` — **current week** | **30** |
| `/v1/stats/nfl/pre/2026/2`, `/pre/2026/3`, `/regular/2026/{1,9}` — future weeks | 600 |
| `/v1/stats/nfl/regular/{2024,2025}/*`, `/pre/2025/*`, `/post/2025/1` — completed weeks | 3600 |

The one-hour TTL is a **completed-week** policy. The in-progress week is served at 30 seconds —
**inside ADR 0002's ~5-minute budget by 10×, not 12× over it.** Caveat: measured while the current
week still had no data (2026 preseason week 1 games kick off 2026-08-14), so this is a
current-week reading, not a live-game reading. But emptiness is not the driver — future weeks are
equally empty and get 600.

**(b) Cache-busting works, and GraphQL is not cached at all.** A random query param on the REST
stats endpoint returns `cf-cache-status: MISS`, i.e. it reaches origin — so even a long edge TTL
was always bypassable. And every GraphQL response carries
`cache-control: max-age=0, private, must-revalidate` with `cf-cache-status: DYNAMIC`. **There is
no edge cache in front of GraphQL.** The freshness gate is answered three separate ways.

`Stat` and `Play` both expose an `updated_at` field (epoch ms), so served output can carry a real
data timestamp rather than a fetch timestamp.

### 5.2 The `plays` query — Sleeper has play-by-play

```
plays(sport, season, season_type, week, game_id, date) -> Play
Play { play_id sequence time date week season season_type sport game_id provider
       updated_at metadata:Json play_stats:PlayStat }
PlayStat { player_id player:Map stats:Map stats_agg:Map }
```

- **`game_id` is silently ignored.** Passing it still returns the whole week — 2,966 plays across
  all 16 games for 2025 wk 1. Filter client-side on the returned `game_id`.
- `metadata` carries `description` (full NFL gamebook text), `play_type`, `possession`, `team`,
  `opponent`, `yards_gained`, `is_scoring_play`, `quarter_name`, clock, down/distance, penalties.
- `play_stats[].stats` is the **per-play stat delta**, keyed by the same stat vocabulary as the
  weekly aggregate, attributed to individual players.
- `play_stats[].player` embeds the **full player object** — `player_id`, first/last name, team,
  position. No join against the 14.6 MB player dump, and **no cross-source name matching at all.**

Cost (2025 wk 1, whole week, uncached):

| Query shape | Size | Latency |
| --- | --- | --- |
| `plays` with `metadata` + embedded `player` | 4.5 MB | 6.1 s |
| `plays` with `player_id` + `stats` only | **574 KB** | **1.2 s** |
| `stats_for_players_in_week` (3 explicit player_ids) | **1.9 KB** | **206 ms** |
| `weekly_stats` | — | requires non-null `order_by` |

The trimmed `plays` call is the same size as the REST weekly dump (570 KB) while being
play-level *and* uncached. `stats_for_players_in_week(player_ids: [...])` is the natural primary
poll for a ~20-starter lineup: 1.9 KB, 200 ms, uncached, with `updated_at`.

### 5.3 ⚠️ `idp_ff` means opposite things in the two feeds

**The single most dangerous finding in this probe.** In `play_stats`, `idp_ff` is attached to the
**player who fumbled**, not the defender who forced it. Verified on 2025 wk 1, BAL@BUF:

| Feed | Derrick Henry (fumbler, RB BAL) | Ed Oliver (forcer, DT BUF) |
| --- | --- | --- |
| REST weekly aggregate | `idp_ff` absent | **`idp_ff` = 1.0** |
| GraphQL `play_stats` | **`idp_ff` = 1.0** | `idp_ff` absent (only tackle keys) |

Same key, inverted subject. Anything that reuses aggregate-derived parsing logic against the PBP
feed will credit forced fumbles to the offense. In PBP, read `idp_ff` as *"fumble forced against
this player."*

### 5.4 The 4-point FF turnover rule is solvable from Sleeper alone

This was ADR 0003's one unfixable gap and the sole reason to keep an ESPN supplement. It closes.

**Turnover qualification:** on an FF play, the fumbler's row carries `fum_lost: 1.0` exactly when
the fumble was lost to the defense. Across all 20 FF plays in 2025 wk 1, **13 were
turnover-qualified and all 20 classifications match the gamebook description text** — own-team
recoveries (J. Hill, K. Gainwell, R. Wilson, D. Maye) and out-of-bounds fumbles (D. Kincaid,
T. Conklin) correctly read as non-turnovers. 13/20 is consistent with the season-wide 204/366.

**Attributing the FF to the forcer** is the only wrinkle, since PBP does not credit them a stat.
The forcer's name appears in `metadata.description` as `"forced by E.Oliver"`, and — critically —
the forcer is almost always already present among that same play's `play_stats` rows under some
other key (a tackle, a sack). Matching the parsed `F.Lastname` against only *that play's* rows
resolved **20/20** to a native Sleeper `player_id` (19/20 on first pass; the miss was the parser
retaining the "II" suffix in "K.Moore II", not an ambiguity). This is a bounded match against
2–6 candidate rows, not a fuzzy roster-wide join — a categorically easier problem than the
ESPN↔Sleeper name matching Tier 4 warned about.

Volume is ~20 FF plays per week league-wide.

### 5.5 Open questions 2 and 5 also close

**Safeties (open question 2) — solo credit is decidable per play.** `idp_safe` appears on exactly
one player's row, alongside `idp_tkl_solo` rather than `idp_tkl_ast`:

- wk 5: `D.Barnes` — `idp_safe: 1, idp_sack: 1, idp_tkl_solo: 1`
- wk 14: `J.Hines-Allen` — `idp_safe: 1, idp_sack: 1, idp_tkl_solo: 1`

Whatever the *aggregate* `idp_safe` means, PBP shows solo/assisted per play, so the rule is
implementable. Only 2 instances observed (safeties are rare); confirm on a shared-credit safety
before fully trusting it.

**40+ yard bonus on defensive / return TDs (open question 5) — return distance is available.**
Not via `metadata.yards_gained`, which reflects the offensive play (0, −2 on these plays), but via
per-player return-yardage keys on the scorer's row: `idp_int_ret_yd` (99, 63 observed) and
`idp_fum_ret_yd` (86). Sleeper's own bonus buckets for these are 50+ only
(`bonus_def_int_td_50p`, `bonus_def_fum_td_50p`) — the wrong threshold, exactly as originally
predicted — but the raw return yardage lets us compute our 40+ threshold ourselves.

### 5.6 Other useful GraphQL surfaces

- `scores(sport, season, season_type, week)` — live game state: `status`
  (`pre_game`/`complete`), and `metadata` with `is_in_progress`, `quarter`, per-quarter scores,
  `is_over`. Useful for knowing which games to poll.
- `/schedule/nfl/{season_type}/{season}` (REST) — 48 preseason games with `date`, `game_id`,
  `status`.
- `stats_for_players_in_week`, `game_stats`, `season_stats`, `get_player_stats` all return `Stat`.

### 5.7 Caveats

- GraphQL is **more** undocumented than the REST stats endpoint — it is the mobile app's private
  API. It can change without notice, and its shape (240 root fields, betting/social features)
  suggests active churn. The ADR 0002 provider interface remains the mitigation.
- No auth was required for these queries, but many sibling fields are user-scoped. If Sleeper
  tightens access, the REST weekly dump remains a working fallback at 30 s current-week TTL.
- Rate limiting was not probed. Do not poll `plays` (574 KB uncached, no edge cache absorbing
  load) aggressively without testing tolerance first.
- **All PBP validation used completed 2025 games.** Live behavior of `plays` — whether plays
  appear promptly mid-game and how `updated_at` moves — is still unmeasured.

### 5.8 Live test window opens 2026-08-14

`/v1/state/nfl` reports the 2026 **preseason** as current. The 2026 preseason week 1 slate is
2026-08-14/15/16. This is a live measurement window **three weeks before the regular season**, and
Tier 3's remaining protocol can run against it rather than waiting for September.

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
| **Sleeper alone** | **Leading candidate.** Covers 19/19 raw stats and — after the 2026-08-09 resolution — **4.5 of the 5 hard rules** from aggregates (the FR rule works via the three-key sum), one call per week for the whole league, half-sacks correct, full IDP coverage. Single remaining gap: **the 4-point FF turnover rule**, which it cannot express at all. |
| ESPN alone | **Viable but more work.** Requires the core PBP endpoint for 5 rules; ~26 calls per slate; typed participants are genuinely good and it is the *only* source that can settle FF-turnover. |
| Hybrid (Sleeper mapping + ESPN stats) | **Rejected as originally conceived.** `espn_id` at 37% makes the ID bridge unworkable; joining would fall back to fuzzy name+team+position matching, which is exactly the fragility the hybrid was meant to avoid. |
| Neither — escalate | Not indicated. |

**Chosen: Sleeper, sole provider for the MVP.** Settled 2026-08-09 in
[ADR 0003](adr/0003-sleeper-as-initial-stat-provider.md), which records the accepted
inaccuracies (FF overpay shown as provisional; safeties excluded; no 40+ bonus on defensive /
return TDs) and the deferred freshness gate.

Note that ADR 0003 does **not** treat the Tier 3 freshness gate as passed — it accepts the risk
in order to ship, and keeps "find a fresher Sleeper surface" as the top follow-up. The
gate-vs-tiebreaker framing below still stands; we have chosen to build against it rather than
wait for it.

**Amendment 2026-08-12 — Sleeper-alone is now unambiguously the right call.** Tier 5 removes every
reservation recorded below. Freshness: gate passed (uncached GraphQL; 30 s current-week REST TTL).
FF turnover rule: solvable from Sleeper PBP, so the ESPN supplement in option (c) is unnecessary.
ID mapping: PBP embeds native Sleeper player objects, so the Tier 4 name-matching hazards do not
apply to the PBP path at all. Both remaining ADR 0003 accepted inaccuracies (safeties, 40+
defensive/return TDs) also close. ESPN is no longer needed even as a verification tool for FF —
though it remains available as an independent oracle if we want one.

**Freshness caveat added 2026-08-09 (superseded by the amendment above):** the leaning below assumes Sleeper clears the ~5-minute
budget. §3.0 shows that assumption is untested and now doubtful — Sleeper's stats endpoint
carries a one-hour edge TTL in the offseason, while ESPN's live surfaces carry 1–8 seconds. This
does not change the leaning yet, but it is a **gate**, not a tiebreaker: if the in-season TTL is
also long, Sleeper-alone is disqualified regardless of its rule coverage.

**Leaning:** Sleeper as the primary provider. Path (a) — validating semantics against nflverse —
was executed on 2026-08-09 and settled the FR half in Sleeper's favour. What remains for the FF
half is a choice between (b) accepting a known-wrong result on one rule, which the season numbers
now price at **162 overpaid forced fumbles a year**, and (c) a narrow ESPN PBP supplement used
*only* for fumble plays. Option (b) is materially worse than it looked when the gap was still
unmeasured; (c) stays tractable because the join volume is small — roughly 20 fumble plays a
week, matched by name+team+position.

### Open questions

1. ~~**Are Sleeper's `idp_ff` / `idp_fum_rec` turnover-qualified?**~~ **Resolved 2026-08-09** —
   `idp_ff` is raw (44% overpay); `idp_fum_rec` + `st_fum_rec` + `def_st_fum_rec` is
   turnover-qualified at 99.6%. See the resolution section above. Successor question: **how do we
   source the 4-point FF rule** — ESPN PBP supplement, or accept the error?
2. ~~**Does Sleeper's `idp_safe` reflect solo credit only?**~~ **Effectively resolved 2026-08-12
   (Tier 5.5)** — GraphQL PBP shows `idp_safe` alongside `idp_tkl_solo` vs `idp_tkl_ast` on the
   play, so solo credit is decidable regardless of aggregate semantics. Safeties can be
   re-enabled. Only 2 instances observed; confirm against a shared-credit safety. The Troy Dye
   recovery-after-INT ruling is still open and is a **league rules** question, not a data one.
3. ~~**Sleeper live-update behavior / in-progress-week `s-maxage`**~~ **Resolved 2026-08-12
   (Tier 5.1)** — the 3600 s TTL is a *completed-week* policy. The current week is served at
   `s-maxage=30`; cache-busting reaches origin; and GraphQL is uncached entirely
   (`max-age=0`, `DYNAMIC`). The gate is passed. Residual: live-game behavior of `plays` and
   `updated_at` is unmeasured — test on the 2026-08-14 preseason slate (Tier 5.8).
4. Do Sleeper's `*_td_40p` buckets use **40+ inclusive**? Our rule is "40+ yards" and was
   confirmed inclusive on 2026-08-09 (see [scoring.md](scoring.md)). Sleeper's bucket boundary is
   still unverified — verify a known 40-yard TD lands in the bucket. **Now lower-stakes:** PBP
   gives raw per-play yardage, so we can compute the threshold ourselves and skip the buckets.
5. ~~**Does Sleeper carry a 40+ distance bucket for defensive / return TDs?**~~ **Resolved
   2026-08-12 (Tier 5.5)** — no 40+ bucket (only `bonus_def_int_td_50p` / `bonus_def_fum_td_50p`,
   the wrong threshold), but `idp_int_ret_yd` / `idp_fum_ret_yd` on the scorer's PBP row give raw
   return distance, so the 40+ bonus is computable.
6. **New:** does the GraphQL API rate-limit or require auth under sustained polling? Unprobed.

Promote the outcome into ADR 0002 (or a successor) and journal working notes in Basic Memory
per CLAUDE.md.
