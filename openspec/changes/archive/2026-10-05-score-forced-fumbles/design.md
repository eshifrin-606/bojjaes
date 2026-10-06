## Context

`sleeper.Client.WeekStats` fetches the REST weekly aggregate and transforms it into a
`score.WeekStats`. `statscache` wraps the client with a 5-minute TTL and single-flight. The page reads
through the cache. Forced fumbles are unscored because the aggregate's `idp_ff` is not
turnover-qualified (~44% of forced fumbles are not turnovers).

Investigation of 2026 weeks 2–3 settled the rule, the data source, and the attribution algorithm (see
proposal and specs). This design covers where the code lives and how fetching is staged so the page
waits on play-by-play only within a bound (see D4).

Observed play-by-play facts that shape the design:

- Sleeper's REST `GET https://api.sleeper.com/plays/nfl/recent?season_type=regular&season=S&week=W&limit=K`
  returns the K newest plays of the week, newest first. Each play has `play_id`, `game_id`,
  `sequence`, `updated_at`, `metadata.description`, and `play_stats[{player_id, stats, game_id,
  play_id}]`. Rows carry no name or team. Responses are CDN-cached for 300 s. `limit=5000` covers a
  whole week: 3.9 MB, 3–12 s.
- `idp_ff` is on the fumbler's row. The forcer often has a row with empty or `null` `stats`. Team
  rows have non-numeric IDs (`CAR`, `NO`).
- Names and teams come from the weekly aggregate, which the client already fetches. Each aggregate row
  has `player_id`, `player.first_name`/`last_name`, and a top-level `team` that is the player's team
  for that week's game. `player.team` is the current team and is not used. On live 2026 week 3, every
  numeric row on a lost forced-fumble play had an aggregate entry, and all 17 lost forced fumbles were
  attributed with the aggregate as the name source.
- Reviewed plays repeat the original text, then "...the play was overturned.", then the final play.
  Both directions occur in the recordings: a fumble overturned away, and a fumble that exists only
  after "overturned".

## Goals / Non-Goals

**Goals:**

- Pay 4 per turnover-qualified forced fumble, attributed per fumble, matching the official record for
  the 2026 week 2–3 ground truth.
- Keep `score` provider-neutral and I/O-free. Keep every Sleeper shape inside `internal/sleeper`.
- Keep the page's latency and failure modes those of the aggregate fetch alone.

**Non-Goals:**

- Safeties, which also need play-by-play and stay unscored.
- A 40+ bonus on defensive or return touchdowns. It is not a rule.
- Persisting plays to the Fly volume. This slice keeps plays in memory; see Decisions.
- Using play-by-play for any stat other than forced fumbles.
- Resolving the open rules questions below.

## Decisions

### D1. Domain: one count field and one flat term

`StatLine` gains `FFTurnover int` (JSON `ff_turnover`), and `Points` adds `4 * FFTurnover`. The count
is already turnover-qualified, so `score` stays a flat sum. That mirrors `FumRec`, which is also
qualified before it reaches the domain.

*Alternative:* carry raw forced fumbles plus a turnover flag. Rejected: the flag is per fumble, not
per line, so it cannot live on a stat line.

### D2. Attribution is a pure function over decoded plays

In `internal/sleeper`, `forcedFumbleTurnovers(plays []play, ids identities, logf) map[string]int`
maps player ID to count. `ids` maps player ID to identity (normalized abbreviated name and the week's
team), built from the weekly aggregate. It has no I/O and is fully tested against fixtures. For each
play it works in four steps:

1. Collect rows with `idp_ff > 0` and numeric player IDs. These are the fumblers. Skip a fumbler whose
   `fum_lost` is not positive; only turnovers are attributed.
2. Take the description text after the last case-insensitive "overturned". Regex out every
   `(\S+) FUMBLES, forced by ([^.]+\.[^.,;]+?)` pair. The exact regex is settled by tests against the
   fixture texts, including suffix names.
