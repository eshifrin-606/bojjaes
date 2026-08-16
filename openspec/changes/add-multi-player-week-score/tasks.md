Follow the red-green TDD loop from `CLAUDE.md`: write the failing test first, then the code that passes it.

## 1. Split the Sleeper transform

- [ ] 1.1 Extend `internal/score/testdata/week14.json` with a stub entry carrying no mapped stat keys (mirroring
      Sleeper's `{gms_active, pos_rank_*}` shape) so "present but scoreless" is covered by a fixture
- [ ] 1.2 Write failing tests for `statLineFrom(raw, playerID, season, week) (StatLine, bool)`: present player
      maps correctly, stub entry maps to an all-zero stat line with `true`, absent player returns `false`
- [ ] 1.3 Write a failing test that a `nil` entry in the payload is treated as absent, not as a zero stat line
- [ ] 1.4 Implement `statLineFrom` and make 1.2 and 1.3 pass
- [ ] 1.5 Write failing tests for `fetchWeekly(ctx, baseURL, season, week)`: decodes the payload, and returns an
      error naming season and week on transport failure, non-200 status, and undecodable body
- [ ] 1.6 Implement `fetchWeekly` and make 1.5 pass
- [ ] 1.7 Reimplement `FetchStatLine` on top of `fetchWeekly` + `statLineFrom`, keeping its existing
      error-on-absence signature so `GET /score` is unaffected; confirm existing tests still pass

## 2. Batch response types

- [ ] 2.1 Write a failing test asserting a scored player with zero fantasy points serializes with an explicit
      `"points": 0` (guards against `omitempty` erasing a genuine zero)
- [ ] 2.2 Define the batch request and response types with the absent bucket as `[]string`, and make 2.1 pass
- [ ] 2.3 Write a failing test asserting both buckets serialize as empty arrays rather than `null` when empty

## 3. Batch handler

- [ ] 3.1 Write failing handler tests over an `httptest` Sleeper stub: all players present, a mix of present and
      absent, every player absent (empty payload), and a present-but-scoreless player reported as scored with
      zero rather than absent
- [ ] 3.2 Write a failing test asserting `len(scores) + len(no_stats) == len(player_ids)`, including when the
      same ID is repeated (echoed once per occurrence)
- [ ] 3.3 Write failing validation tests, each asserting no upstream request is made: non-JSON body, missing or
      out-of-range season, missing or out-of-range week, empty player list, and 27 player IDs
- [ ] 3.4 Write a failing test that exactly 26 player IDs is accepted
- [ ] 3.5 Write a failing test that an upstream failure returns 5xx with an error message and no partial body
- [ ] 3.6 Implement the batch handler and make section 3 pass

## 4. Wire up the server

- [ ] 4.1 Register `POST /scores` in `cmd/server/main.go`, rejecting other methods on that path
- [ ] 4.2 Reimplement `GET /score` as a one-element batch call for the Nacua constants, preserving its
      `ScoreResponse` shape and its 502 when he is absent; confirm existing handler tests still pass
- [ ] 4.3 Retire `FetchFunc` if nothing still uses its context-only shape
- [ ] 4.4 Update the package comment in `cmd/server/main.go`, which still describes the walking skeleton

## 5. Documentation

- [ ] 5.1 Add a `POST /scores` curl example to `README.md` alongside the existing one, showing a response with a
      populated `no_stats` bucket, noting that absence does not identify its own cause, and noting that season
      and week are read as regular season
- [ ] 5.2 Amend `openspec/specs/player-week-score/spec.md` in place with the delta, and note in its Purpose that
      scores are only meaningful for rushing/receiving production
- [ ] 5.3 Run `go vet ./...` and the full test suite
- [ ] 5.4 Move the flexibility task in `TODO.md` to Recently Completed, and add follow-ons: player-ID validation
      via the Sleeper player index, `no_stats` root-cause granularity, and scoring for positions beyond
      rushing/receiving
