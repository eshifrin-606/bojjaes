## Context

The server currently exposes `GET /score`, which fetches Sleeper's weekly stats aggregate, pulls one hardcoded
player out of it, scores the rushing/receiving slice of the HMFFL rules, and returns JSON. Player, season, and
week are compile-time constants, so `FetchFunc` takes only a `context.Context`.

The upstream endpoint, `/v1/stats/nfl/{season_type}/{season}/{week}`, returns a map of every player in the
league keyed by player ID — roughly half a megabyte to read six numbers for one player. Scoring a lineup one
player at a time would repeat that download per player.

A live probe informed the absence handling in this design. Against 2025 regular week 14 and the in-progress
2026 preseason week 1, cross-referencing the stats payload with `/schedule/nfl/pre/2026` (which reports each
game as `complete`, `in_game`, or `pre_game`), counting active QB/RB/WR/TE only:

| game status | absent from payload | entry, no stats | entry with stats |
| --- | --- | --- | --- |
| `pre_game` | 244 | 109 | 0 |
| `in_game` | 2 | 1 | 54 |
| `complete` | 28 | 124 | 388 |

Three facts follow. A player who dressed and recorded nothing is present with a stub entry, whose missing keys
already read as zero, so scoring 0.0 needs no new code. A player whose game has not kicked off is usually
absent but sometimes carries a seeded stub with `gp: 1` and no stats, so `gp` does not mean "played". And 28
active players were absent from a *completed* week — inactives and scratches — which the current rule reports
as a 502 on entirely valid input.

A future week returns `200 {}`, an empty payload, so "week has not happened" looks like every player missing
rather than like an error.

Absence has a fourth cause beyond the three above. The stats endpoint carries a one-hour edge TTL (recorded in
`docs/basic-memory/2026-08-09-sleeper-stats-endpoint-has-1h-edge-ttl-espn-is-seconds.md`), so a player whose
game has just finished can read absent purely because of the CDN.

The full probe, its reproduction script, and the findings that outlive this change are recorded in
`docs/basic-memory/2026-08-15-sleeper-weekly-stats-absence-is-routine-ambiguous-and-sometimes-stale.md`.

## Goals / Non-Goals

**Goals:**

- Score many players for one season and week in a single upstream request.
- Never serve a fantasy point total that is not backed by real stats, and never suppress a genuine 0.0.
- Stop reporting valid player IDs as server errors when the player simply has no stats yet.
- Keep the change small enough to ship without adding a cache, a background refresh, or a new dependency.

**Non-Goals:**

- Validating player IDs. Distinguishing a typo from a player who has not played requires the 14.6 MB player
  index, which needs a cache and a refresh policy.
- Subdividing `no_stats` by root cause (not yet kicked off, inactive, unknown ID). The bucket is a place to add
  that later without reshaping the response.
- Deduplicating repeated player IDs. Repeats are echoed once per occurrence; the caller is trusted.
- Scoring positions beyond rushing and receiving. Passing, kicking, defense, and two-point conversions remain
  out of scope, so a quarterback's ID still yields a confidently incomplete number.
- Preseason and postseason. Only the `regular` season type is fetched.

## Decisions

### Absence is reported per player, not as a request failure

The stats payload cannot distinguish a bad ID from a player who has not played — the probe table above shows
absence spanning both. Three options were weighed:

1. Keep absence as an error. Zero new code, but 502s the 28 valid inactives in a completed week and fails an
   entire request over one typo.
2. Treat an empty payload as "week not started" and score everyone zero, keeping absence-within-a-populated-
   payload an error. One `len()` check, but it still 502s inactives and pre-kickoff players, so it does not
   deliver the behavior we want.
3. Validate IDs against the player index. Fully disambiguates, and yields player names and positions as a
   bonus, but costs a 14.6 MB startup fetch, a cache, a refresh policy, and a cold-start failure mode.

The chosen fourth option reports absent players in the response and returns 200. It needs no new machinery, it
never serves an invented zero, and it never rejects valid input. The caller determines root cause. Option 3
remains available later purely as an addition, since the `no_stats` bucket is where that detail would land.

