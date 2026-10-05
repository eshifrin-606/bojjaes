Each task is one red-green behavior unless marked otherwise. When a test needs a new symbol, first
add it as a stub that compiles and returns the zero value. That way the red shows a value mismatch,
not a build error.

## 1. Fixtures

- [x] 1.1 Make one live `GET https://api.sleeper.com/plays/nfl/recent?season_type=regular&season=2026&week=3&limit=5`
      call to confirm the response envelope (bare array or wrapped). Record the finding in design.md
      D6.
- [x] 1.2 With a throwaway script in the scratchpad, not the repo, trim the 2026 wk 2 and wk 3
      GraphQL recordings (`.../scratchpad/plays_w2.json`, `plays_w3.json`). Keep only plays with an
      `idp_ff` row or "forced by" in `metadata.description`, written in the `recent` shape, to
      `internal/sleeper/testdata/plays_2026_w2.json` and `plays_2026_w3.json`. Keep `player` objects
      and team rows.

## 2. Domain: pay turnover-qualified forced fumbles

- [x] 2.1 Stub: add `FFTurnover int` (`json:"ff_turnover"`) to `score.StatLine`, with a why-comment
      that it is turnover-qualified before it arrives. No `Points` change.
- [x] 2.2 Red-green: one forced fumble with no other production scores 4.
- [x] 2.3 Red-green, or confirm green: two forced fumbles score 8.
- [x] 2.4 Confirm green: 2.5 sacks + 1 recovery + 1 forced fumble scores 13.5, Will Anderson's
      official wk 3 total.
- [x] 2.5 Correct the `Points` doc comment. The only unimplemented rule is safeties. Drop "forced
      fumbles" and the 40+ bonus on defensive/return touchdowns.

## 3. Description parsing (pure)

- [ ] 3.1 Stub: `forcedByPairs(description string) []fumblePair` returning nil.
- [ ] 3.2 Red-green: "C.Keenum FUMBLES, forced by J.Greenard." yields one pair (C.Keenum, J.Greenard).
- [ ] 3.3 Red-green: a description with two fumbles yields both pairs in order (the wk 3 HOU@IND Q1
      11:28 text).
- [ ] 3.4 Red-green: only text after the last "overturned" is read. One case where the pair exists
      only before it (wk 3 LAC@BUF-style Herbert/Oliver text). One where it exists only after
      (Allen/Henley, wk 3 Q4 15:00).
- [ ] 3.5 Red-green: a suffixed name ("K.Moore II") parses as the full name, without swallowing the
      following sentence.
- [ ] 3.6 Red-green: name normalization. `abbrevName(first, last)` and the parsed name compare equal
      across suffixes (`Jr.`, `Sr.`, `II`–`V`) and case.

## 4. Attribution over plays (pure)

- [ ] 4.1 Stub: decode-only `play`/`playRow` types (only the fields attribution needs) and
      `forcedFumbleTurnovers(plays, logf) map[string]int` returning an empty map.
- [ ] 4.2 Red-green: a single lost fumble with an opposing forcer credits the forcer 1, not the
      fumbler (hand-built play).
- [ ] 4.3 Red-green: a fumble with `idp_ff` but no `fum_lost` credits no one.
- [ ] 4.4 Red-green: a "forced by" text with no `idp_ff` row credits no one.
- [ ] 4.5 Red-green: a forcer row with empty stats is still credited (Crosby, wk 3 LV@NO Q4 2:21, from
      fixture).
- [ ] 4.6 Red-green: the forcer must be on the other team. A same-team namesake is not credited.
- [ ] 4.7 Red-green: team rows (non-numeric `player_id`) are ignored as fumbler and forcer.
- [ ] 4.8 Red-green: double fumble, wk 3 HOU@IND Q1 11:28. Anderson gets 1, Alie-Cox 0.
- [ ] 4.9 Red-green: sacker is not the forcer, wk 3 HOU@IND Q4 4:45. D.Hunter gets 1, Anderson 0 from
      that play.
- [ ] 4.10 Red-green: ambiguous forcer (two opposing matches) credits no one and calls `logf` with the
      play ID.
- [ ] 4.11 Red-green: no pair matching the fumbler credits no one and calls `logf`.
- [ ] 4.12 Red-green: overturned fixture plays (wk 2 OT Treadwell, wk 3 Allen/Henley, Willis/Sneed,
      Price/Robertson) are attributed from the final version only.
