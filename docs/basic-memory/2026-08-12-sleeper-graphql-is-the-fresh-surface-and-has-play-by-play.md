---
title: 2026-08-12-sleeper-graphql-is-the-fresh-surface-and-has-play-by-play
type: note
permalink: bojjaes-memory/basic-memory/2026-08-12-sleeper-graphql-is-the-fresh-surface-and-has-play-by-play
tags:
- nfl-data
- sleeper
- graphql
- provider
- freshness
- play-by-play
- adr-0003
- forced-fumble
- reversal
---

Ran ADR 0003's top follow-up — *"investigate whether a fresher Sleeper surface exists"* — on
2026-08-12. It exists, and the answer is bigger than the question. Full evidence in
`docs/probe-espn-sleeper.md` Tier 5. This note records the reversals and the reasoning.

`POST https://api.sleeper.app/graphql`, no auth, introspection enabled (snake_case meta-fields:
`query_type`, `of_type`). 240 root query fields.

## Observations

- [finding] GraphQL responses are **uncached**: `cache-control: max-age=0, private,
  must-revalidate`, `cf-cache-status: DYNAMIC`. No Cloudflare edge cache in front of it at all.
  #freshness #graphql
- [finding] The REST stats endpoint's TTL is **a function of week recency, not a fixed policy**.
  Current week `s-maxage=30`; future weeks 600; completed weeks 3600. The one-hour TTL that
  ADR 0003 accepted as risk is a *completed-week* policy. #freshness #reversal
- [finding] Cache-busting the REST endpoint with a random query param returns
  `cf-cache-status: MISS` — it reaches origin. The long TTL was always bypassable. #freshness
- [reversal] The freshness gate from [[2026-08-09-sleeper-stats-endpoint-has-1h-edge-ttl-espn-is-seconds]]
  is **passed, three independent ways**. That note's headline — "one-hour edge TTL, 12× over
  budget" — measured completed weeks and generalized to live ones. The generalization was wrong.
  #failed-assumption
- [finding] **Sleeper has play-by-play.** `plays(sport, season, season_type, week)` returns full
  PBP: gamebook `description`, `play_type`, possession, yardage, clock, plus `play_stats[]` —
  per-play stat deltas attributed to individual players. The probe's Tier 2.12 recorded "no
  play-level endpoint"; that finding was scoped to `/v1/*` REST and wrongly generalized to
  Sleeper as a whole. #reversal #play-by-play
- [gotcha] The `game_id` argument on `plays` is **silently ignored** — you always get the whole
  week (2,966 plays / 16 games for 2025 wk 1). Filter client-side on the returned `game_id`.
  #gotcha
- [finding] `play_stats[].player` embeds the **full player object** (id, name, team, position).
  No join against the 14.6 MB player dump, and **no cross-source name matching** on the PBP path
  — the Tier 4 hazards (suffix stripping, 83 name collisions, punctuation) simply don't arise.
  #id-mapping
- [gotcha] ⚠️ **`idp_ff` means opposite things in the two feeds.** In the REST weekly aggregate it
  credits the *forcer*; in GraphQL `play_stats` it sits on the *fumbler*. Verified on Derrick
  Henry / Ed Oliver, 2025 wk 1 BAL@BUF: aggregate gives Oliver `idp_ff=1` and Henry nothing, PBP
  gives Henry `idp_ff=1` and Oliver only tackle keys. Reusing aggregate parsing logic against PBP
  will credit forced fumbles to the offense. Read PBP `idp_ff` as "fumble forced *against* this
  player." #gotcha #forced-fumble
- [finding] **The 4-point FF turnover rule is solvable from Sleeper alone.** The fumbler's PBP row
  carries `fum_lost: 1.0` exactly when the fumble was lost. All 20 FF plays in 2025 wk 1 classify
  correctly against the description text (13 turnover-qualified) — own-team recoveries and
  out-of-bounds fumbles read as non-turnovers. This was the *only* rule ADR 0003 said Sleeper
  could not express at any accuracy. #forced-fumble #reversal
- [method] Attributing the FF to the forcer: PBP gives them no stat, but the name is in the
  description (`"forced by E.Oliver"`) **and the forcer is almost always already among that same
  play's `play_stats` rows** under a tackle or sack key. Matching the parsed name against only
  that play's 2–6 rows resolved 20/20 to a native Sleeper `player_id`. A bounded match, not a
  roster-wide fuzzy join. #method