### Two buckets rather than one list with optional fields

The response separates scored players from absent ones:

```json
{
  "season": 2025,
  "week": 14,
  "scores":   [ { "stats": { "player_id": "9493", ... }, "points": 24.5 } ],
  "no_stats": [ "4034", "banana" ]
}
```

The alternative — one ordered list where each entry carries optional `stats`/`points` pointers plus a flag —
keeps request ordering and guarantees every requested ID appears exactly once. It was rejected because its
correctness depends on the pointer types and `omitempty` tags surviving future edits. Two plausible
simplifications reintroduce the exact bug this change exists to avoid: dropping the pointer on `Points`
serializes an absent player as `"points": 0`, and adding `omitempty` to a non-pointer `Points` erases a
genuine 0.0 score. Both compile, and both pass tests that only assert on players who scored.

With two buckets, `no_stats` is a `[]string`. There is nowhere to put a number, so the invariant is carried by
the type rather than by remembering. `Score` needs no pointers and no `omitempty`, so a real 0.0 always
serializes. This mirrors the reasoning already recorded in `internal/score/stats.go`, where `StatLine` omits a
points field so it cannot carry a stale total.

The trade-off is real: splitting into two lists weakens the guarantee that every requested ID appears exactly
once. A test asserting `len(scores) + len(no_stats) == len(player_ids)` covers it.

### The transform reports presence instead of returning an error

`FetchStatLine` splits in two:

```
fetchWeekly(ctx, baseURL, season, week) → map[string]map[string]float64   ← one HTTP call
statLineFrom(raw, playerID, season, week) → (StatLine, bool)              ← per player
```

The boolean replaces the current "player %s not found" error. Absence stops being exceptional and becomes a
value, which is what makes bucketing fall out for free. Transport and decode failures remain errors and still
fail the whole request — those are genuinely exceptional and affect every requested player equally.

### Request validation

`POST /scores` with a JSON body. Season and week must be present and in range; `player_ids` must be non-empty
and at most 26 entries. Twenty-six is the league's maximum roster size, comfortably more than two full starting
lineups. The cap is a sanity bound rather than a cost control — the upstream request is the same size
regardless of how many IDs are requested.

### `GET /score` becomes a smoke test over the batch path

It keeps its current `ScoreResponse` shape and its 502-on-absence behavior, now implemented as a one-element
batch call for Nacua in 2025 week 14. That week is settled history where he is known present, so absence there
signals a real breakage and the 502 keeps its diagnostic value. Retaining it costs a handful of lines and keeps
the documented `curl` in `README.md` working.

## Risks / Trade-offs

- **A typo'd player ID returns 200 with the ID in `no_stats`, not an error.** → Accepted deliberately. The
  response names the ID rather than silently omitting or zeroing it, so the caller can see and diagnose it. The
  player index would resolve this and is scoped as a follow-on.
- **A player can land in `no_stats` because of cache lag rather than because they have no stats.** The
  one-hour edge TTL means a just-finished game may not be reflected yet. → Not addressable from this endpoint,
  and it resolves itself on the next request after the TTL expires. It is one more reason the server must not
  claim to know why a player is absent.
- **A caller could ignore `no_stats` and treat the response as a complete lineup.** → The bucket is always
  present in the JSON, and a lineup total computed from `scores` alone will visibly disagree with the roster
  size. Documented in the README.
- **Splitting the response into two lists lets an ID go missing from both.** → Covered by a test asserting the
  two bucket lengths sum to the number of requested IDs.
- **A quarterback's ID scores as if the only stats that exist are rushing and receiving.** → Pre-existing, and
  a batch endpoint makes it easier to hit by accident. Recorded in the spec's purpose rather than fixed here;
  broader position coverage is the next change.
- **`GET /score` and `POST /scores` now maintain two response shapes.** → Accepted for the smoke test's value.
  If the shapes drift, delete `GET /score`.