- [ ] 4.13 Red-green: ground truth. Join every `ff-test-players.csv` row to the fixtures on team,
      quarter, and clock. Assert the row's `sleeper_id` is credited exactly when `ff_turnover=yes`
      (covers Campbell's out-of-bounds non-turnover).

## 5. Play store (pure, in memory)

- [ ] 5.1 Stub: `PlayStore` with `merge(season, week, plays)` and `snapshot(season, week) []play`.
- [ ] 5.2 Red-green: merged plays are returned by snapshot; weeks do not share plays.
- [ ] 5.3 Red-green: a copy with newer `updated_at` replaces the held play.
- [ ] 5.4 Red-green: a copy with older `updated_at` is ignored.
- [ ] 5.5 Red-green: gap check. `needsWholeFetch` is true for an empty week, true when the poll's
      oldest `updated_at` is newer than the newest held, false on overlap.
- [ ] 5.6 Red-green: post-game check, with an injected clock. True when the newest change is more
      than 1 h old and the last whole fetch predates it; false when the last whole fetch completed
      more than 1 h after the newest change.

## 6. Fetching plays

- [ ] 6.1 Red-green: `fetchRecentPlays(ctx, baseURL, season, week, limit)` requests
      `/plays/nfl/recent` with `season_type=regular`, season, week, and limit, and decodes the fixture
      served by `httptest`.
- [ ] 6.2 Red-green: a non-200 response or a bad body returns an error.
- [ ] 6.3 Red-green: `PlayStore.ForcedFumbles` polls with limit 300, merges, and returns the
      attribution counts.
- [ ] 6.4 Red-green: an empty week starts a background whole-week fetch (limit 5000), and
      `ForcedFumbles` returns before it completes. Use a blocking test server and assert the return.
- [ ] 6.5 Red-green: a detected gap starts a background whole-week fetch; an overlapping poll does
      not.
- [ ] 6.6 Red-green: concurrent triggers for one week start only one whole-week fetch.
- [ ] 6.7 Red-green: after the whole-week fetch completes, its plays are attributed on the next call.
- [ ] 6.8 Red-green: a poll failure with nothing held returns zero counts and logs it.
- [ ] 6.9 Red-green: a poll failure with plays held returns counts from the held plays and logs it.
- [ ] 6.10 Red-green: a whole-week fetch failure is logged, keeps held plays, and allows a later
      retry.
- [ ] 6.11 Red-green: the poll is bounded by its own short timeout. A hung server does not hold
      `ForcedFumbles` past it.
- [ ] 6.12 Red-green: a quiet week whose last whole fetch predates its newest change starts one
      post-game refresh.

## 7. Merging into the weekly snapshot

- [ ] 7.1 Red-green: `Client` with `Plays` set credits a fixture forcer's `FFTurnover` on the stat
      line read from `WeekStats`.
- [ ] 7.2 Red-green: a forcer absent from the aggregate appears in the snapshot with only
      `FFTurnover`.
- [ ] 7.3 Confirm green: with `Plays` nil, `WeekStats` behaves as before (existing tests pass
      unchanged).
- [ ] 7.4 Red-green: play-by-play failure still returns the aggregate's stat lines; aggregate failure
      still errors.
- [ ] 7.5 Confirm green: the aggregate `idp_ff` is still not mapped to any stat line field (existing
      exclusion test).
- [ ] 7.6 Wire `PlaysBaseURL: "https://api.sleeper.com"` and a `PlayStore` into the client in
      `cmd/server/main.go`. Run the full suite and `go vet`.

## 8. Docs and spec cleanup

- [x] 8.1 In `openspec/specs/player-week-score/spec.md`, rewrite the Purpose paragraph (~line 10).
      Only safeties remain unscored, and the defensive/return 40+ bonus is not a rule. The
      requirement deltas themselves land via sync/archive.
- [x] 8.2 In `docs/scoring.md`, remove the forced-fumble bullet from "Planned implementation
      deviations". Add open rules questions (a), lost-then-recovered-back on the same play, and (b), an
      offensive player forcing a turnover after an interception. Do not edit
      `docs/adr/0003-sleeper-as-initial-stat-provider.md`.
- [ ] 8.3 Run `openspec validate score-forced-fumbles` and fix any findings.