3. Pick the fumbler's pair by normalized abbreviated name: `first initial + "." + last name`, with
   suffixes stripped and compared case-insensitively. Then find forcer rows by the same normalized
   name among all numeric rows on the play whose aggregate team differs from the fumbler's. Names and
   teams are looked up in `ids` by `player_id`. The candidates are still the play's own rows; the
   aggregate only identifies them. A row absent from `ids` cannot be identified and never matches.
4. If exactly one pair and exactly one forcer row are found, add 1 to the forcer. Otherwise log the
   play ID and description and award nothing. A forcer absent from the aggregate therefore ends here:
   logged, not credited.

*Alternative:* parse the description alone and ignore `idp_ff`. Rejected: overturned and no-play
fumbles keep their text. `idp_ff` is the provider's signal that the fumble stands.

### D3. A play store per week, in memory for this slice

A `PlayStore`, inside `internal/sleeper`, holds `map[weekKey]*weekPlays`. Each `weekPlays` holds
`map[playID]play`, the newest `updated_at` held, the time of the last successful whole-week fetch, and
an in-flight flag for the whole-week fetch. A mutex guards it. A merge replaces a play only when the
incoming `updated_at` is newer.

The store is an in-memory map because the volume plan treats the store as a cache that is wiped and
re-seeded on deploy. In-memory has the same lifecycle: one whole-week fetch per week per process. The
store's API (merge, snapshot, last-whole-fetch) is written so a volume-backed implementation can
replace it later without touching attribution or the client.

*Alternative:* write to the Fly volume now. Deferred: it adds file format and fsync concerns for no
behavior the spec needs today.

### D4. Fetch staging: synchronous small poll, background whole-week fetch

`Client.WeekStats(ctx, season, week)` runs these steps:

1. Fetch the aggregate, as today. An error still returns an error. The same decode also builds the
   `identities` lookup from every row, including rows whose `stats` is `null`.
2. If `c.Plays` is non-nil, call `c.Plays.ForcedFumbles(ctx, season, week, ids)`:
   - Poll `recent` with `limit=300` under its own short timeout (~5 s). On success, merge the plays.
     On failure, log and continue with what is held.
   - If the store held nothing, or the poll's oldest play is newer than the newest held play, start a
     background whole-week fetch (`limit=5000`), single-flighted per week with a detached context and
     its own timeout (~30 s). It merges on success and logs on failure.
   - If the week is quiet (newest change > 1 h ago) and the last whole fetch predates newest change
     + 1 h, start the same background fetch. This is the post-game refresh.
   - Run attribution over a snapshot of the week's plays and return the counts.
3. Merge counts into the players map before `score.NewWeekStats`. A credited forcer is always in the
   aggregate, but may have `null` stats and so no stat line; such a forcer gets a line with only
   `FFTurnover` set.

The poll piggybacks on `statscache`, so it runs at most once per TTL per week. The TTL matches the
CDN's 300 s, so polling more often would gain nothing.

`Client` gains `Plays *PlayStore`. A nil `Plays` keeps today's behavior, so existing tests and
benches are untouched. The weekly aggregate and the plays endpoint share the `api.sleeper.com` host,
so the play store fetches from the client's existing `BaseURL`. `main.go` wires
`sleeper.NewPlayStore(...)` into the client.

*Alternative:* a background ticker that polls every live week. Rejected: it fetches with no readers,
which `weekly-stats-cache` deliberately avoids, and it needs a definition of "live week".

*Alternative:* block on the whole-week fetch for a cold week. Rejected as unbounded: 3–12 s on the
request path. Amended by `plays-cold-wait`: a cold week now waits on its in-flight whole-week fetch
for up to `PLAYS_COLD_WAIT` (default `30s` locally, `0` on Fly), so a fresh process scores turnover
forced fumbles on first load instead of 5 minutes later. Gap fills and post-game refreshes still never
wait.

