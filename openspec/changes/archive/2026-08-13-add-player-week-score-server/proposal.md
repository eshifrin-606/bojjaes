## Why

The repo has three ADRs, a probe write-up, and a settled scoring spec — and zero lines of code. The
top item on [TODO.md](../../../TODO.md) is "score single player single week (high) — first piece of
code." This change closes that gap with a deliberately throwaway walking skeleton: one player, one
week, one HTTP endpoint, built red-green so the scoring rules get pinned by tests from the start.

The point is not the artifact. The point is to run the whole path end to end — Sleeper request →
domain stats → league points — and find out what actually hurts before committing to structure.

## What Changes

- New Go module at the repo root with a single `cmd/` server binary.
- A **domain object** holding a player's NFL stats for one week, plus the fantasy points computed
  from them.
- A **Sleeper transform**: fetch the Sleeper weekly stats aggregate and map one player's raw stat
  keys onto the domain stats object.
- A **fantasy points calculator** implementing only the rushing/receiving slice of
  [docs/scoring.md](../../../docs/scoring.md) — the rules a wide receiver can actually trigger:
  the yardage bonus, rush/rec touchdowns, the 40+ yard touchdown bonus, and fumbles lost.
- An **HTTP server**: hitting it scores Puka Nacua for 2025 regular-season week 14, prints the score
  to stdout, and returns it as JSON.
- Tests written before implementation at each layer (calculator, transform, handler).

**Non-goals** (explicitly deferred, not forgotten): the ADR 0002 provider interface, GraphQL/PBP,
play-by-play-derived rules (forced fumbles, safeties, defensive TD distance), name→ID mapping,
passing / kicking / defensive / two-point scoring, caching, lineups, and any parameterization of
player or week. All of those are real; none of them are needed to prove the path works.

## Capabilities

### New Capabilities

- `player-week-score`: fetching a single player's stats for a single NFL week from Sleeper,
  representing them as a provider-neutral domain object, computing HMFFL fantasy points from them,
  and serving the result over HTTP.

### Modified Capabilities

None. `openspec/specs/` is empty; this is the first capability.

## Impact

- **New code:** `go.mod`, `cmd/server/`, and a small package holding the stats domain object,
  Sleeper transform, and calculator. No existing code to break.
- **External dependency:** Sleeper's undocumented REST weekly stats endpoint
  (`GET /v1/stats/nfl/regular/{season}/{week}`), per ADR 0003. Unauthenticated, ~570 KB per call.
  Hit live at request time — no fixture-only mode, no cache.
- **Hardcoded values:** Puka Nacua's Sleeper `player_id`, season `2025`, week `14`. This is the
  shortcut that makes the change small, and the first thing a follow-up change should remove.
- **Docs:** none updated. This change deliberately implements a subset of `docs/scoring.md`; the
  gap is scope, not a rules deviation, so the ADRs and scoring doc stand as written.
- **Dependencies:** Go stdlib only (`net/http`, `encoding/json`, `testing`).
