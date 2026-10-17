Each task is one red-green behavior unless marked otherwise. When a test needs a new symbol, first
add it as a stub that compiles and returns a wrong value. That way the red shows a value mismatch,
not a build error. "Confirm green" tasks pin behavior the previous step already gives; break the code
briefly to see each one fail. "Refactor" tasks change no assertion: the existing suite is the test.

Each numbered group is one batch and can be done from a fresh context with the design and the spec
files named in its heading.

## 1. Refactor: the cache becomes `weekcache.Cache[V]` (D4, D5)

No behavior changes in this group. Run `go test -race ./...` after each task.

- [x] 1.1 Refactor: move `internal/statscache` to `internal/weekcache`, package name included, and
      fix the imports in `cmd/server`. Suite green.
- [x] 1.2 Refactor: replace the `StatsSource` interface the cache wraps with a function type
      `func(ctx, season, week) (V, error)` and make `Cache` generic over `V`. `main` passes
      `client.WeekStats` as a method value. Test fakes are passed the same way. Suite green.
- [x] 1.3 Refactor: `New` takes a label and the hit and miss lines are prefixed with it. `main`
      passes `statscache`, so every existing log assertion still passes unchanged.
- [x] 1.4 Refactor: rename `WeekStatsAsOf` to `AsOf` on the cache, on `web.StatsSource`, and in the
      `web` and `cmd/server` test fakes. Suite green.
- [x] 1.5 Confirm green: `*weekcache.Cache[score.WeekStats]` is passed to `newMux` with no adapter
      type, and `plays.OnWholeWeekChanged(stats.Invalidate)` still compiles.
- [x] 1.6 Update the package comment and `docs/package-dependencies.md` for the new name and the
      function source.

## 2. `games`: the domain (D2; `specs/game-schedule`)

- [ ] 2.1 Create `internal/games` with `State` (`Upcoming`, `InProgress`, `Complete`, `Other`),
      `Game`, and `Week`. Stub `NewWeek`, `ForTeam` returning `false`, and `Empty` returning `false`.
- [ ] 2.2 Red-green: `ForTeam("ARI")` on a week holding `DET` at `ARI` returns that game.
- [ ] 2.3 Red-green: `ForTeam("DET")` returns the same game, found as the away team.
- [ ] 2.4 Confirm green: `ForTeam("KC")` on a week without `KC` reports `false`; `ForTeam("ari")`
      reports `false`.
- [ ] 2.5 Red-green: `Empty` is `true` for a week built from no games.
- [ ] 2.6 Confirm green: `Empty` is `false` for a week with one game. `go list -deps` shows `games`
      imports no project package.

## 3. `sleeper`: fetch the week's games (D1, D3, D11; `specs/game-schedule`, `specs/upstream-call-logging`)

Tests serve fixtures from an `httptest.Server`, as the stats and plays fetch tests do. Record one
real response as `testdata`; a capture from 2026-10-12 with `F`, `F/OT`, and empty quarter text is
in the exploration scratchpad if it is still there, otherwise fetch a fresh one.

- [ ] 3.1 Stub `func (c Client) WeekGames(ctx, season, week) (games.Week, error)` returning an empty
      week and no error.
- [ ] 3.2 Red-green: the request path is `/scores/nfl/regular/{season}/{week}`.
- [ ] 3.3 Red-green: one game's `metadata.home_team` and `away_team` become `Home` and `Away`, found
      through `ForTeam`.
- [ ] 3.4 Red-green: `status` `pre_game` is `Upcoming`.
- [ ] 3.5 Red-green: `in_game` is `InProgress` and `complete` is `Complete`.
- [ ] 3.6 Red-green: `status` `postponed` is `Other` with `Status` equal to `postponed`.
- [ ] 3.7 Red-green: `start_time` `1791750300000` gives `Kickoff` equal to 2026-10-11T20:25:00Z.
- [ ] 3.8 Red-green: `metadata.quarter` `HALF` gives `Period` `HALF`.
- [ ] 3.9 Red-green: a null `quarter` gives an empty `Period` and no error.
- [ ] 3.10 Red-green: a game with no `home_team` is skipped and the other games are kept.
- [ ] 3.11 Red-green: a 503 fails with an error naming the season and week.
- [ ] 3.12 Red-green: a 200 whose body is a JSON object, not a list, fails with an error naming the
      season and week.
