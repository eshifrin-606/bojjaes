## Why

The scoring server only answers for one hardcoded player in one hardcoded week (Puka Nacua, 2025 week 14). A
fantasy lineup is scored as a set, and the Sleeper weekly stats aggregate already returns every player in the
league on a single request — so scoring a whole roster costs the same upstream call as scoring one player.
Batching is closer to the provider's natural shape than the current single-player fetch is.

The current "player absent from the payload is an error" rule also turns out to be wrong. A probe of live
Sleeper data (2025 regular week 14 and the in-progress 2026 preseason week 1) showed absence is routine and
benign:

- A player whose game has not kicked off is usually absent, but sometimes present with a stub entry carrying
  no stats — Sleeper seeds part of the slate early, so absence is a timing artifact.
- A player who dressed but recorded nothing is present with a stub entry, and already scores 0.0 correctly.
- In a *completed* week, 28 active skill-position players were absent entirely — inactives and healthy
  scratches. Today those valid player IDs produce a 502.

Nothing in the stats payload distinguishes a bad player ID from a player who has not played. Only the 14.6 MB
player index can, and buying that disambiguation is not worth its cost yet.

## What Changes

- Add `POST /scores` accepting `{season, week, player_ids[]}` and returning stats and fantasy points for each
  requested player.
- Report players missing from the Sleeper payload in a separate `no_stats` bucket of player IDs instead of
  failing the request. The caller sees which IDs produced nothing and diagnoses the cause; the server does not
  guess between "hasn't played" and "bad ID".
- **BREAKING** (spec-level): a player absent from the weekly payload is no longer an error from the transform.
  Absence becomes a value the caller handles, not an exception.
- Split the Sleeper fetch from the per-player mapping so one upstream request serves many players.
- Reimplement `GET /score` on top of the batch path as a fixed smoke test, keeping its existing response shape
  and its 502-on-absence behavior. Week 14 2025 is a settled historical week where Nacua is known present, so
  absence there means something broke.
- Validate requests: reject a missing or out-of-range season or week, an empty `player_ids`, and more than 26
  IDs (the league maximum roster size).

Out of scope, deliberately: player-ID validation against the Sleeper player index, subdividing `no_stats` into
root causes, deduplicating repeated IDs, scoring positions beyond rushing/receiving, and selecting a season
type. The endpoint is regular-season only; preseason would serve local manual testing against a live
in-progress week, not the league. Each is a clean follow-on that does not change this contract's shape.

## Capabilities

### New Capabilities

None. This extends the existing scoring capability rather than introducing a separate one.

### Modified Capabilities

- `player-week-score`: the Sleeper transform no longer errors on an absent player; the capability gains a batch
  scoring endpoint alongside the existing fixed-target endpoint; the batch endpoint is specified as regular
  season only; request validation limits are specified; the fixed-target endpoint is respecified to convert an
  absent player into a 502 itself, since the transform no longer does.

## Impact

- `internal/score/sleeper.go` — `FetchStatLine` splits into a weekly-payload fetch and a per-player mapping
  that reports presence rather than returning an error. `NacuaPlayerID`, `TargetSeason`, and `TargetWeek` stay
  for the smoke-test endpoint.
- `internal/score/handler.go` — new batch handler and response types; `FetchFunc`'s closure-over-constants
  shape no longer fits, since parameters now arrive in the request body.
- `cmd/server/main.go` — registers `/scores`; `/score` rewires onto the batch path.
- `openspec/specs/player-week-score/spec.md` — amended in place.
- `README.md` — documents the new endpoint alongside the existing curl.
- No new dependencies. Upstream request volume per scored roster drops from N requests to one.
