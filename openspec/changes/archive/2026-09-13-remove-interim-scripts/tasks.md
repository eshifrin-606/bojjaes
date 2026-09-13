Removal work. The red step is a failing test only where the spec states new observable behaviour
(the 404 on `/scores`). Everything else is deleting code together with its tests, then confirming
`go build ./... && go test ./...` stays green. `scripts/*.sh` are exempt from TDD and are simply
deleted.

## 1. Mux: one route

- [x] 1.1 Add `TestMuxRejectsTheWrongMethodOnTheMatchupRoute`: `POST /2025/15` answers 405 and the
      stub handler is not reached. This already passes (a characterization test); record that
      rather than faking a red step.
- [x] 1.2 Add a test that `POST /scores` answers 404. Run it: red (it answers 400 today, from
      `BatchHandler` rejecting the empty body).
- [x] 1.3 Remove the `POST /scores` route from `newMux`, change its parameter to `web.StatsSource`,
      and delete the combined `statsSource` interface and the `api` import. Run 1.2: green.
- [x] 1.4 Delete `TestMuxRejectsTheWrongMethodOnScores` and `TestMuxAnswersTheWrongMethodItself`,
      and drop `stubStats.WeekStats`. Full `cmd/server` tests green.

## 2. Remove `internal/api`

- [x] 2.1 Delete the `_ api.StatsSource = (*Cache)(nil)` assertion and the `api` import from
      `internal/statscache/cache_test.go`.
- [x] 2.2 Delete `internal/api/` (batch.go, dto.go, and their tests). `go build ./... && go test
      ./...` green; `grep -rn internal/api --include='*.go' .` empty.

## 3. Remove `Cache.WeekStats`

- [x] 3.1 In `cache_test.go`, rewrite every `cache.WeekStats(...)` call to `cache.WeekStatsAsOf(...)`,
      discarding the instant. Run the statscache tests with `WeekStats` still present: green, and
      the same test count as before.
- [x] 3.2 Delete `Cache.WeekStats` and its doc comment. Build and full test suite green.
- [x] 3.3 Confirm `internal/sleeper/sleeper_test.go` still pins the `/v1/stats/nfl/regular/` path,
      which covers the new "Weekly stats are regular-season stats" requirement. No code change
      expected.

## 4. Remove the scripts

- [x] 4.1 Delete `scripts/scores.sh`, `scripts/fantasycast.sh`, and `scripts/teams/`. The `scripts/`
      directory should be gone afterwards.
- [x] 4.2 `grep -rnI "scores\.sh\|fantasycast\.sh\|scripts/\|POST /scores\|internal/api"` outside
      `openspec/changes/` lists only the docs handled in section 5.

## 5. Docs, amended in place

- [x] 5.1 README: remove the `POST /scores` curl examples and the `no_stats` explanation that goes
      with them, plus "Scoring a lineup" and "A matchup, side by side". Replace the smoke test with
      running the server and opening `localhost:8080/2025/14`. Keep any caveat the page still needs
      (a starter with no stats is not a zero), stated in terms of the page.
- [x] 5.2 `docs/architecture.md`: delete §5 "The other endpoint", and renumber or fix
      cross-references if any follow.
- [x] 5.3 `docs/package-dependencies.md`: remove the `internal/api` section and any graph edges to
      it. Reword `internal/lineup`'s intro so it no longer calls itself "the Go home for what
      `scripts/scores.sh` knows in bash" or mentions "a question no script asks".
- [x] 5.4 ADR 0004: put the scripts-as-context lines in past tense. In decision 2, drop "`scripts/…`
      keep working" and the `POST /scores`/`GET /score` endpoint list. In decision 9, drop "exactly
      as in `scripts/scores.sh`" and "which `scripts/scores.sh` and ...". Attribute the no-margin
      rationale to the page. Remove the consequence "The CSV format change touches
      `scripts/scores.sh`", and make the field-order follow-up name only the lineup parser.
- [x] 5.5 `backlog.md`: in the lineup-CSV item, drop "and update `scores.sh`" and "the parser and
      the script both read it".

## 6. Spec purposes and sync

- [x] 6.1 After syncing the delta specs, edit the purposes by hand. `player-week-score`: drop
      "through to an HTTP response". `matchup-page`: drop the `matchup-report` sentence, and change
      "the scoring endpoints" in the range rule to something that exists. `lineup-source`: drop "belongs
      to `lineup-score-report`". `http-server-lifecycle`: drop the "scores are `player-week-score`'s"
      clause.
- [x] 6.2 Delete `openspec/specs/lineup-score-report/` and `openspec/specs/matchup-report/` once
      their requirements are all removed. `openspec validate --specs` passes.

## 7. Memory and wrap-up

- [x] 7.1 Delete the `shell-scripts-are-interim-ui` auto-memory and its `MEMORY.md` line, and fix any
      `[[shell-scripts-are-interim-ui]]` links in other memories.
- [x] 7.2 Final check: `go vet ./... && go test ./...` green, and the Docker image still builds
      (`.dockerignore` and the `Dockerfile` do not reference `scripts/`).