- [ ] 3.13 Confirm green: a 200 with `[]` succeeds and the week is `Empty`.
- [ ] 3.14 Red-green: a successful fetch logs `sleeper games {season} w{week}: {n} games in
      {duration}`.
- [ ] 3.15 Confirm green: the recorded fixture decodes to the number of games it holds, with at
      least one `Complete` game whose `Period` is `F`.

## 4. `weekcache`: games under the same rules (`specs/weekly-stats-cache`, `specs/upstream-call-logging`)

These are mostly confirmations, since the generic cache already does this. They exist so the games
entry is pinned by name.

- [ ] 4.1 Confirm green: a `Cache[games.Week]` over a counting fake serves two requests a minute
      apart from one call, and makes a second call six minutes later.
- [ ] 4.2 Confirm green: a failed games fetch is not stored; the next request calls again.
- [ ] 4.3 Confirm green: with a stats cache and a games cache over counting fakes, invalidating the
      stats week leaves the games week served from cache.
- [ ] 4.4 Confirm green: a games cache built with the label `gamescache` logs `gamescache hit:
      {season} week {week}, {age} old`, and a stats cache hit logs no `gamescache` line.

## 5. `web`: the status text (D7; `specs/matchup-page`)

A pure function, tested with a table and no server. Kickoff is returned separately from text so the
template can put it in a `<time>` element.

- [ ] 5.1 Stub the function: given a lineup team, a `games.Week`, and whether the games are
      available, it returns empty text and a zero kickoff.
- [ ] 5.2 Red-green: an `Upcoming` game returns its kickoff and empty text.
- [ ] 5.3 Red-green: `InProgress` with `Period` `3` returns `Q3`.
- [ ] 5.4 Red-green: `InProgress` with `Period` `HALF` returns `HALF`.
- [ ] 5.5 Red-green: `InProgress` with an empty `Period` returns `Live`.
- [ ] 5.6 Red-green: `Complete` with `Period` `F` returns `Final`.
- [ ] 5.7 Red-green: `Complete` with `Period` `F/OT` returns `Final/OT`.
- [ ] 5.8 Confirm green: `Complete` with an empty `Period` returns `Final`; `Complete` with a
      `Period` not beginning with `F` returns it as sent.
- [ ] 5.9 Red-green: `Other` with `Status` `postponed` returns `postponed`.
- [ ] 5.10 Red-green: a team with no game in a non-empty week returns `Bye`.
- [ ] 5.11 Red-green: team `FA` returns nothing, in a week where other teams have games.
- [ ] 5.12 Red-green: an empty week returns nothing for every team.
- [ ] 5.13 Confirm green: games marked unavailable return nothing for every team.

## 6. `web`: the handler and the markup (D6, D8, D9; `specs/matchup-page`)

- [ ] 6.1 Declare `GamesSource` with `AsOf`, add it to `Handler`'s parameters with a fake that
      returns an empty week, and pass the fake everywhere `Handler` is constructed, `newMux`
      included. No markup change. Suite green.
- [ ] 6.2 Red-green: with `CIN` complete, `Ja'Marr Chase,WR,CIN`'s meta line reads
      `WR · CIN · Final`, with `Final` inside a `span.game`.
- [ ] 6.3 Red-green: with the games fake returning an error, the response is 200, the points are
      unchanged, and that meta line reads `WR · CIN` with no trailing middle dot and no `span.game`.
- [ ] 6.4 Confirm green: an empty week gives the same line as 6.3, and the page contains no `Bye`.
- [ ] 6.5 Red-green: an upcoming game's line contains a `<time>` whose `datetime` is the kickoff in
      RFC 3339.
- [ ] 6.6 Red-green: that `<time>`'s served text for a 2026-10-11T17:00:00Z kickoff is
      `Sun 12:00 PM CDT`.