- [finding] Safeties: `idp_safe` appears alongside `idp_tkl_solo` (not `idp_tkl_ast`) on the play,
  so solo credit is decidable from PBP regardless of what the aggregate key means. Only 2
  instances observed — safeties are rare. #open-question
- [finding] 40+ defensive/return TDs: Sleeper's own buckets are 50+ only
  (`bonus_def_int_td_50p`, `bonus_def_fum_td_50p`) — the wrong threshold, as originally predicted
  — but `idp_int_ret_yd` (99, 63 observed) and `idp_fum_ret_yd` (86) give raw return distance, so
  our 40+ threshold is computable.
- [finding] `Stat` and `Play` both expose `updated_at` (epoch ms) — a real data timestamp, not a
  fetch timestamp. Removes the need for the poll-diffing measurement in the Tier 3 protocol.
- [finding] Cost: trimmed `plays` (ids + stats, no metadata) is 574 KB / 1.2 s for a whole week —
  same size as the REST weekly dump but play-level *and* uncached. Full `plays` with metadata and
  embedded players is 4.5 MB / 6.1 s. `stats_for_players_in_week(player_ids: [...])` is 1.9 KB /
  206 ms for 3 players — the natural primary poll for a ~20-starter lineup.
- [finding] `/v1/state/nfl` reports the current season as 2026 **preseason**, week 1, with games
  2026-08-14/15/16. A live measurement window opens three weeks before the regular season; Tier 3's
  remaining protocol can run then instead of in September. #next-step
- [caveat] All PBP validation used **completed 2025 games**. Live behavior of `plays` — whether
  plays land promptly mid-game, how `updated_at` moves — is unmeasured. The 30 s current-week TTL
  was also read while the week still had no data. #open-question
- [caveat] GraphQL is *more* undocumented than the REST stats endpoint — it is the mobile app's
  private API, and its shape (betting, social, 240 root fields) suggests active churn. Rate
  limiting is unprobed; don't hammer a 574 KB uncached call without testing tolerance. The
  ADR 0002 provider interface stays the mitigation, and the REST dump is a working fallback.
- [method] Introspection was the whole game here. One `__schema{query_type{fields{...}}}` query
  surfaced `plays`, `scores`, and `stats_for_players_in_week` in a single round trip. The earlier
  probe concluded "no play-level endpoint" from enumerating documented REST paths. **When a
  vendor's app visibly does something the API seems unable to do, look for a different protocol,
  not a different path.** #method

## Consequence for the provider decision

Everything [[0003-sleeper-as-initial-stat-provider]] accepted as a cost is now recoverable:

| ADR 0003 accepted inaccuracy | Status |
| --- | --- |
| #1 FF overpay (~162/season), shown as provisional | **Fixable** — `fum_lost` on the PBP row |
| #2 No 40+ bonus on def/return TDs | **Fixable** — `idp_*_ret_yd` |
| #3 Safeties excluded entirely | **Fixable** — solo credit visible per play |
| Freshness risk (up to 1 h stale) | **Gone** — uncached GraphQL |

The ESPN PBP supplement is no longer needed for the FF rule, which removes the only remaining
argument for a second backend. ESPN stays useful purely as an independent verification oracle,
which is a want, not a need.

ADR 0003's core decision — Sleeper, sole provider, behind the ADR 0002 provider interface — is
**unchanged and strengthened**. What needs revising is the list of accepted costs and the
provisional-FF display design, which was built to make an error legible that we can now just not
make.

## Relations

- part_of [[2026-08-08-espn-vs-sleeper-stat-source-probe]]
- supersedes_assumption_in [[2026-08-09-sleeper-stats-endpoint-has-1h-edge-ttl-espn-is-seconds]]
- relates_to [[2026-08-09-sleeper-idp-ff-is-raw-idp-fum-rec-is-turnover-qualified]]
- revises_costs_accepted_by [[0003-sleeper-as-initial-stat-provider]]
- relates_to [[0002-live-scoreboard-backend]]

---

Informative, not authoritative. **Promoted 2026-08-12** into
[[0003-sleeper-as-initial-stat-provider]] as an in-place amendment section (not a superseding
ADR — see the project preference for growing one document rather than splitting out new ones).
That ADR is authoritative for what we build; this note is the working record of how we got there.
