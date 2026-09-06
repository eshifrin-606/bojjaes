Most of this change is movement, not new behaviour, so the red-green loop applies only where
something genuinely new is being written: `score.Week`, the eager transform, and the consumer-side
interface. Those tasks are marked RED/GREEN and follow the usual rule — a RED task is not done until
the test has been run and has failed **for the expected behavioural reason**, a wrong value rather
than a build error, so every RED assumes the symbols it calls exist, stubbed wrong-but-compiling by
the task before it.

Every other task is a move. Its tests move with it, unedited apart from the package clause. **A test
that has to change to compile is a signal the split cut through something** — stop and look rather
than patching it.

`go build ./...` and `go test ./...` are expected green at the end of every numbered group.

## 1. A baseline that survives the refactor

- [x] 1.1 With the current binary running, capture `scripts/scores.sh` and `scripts/fantasycast.sh`
      output for weeks 14, 15, and 16 into files outside the repo. These are the external pin on
      `POST /scores`: the suite moves with the code, so a green suite alone is weaker evidence than
      usual here.
- [x] 1.2 Record the current `go test ./...` pass count and `wc -l internal/score/*.go`, so the
      split can be shown to have moved tests rather than lost them.

## 2. `score.Week`, in place

- [x] 2.1 Add `week.go` to `internal/score` with `Week` holding a season, a week, and a
      `map[string]StatLine`, plus `Player(id string) (StatLine, bool)` stubbed to always report
      `false`, and an unexported constructor the adapter will use. Confirm `go build ./...` passes.
- [x] 2.2 RED: test that a `Week` built over one `StatLine` returns it from `Player`, found. Run;
      confirm it fails on the `false`, not on a build error.
- [x] 2.3 GREEN: look the id up in the map. Re-run; confirm pass.
- [x] 2.4 RED: test that `Player` on an id the `Week` does not hold reports `false` and a zero
      `StatLine` — not a zeroed line reported as found. Run; confirm.
- [x] 2.5 GREEN: make it pass. Comment why absence is a value and not an error, referring to the
      reason rather than restating the code.
- [x] 2.6 RED: test that a stat line read from a `Week` carries the season and week the `Week` was
      built for, even when the stat line put into it carried different ones. Run; confirm it fails
      on the wrong season. This pins that a line cannot be attributed to another week.
- [x] 2.7 GREEN: stamp season and week on read, or refuse them on construction — whichever reads
      better — and confirm pass.
- [x] 2.8 Confirm `Week`'s exported surface mentions no provider type and no stat key, and that
      nothing about it can be mutated after construction.

## 3. The adapter moves out

- [x] 3.1 Create `internal/sleeper`. Move `sleeper.go`, `sleeper_test.go`, and `testdata/week14.json`
      into it unchanged apart from the package clause, exporting `BaseURL` and keeping `fetchWeekly`
      and `statLineFrom` unexported. `internal/score` will not build until 3.4 — that is expected.
- [x] 3.2 Add `sleeper.FetchWeek(ctx, baseURL string, season, week int) (score.Week, error)`,
      stubbed to return an empty `Week` and a nil error. Confirm `internal/sleeper` builds.
- [x] 3.3 RED: test that `FetchWeek` against a stub server serving `testdata/week14.json` returns a
      `Week` whose `Player("9493")` is found with the receiving yards the existing transform tests
      assert. Run; confirm it fails on the empty `Week`.
- [x] 3.4 GREEN: fetch once, then transform **every** entry in the payload into the map. Re-run;
      confirm pass. Comment that mapping the whole payload is what lets the raw map's lifetime end
      in this function.
- [x] 3.5 RED: test that a payload entry of JSON `null` yields a `Week` that reports that player
      absent rather than holding a zeroed line for them. Run; confirm.
- [x] 3.6 GREEN: skip entries the transform rejects when building the map. Re-run; confirm pass.
- [x] 3.7 RED: test that an empty payload — an unplayed week — yields a `Week` with no players and
      no error. Run; confirm.
- [x] 3.8 GREEN: confirm pass; an unplayed week is a legitimate answer, not a failure.
- [x] 3.9 RED: test that a stub server returning a non-200 makes `FetchWeek` return an error naming
      the season and week, and that a `Week` is not returned alongside it. Run; confirm.
- [x] 3.10 GREEN: confirm pass.
- [x] 3.11 Move the settled-week smoke assertion into this package's tests: `9493` in 2025 week 14
      transforms to the known line and scores the known total. This is what `GET /score` was
      actually for.