### D5. The poll-failure fallback is "what is held", not zero

If the poll fails but earlier polls or a whole-week fetch filled the store, the held plays are scored.
Zero is the result only when nothing is held. Dropping known awards because one 300-play poll failed
would make scores flicker.

### D6. Fixtures are trimmed to forced-fumble plays in the REST shape

The 2026 week 2/3 GraphQL recordings are about 5 MB each. A one-off script, kept out of the repo,
keeps only plays with an `idp_ff` row or "forced by" in the description, and writes them as
`internal/sleeper/testdata/plays_2026_w2.json` and `plays_2026_w3.json` in the REST `recent` response
shape. One live `recent` call confirms the envelope before recording. Confirmed 2026-10-04: the response is a
bare JSON array of plays (no `data` wrapper). Each play has `play_id`, `game_id`, `sequence`, `updated_at`,
`metadata` (`description`, `team`, `opponent`, `quarter_name`, `time_remaining_minutes`/`_seconds`) and
`play_stats[{player_id, stats, game_id, play_id}]`. REST rows carry no `player` object, unlike GraphQL, so
the fixtures drop the GraphQL `player` objects, and names and teams come from the weekly aggregate (see
D2). `stats_2026_w2.json` and `stats_2026_w3.json` are the live aggregate for those weeks, trimmed to
the players on fixture plays and kept in the live row shape. Ground-truth tests join
`ff-test-players.csv` to plays on `quarter_name` + `time_remaining_minutes:seconds`, filtered by the
CSV team appearing as `metadata.team` or `metadata.opponent`. Sleeper `game_id` and nflverse
`game_id` do not share a format.

### D7. Logging goes through an injectable `logf`

Unresolved attributions and fetch failures are logged through an injectable `logf`, as in
`statscache`, so tests can assert that a play was logged.

## Risks / Trade-offs

- **A forcer missing from the aggregate is not paid.** The play rows name no one, so a forcer with no
  aggregate entry cannot be identified. → Logged, not credited. Not observed on live week 3.
- **Description format drift.** If Sleeper changes "FUMBLES, forced by", attribution silently goes to
  zero. → Every `idp_ff` turnover row without a resolved forcer is logged, so drift shows up as log
  noise rather than silence.
- **Unknown: whether `recent` resurfaces corrections to old plays.** → Mitigated by the post-game
  whole-week refresh.
- **Unknown: Sleeper rate limits.** The poll is bounded by the cache TTL. Whole-week fetches happen
  once per week per process, plus gap fills and post-game refreshes. → Log every whole-week fetch
  with its duration so the budget is visible.
- **Unknown: live mid-game behavior** (how quickly `idp_ff` and `fum_lost` appear and settle). →
  The rule tolerates awards arriving up to an hour late. Re-check against a live Sunday after
  shipping.
- **Gap detection uses `updated_at`, not `sequence`.** A corrected old play has a new `updated_at`,
  so it may be mistaken for overlap. → It only suppresses a whole-week fetch that the post-game
  refresh will still trigger.
- **Memory.** A whole week's trimmed plays are retained per week viewed. Full plays are ~4 MB of JSON
  per week. → Retain only the fields attribution needs, not raw JSON.

## Migration Plan

Additive. Deploying wires the store, and the first read of each week seeds it in the background. To
roll back, revert the wiring in `main.go`: `Plays: nil` restores aggregate-only behavior.

## Open Questions

Rules questions to record, not resolve:

- **(a)** A fumble lost and then recovered back by the original team on the same play. Per-fumble
  scoring from the fumbler's `fum_lost` is chosen as the simplest reading. Confirm with the
  commissioners.
- **(b)** An offensive player forcing a turnover after an interception, such as a QB forcing a fumble
  on the return. The algorithm would credit the forcer if the returner's row shows `fum_lost`. Does the
  rule pay an offensive player?

Known unknowns: `recent` and corrections, rate limits, live mid-game behavior (see Risks).