- [ ] 6.7 Confirm green: a 2026-10-13T00:15:00Z kickoff is served as `Mon 7:15 PM CDT`, and a
      December kickoff carries `CST`.
- [ ] 6.8 Red-green: the games source is called exactly once per request, and two starters on `CIN`
      in opposite columns show the same state.
- [ ] 6.9 Confirm green: a starter with no stats shows `--` and still shows its game state. A `KC`
      starter in a week without `KC` shows `Bye`. An `FA` starter shows `WR · FA` and nothing after.
- [ ] 6.10 Confirm green: with the stats fake failing and the games fake succeeding, the response is
      still 502.
- [ ] 6.11 Confirm green: the as-of line still contains exactly one `<time>`. Change the existing
      assertion from "the page contains one" to "the as-of line contains one" if it counts
      page-wide.
- [ ] 6.12 Confirm green: `TestTheScriptImpliesNoWinner` and the as-of "does not claim live or
      current" test still pass with a row showing `Live`. If either searches the whole page for
      `live`, scope it to the as-of line.
- [ ] 6.13 Style `.game` to match the meta line. Add a Chrome layout test at 390 CSS px with an
      upcoming game: the meta line lies inside its column and starter rows still line up across the
      columns. If the labelled kickoff cannot fit, record that here and take the fallback in the
      design's risks.

## 7. `web`: the timezone rewrite (D9; `specs/matchup-page`)

Script-text tests read the script through `renderedScript`, as the existing script tests do.

- [ ] 7.1 Red-green: the script selects `time[datetime]` elements and reads each instant from the
      `datetime` attribute.
- [ ] 7.2 Red-green: the script formats with `Intl.DateTimeFormat` and `formatToParts` under the
      `en-US` locale, and names no timezone, so the browser's own is used.
- [ ] 7.3 Red-green: the rewrite of one element is inside a `try`, so a failure leaves that
      element's text alone.
- [ ] 7.4 Confirm green: the script contains no kickoff instant, no fetch instant, and no roster
      name. Add the kickoff and fetch instants to the existing forbidden list.
- [ ] 7.5 Headless-Chrome test, skipped under `-short` or without Chrome: run Chrome with
      `TZ=America/New_York` against a page with a 2026-10-11T17:00:00Z kickoff and a
      2026-10-11T22:07:00Z fetch instant. Expect `Sun 1:00 PM` in the row and
      `Sun, Oct 11 2026 6:07 PM EDT` in the as-of line. If headless Chrome ignores `TZ` on this
      platform, record that here and rely on 7.8.
- [ ] 7.6 Same harness with `TZ=Europe/London` and a 2026-10-13T00:15:00Z kickoff: expect
      `Tue 1:15 AM`.
- [ ] 7.7 Same harness with `TZ=America/Chicago`: each kickoff shows the served weekday and time,
      without the zone label.
- [ ] 7.8 Manual check in a real browser with the system timezone changed: kickoffs and the as-of
      line follow it, and with scripts disabled the Central text stays.

## 8. `cmd/server`: wiring (D4, D6)

- [ ] 8.1 Red-green in `mux_test.go`: `GET /2025/15` renders a game state supplied by a stub games
      source passed to `newMux`.
- [ ] 8.2 In `main.go`, build a second cache, `weekcache.New(client.WeekGames, weekcache.TTL,
      "gamescache")`, from the one `sleeper.Client`, and pass it to `newMux`. Confirm it compiles
      and `go vet ./...` is clean.
- [ ] 8.3 Manual check against a local server on the current week: rows show kickoffs, quarters,
      `Final`, and `Bye` that match Sleeper's own app, and the log shows one `sleeper games` line
      per five minutes with `gamescache hit` lines between.

## 9. Wrap up

- [ ] 9.1 Update the comment on `lineup.Record.Team`: it is display text and the key to the
      player's game, never to the player.
- [ ] 9.2 Run `go vet ./...` and `go test -race ./...`.
- [ ] 9.3 Update `docs/architecture.md` and `docs/package-dependencies.md`: the `games` package,
      the second cache entry, the handler's order, and the script's rewrite.
- [ ] 9.4 Run `openspec validate show-game-status-on-matchup-page`.
