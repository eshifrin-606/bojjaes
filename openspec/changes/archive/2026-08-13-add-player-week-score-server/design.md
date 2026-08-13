## Context

First code in the repo. Three ADRs already settle the big questions (Sleeper as sole provider, a
provider interface, on-demand serving, hardcoded scoring), so this change has no architecture left
to invent — it only has to walk the path once and find out what hurts. Go, chosen by the user, with
stdlib only.

The one design constraint that matters: this is throwaway, and treating it otherwise is the failure
mode. Every decision below picks the smaller thing.

## Goals / Non-Goals

**Goals:**

- A running server that, when hit, produces a real HMFFL score for Puka Nacua in 2025 week 14.
- Red-green TDD: a failing test precedes every piece of behavior, at three layers — calculator,
  transform, handler.
- Three named seams — stat line, transform, calculator — so the next change has something to widen.

**Non-Goals:**

- ADR 0002's provider interface. One concrete function, no interface, until a second provider exists.
- GraphQL / play-by-play. The REST weekly aggregate is enough for a WR.
- Any parameterization: player, season, and week are constants.
- Caching, freshness handling, `updated_at` plumbing, retries, timeouts beyond a default.
- Persistence, config files, logging framework, Docker, CI.

## Decisions

### Layout: one package plus a `main`

```
go.mod                       module github.com/eshifrin/bojjaes
cmd/server/main.go           wires the handler, listens on :8080
internal/score/stats.go      StatLine domain object
internal/score/calc.go       Points(StatLine) float64
internal/score/sleeper.go    FetchStatLine(season, week, playerID) (StatLine, error)
internal/score/*_test.go
```

One package, `score`, holding all three concepts. Splitting `domain` / `provider` / `scoring` into
separate packages at ~150 lines buys import ceremony and nothing else. *Alternative considered:*
a flat `main` package — rejected because `_test.go` on unexported helpers in `main` makes the
calculator awkward to test in isolation, which is the whole point of the exercise.

### Domain object: a flat struct with points computed, not stored

```go
type StatLine struct {
    PlayerID  string
    Season    int
    Week      int
    RushYd    int
    RecYd     int
    RushTD    int
    RecTD     int
    TD40Plus  int   // rush_td_40p + rec_td_40p
    FumLost   int
}
```

Points come from `Points(s StatLine) float64`, a pure function — not a field on the struct, and not
a method that mutates. The proposal asks for "a domain object which holds nfl stats and fantasy
points"; the served JSON satisfies that by carrying `{stats: …, points: …}` as a small response
struct built at the boundary. Keeping the computed value out of the stat line means the calculator
stays trivially testable with struct literals and there is no way to hold a stale total.

Ints throughout; Sleeper sends JSON numbers that are floats, so the transform reads `float64` and
converts. Points are `float64` because the rules pay in halves.

*Alternative considered:* a `map[string]float64` of raw Sleeper keys carried through to the
calculator. Rejected — it is less code today and re-introduces exactly the vendor-shape leak ADR
0003 spent a section preventing.

### Yardage bonus: max over three clauses

The rules give three qualifying paths (80 rush, 80 rec, 100 combined) and clarify the award is made
once. Implement as: compute `3 + 0.5*floor(excess/10)` for each clause that qualifies, take the max,
zero if none qualify. This satisfies "awarded once" and resolves the increment-basis ambiguity the
scoring doc leaves open (increments count from *that clause's* threshold) in the player's favor,
which matches how the 80/80 case reads in the spreadsheet.

Integer arithmetic on tenths would avoid float comparison noise, but at these magnitudes `float64`
is exact for halves. Not worth the ceremony.

### Sleeper access: fetch the whole week, index by player ID

`GET https://api.sleeper.app/v1/stats/nfl/regular/2025/14` returns `map[string]map[string]float64`
keyed by Sleeper player ID. Decode straight into that type, pull one key, map the six stats. ~570 KB
per request for one player is wasteful and completely irrelevant at one request per manual hit.

`FetchStatLine` takes a base URL so tests point it at an `httptest.Server` with a hand-written
two-player fixture. No live network in tests. *Alternative considered:* GraphQL
`stats_for_players_in_week` at 1.9 KB — better in every way that does not matter here, and it costs
a POST body and a nested response shape.

Missing stat key means zero, per ADR 0003. Missing *player* is an error, not a zero score — a silent
zero is the failure this change would most easily hide.

### Puka Nacua's Sleeper ID

Looked up once during implementation (`GET /v1/players/nfl`, or by grepping the week-14 payload
against a known stat line) and pasted into a `const` beside season and week. The first task in
`tasks.md` is to confirm it; the whole change is meaningless if it scores the wrong player.

### TDD order

Calculator first (pure, no I/O, where the real rules live and where red-green pays), then the
transform against a fixture server, then the handler against `httptest`. Each layer: write the
failing test, watch it fail for the right reason, implement the minimum.

## Risks / Trade-offs

- **The scored number is unverifiable.** We have no independent source for Nacua's week-14 HMFFL
  score, so a passing test suite plus a plausible number is all the confidence we get. → Hand-check
  the raw stat line against a box score once, and print stats alongside points so the arithmetic is
  auditable by eye.
- **Hardcoded IDs rot silently.** A wrong `player_id` produces a confident wrong answer. → The
  response echoes the player ID and the fetched stats; a wrong player is visible in the output.
- **Sleeper is undocumented and could change or rate-limit.** → One manual request per hit; nothing
  to abuse. ADR 0003 already owns this risk.
- **"Throwaway" code that survives.** The likeliest outcome is this becomes the seed of the real
  thing with its shortcuts intact. → Shortcuts are listed in the proposal's Impact section as the
  explicit removal list for the next change.
- **The max-over-clauses reading is an interpretation.** The scoring doc does not state the
  increment basis for multi-clause qualification. → Flagged here; if the league reads it otherwise
  it is a one-line change in `calc.go`.