- [x] 3.12 Confirm by grep that no vendor stat key string appears outside `internal/sleeper`, and
      that no exported symbol of `internal/sleeper` returns a `map[string]map[string]float64`.

## 4. Deleting the walking skeleton

- [x] 4.1 Delete `Handler`, `fetchTarget`, `ScoreResponse`'s single-player-only usage if it has one,
      and the `NacuaPlayerID` / `TargetSeason` / `TargetWeek` constants, along with the parts of
      `handler_test.go` that test them. Keep whatever `POST /scores` shares.
- [x] 4.2 Remove the `GET /score` route from `cmd/server/main.go` and its mention from the package
      comment. Confirm `go build ./...` and `go test ./...` are green.
- [x] 4.3 Confirm by grep that nothing in `scripts/**` or `README.md` still calls `GET /score`;
      update the README to point at the equivalent `POST /scores` request.

## 5. The JSON layer moves out

- [x] 5.1 Create `internal/api`. Move the remaining `handler.go`, `batch.go`, and their tests into
      it, unchanged apart from the package clause and the `score.` qualifier now needed on `StatLine`
      and `Points`. Confirm the suite is green and that no test's assertions changed.
- [x] 5.2 Declare the consumer-side interface in `internal/api`:
      `type WeekSource interface { Week(ctx context.Context, season, week int) (score.Week, error) }`,
      and change `BatchHandler` to take one instead of a `baseURL` string. Confirm it builds.
- [x] 5.3 RED: rewrite one existing batch test to drive the handler through a hand-built fake
      `WeekSource` returning a `Week` constructed in the test — no `httptest` server, no Sleeper
      JSON. Run; confirm it fails on whatever the fake returns rather than on a build error, then
      make it pass.
- [x] 5.4 Convert the remaining batch tests that only needed a stub server for its payload. Leave
      any test that is genuinely about transport — a 502 on upstream failure — driving a fake that
      returns an error. Confirm the suite is green and that the endpoint's behaviour is asserted no
      less than before.
- [x] 5.5 Confirm `internal/api` does not import `internal/sleeper`, and that `internal/score`
      imports neither of them.
- [x] 5.6 Mark `POST /scores` interim where a reader will meet it — a comment on the handler and a
      line in the README — naming what it is for (`scripts/scores.sh`, `scripts/fantasycast.sh`) and
      that it retires with them.

## 6. `cmd/server` as the composition root

- [x] 6.1 Add the `sleeper`-backed `WeekSource` implementation — a small type holding the base URL
      whose `Week` method calls `sleeper.FetchWeek`. Decide whether it lives in `sleeper` or in
      `main`, and record which in a comment; it is the only place the two sides meet.
- [x] 6.2 Wire it in `main.go`, register the single remaining route, and update the package comment
      to describe the endpoint that now exists. Confirm the server starts.
- [x] 6.3 Confirm `cmd/server` is the only package importing `internal/sleeper`, by grep.

## 7. Pinning the eager transform's cost

- [x] 7.1 Add a benchmark over `FetchWeek`'s transform against the real `week14.json` payload,
      isolating the mapping from the fetch. Run it and record the number in the change's design
      notes.
- [x] 7.2 Confirm the number is small next to a JSON decode of the same payload — benchmark the
      decode alone if the comparison is not obvious — so the eager decision rests on a measurement
      rather than on the argument for it.

## 8. Close out

- [x] 8.1 Re-run `scripts/scores.sh` and `scripts/fantasycast.sh` for weeks 14, 15, and 16 against
      the rebuilt binary and diff against 1.1. Confirm the output is identical, byte for byte.
- [x] 8.2 Run `gofmt -l ./...`, `go vet ./...`, and `go test ./...`; confirm clean, and compare the
      test count against 1.2 to confirm no test was dropped in a move.
- [x] 8.3 Re-read each new package's doc comment and confirm it says what the package is for and
      what it may not depend on. Delete any comment left behind that describes the old arrangement.
- [x] 8.4 Walk the `player-week-score` delta and confirm the snapshot requirement's scenarios each
      have a test, and that nothing still implements the removed endpoint requirement.
- [x] 8.5 Update `openspec/changes/serve-matchup-page` — its design's `score.FetchWeek` seam becomes
      `sleeper.FetchWeek` returning `score.Week`, `internal/web` declares its own `WeekSource`, and
      its tasks drop the tasks this change already did.
- [x] 8.6 Tick the backlog's "Drop the walking-skeleton `fmt.Printf` out of the `/score` handler"
      line, and add a line recording the package split if the backlog does not already imply it.
