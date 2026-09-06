Each behaviour below is one red-green pair. A RED task is not done until the test has been run and
has failed **for the expected behavioural reason** — a wrong value, not a build error. So every RED
task assumes the symbols it calls already exist, stubbed to a wrong-but-compiling answer by the
preceding task. GREEN tasks make the smallest change that turns that failure into a pass, then
re-run the test.

The provider stand-in throughout is a fake `WeekSource` returning a `score.Week` built with
`score.NewWeek`, as in `internal/api`'s batch tests — no `httptest` server and no Sleeper JSON. The
lineup tree is a `t.TempDir()` written per test, never the real `scripts/lineups`.

`split-score-packages` is a prerequisite: it built `score.Week`, `sleeper.FetchWeek`, and the
consumer-declared `WeekSource` pattern this change consumes.

## 1. The scoring seam

`split-score-packages` delivered `score.Week`, `score.NewWeek`, `sleeper.FetchWeek`, and the
`WeekSource` interface pattern, so the seam itself needs no work here.

- [ ] 1.1 Move the season and week bound constants (`minSeason`, `maxSeason`, `minWeek`, `maxWeek`,
      today unexported in `internal/api`) to where both the batch validator and this page's URL
      parsing can use them — `internal/score` is the package both already depend on. Export a single
      validation helper rather than the four numbers if that reads better. Re-run `go test ./...`;
      confirm the batch endpoint's validation tests still pass unchanged.

## 2. The page's URL

- [ ] 2.1 Create `internal/web`, declaring its own
      `WeekSource interface { Week(ctx context.Context, season, week int) (score.Week, error) }` and
      `Handler(tree *roster.Tree, weeks WeekSource) http.Handler`, stubbed to write `501` and
      nothing else. Confirm it does not import `internal/sleeper`. Register nothing in `main.go` yet. Confirm it builds.
- [ ] 2.2 RED: test, through an `http.ServeMux` registered as `GET /{season}/{week}`, that
      `GET /2025/fifteen` responds `400`. Run; confirm it fails on the `501`.
- [ ] 2.3 GREEN: parse both segments with `strconv.Atoi` and answer `400` when either fails. Re-run;
      confirm pass. Confirm the well-formed case still reaches the stub.
- [ ] 2.4 RED: test that `GET /2025/23` and `GET /1998/3` each respond `400`. Run; confirm they
      currently pass the range check that does not exist yet.
- [ ] 2.5 GREEN: range-check against the shared bounds from 1.1. Re-run; confirm pass.
- [ ] 2.6 RED: test that a `400` request opens no roster file and makes no upstream request —
      point the handler at a lineup tree that does not exist and at a fake `WeekSource` that fails
      the test if called. Run; confirm.
- [ ] 2.7 GREEN: confirm the validation returns before both. Comment why the order matters: a typo
      in a URL must not reach Sleeper.
- [ ] 2.8 RED: test that `POST /2025/15` responds `405`. Run; confirm the mux, not the handler,
      produces it — if it already passes, keep it as the test that pins the method-qualified
      pattern.

## 3. Resolving the week into two columns

- [ ] 3.1 RED: test that a `t.TempDir()` week holding `bojjaes.csv` and `wood.csv` renders a `200`
      whose body contains both team names, with `bojjaes` appearing before `wood`. Run; confirm it
      fails on the `501`.
- [ ] 3.2 GREEN: call `tree.Matchup`, build a two-column view model, and render it through a
      minimal `matchup.html` embedded with `//go:embed` and parsed once with
      `template.Must(template.ParseFS(...))` at package level. Re-run; confirm pass.
- [ ] 3.3 RED: test that a week whose opponent sorts before us (`aardvarks.csv`) still renders
      `bojjaes` first. Run; confirm — if it passes, keep it as the regression test that the column
      order comes from `Matchup` and not from the directory listing.
- [ ] 3.4 RED: test that a week directory that does not exist responds `404`. Run; confirm it is
      currently a `200` or a `500`.
- [ ] 3.5 GREEN: map `roster.ErrNoWeek` with `errors.Is` to `404`. Re-run; confirm pass.
- [ ] 3.6 RED: test that a week holding three rosters responds `500`, and that the body is not a
      scoreboard. Run; confirm.
- [ ] 3.7 GREEN: map the remaining matchup sentinels — too many, too few, not ours — to `500`.
      Re-run; confirm pass. Comment the 404/500 line: a week we never played is the reader asking
      for something that does not exist; a malformed week directory is our tree being wrong about a
      well-formed request.
- [ ] 3.8 RED: test that a week holding two rosters, neither of them ours, responds `500`. Run.
- [ ] 3.9 GREEN: confirm pass.
- [ ] 3.10 RED: test that a week whose `wood.csv` has a line with no id responds `500` and renders
      no page. Run; confirm the parse error is not currently surfaced as a `200`.
- [ ] 3.11 GREEN: refuse an unreadable roster with `500`. Re-run; confirm pass.
- [ ] 3.12 Confirm each of the four refusals logs server-side with the failing path or directory,
      and that the response body carries no filesystem path.

## 4. Scored starters and totals

- [ ] 4.1 RED: test that a week whose rosters hold ids present in the stub payload renders each
      starter's **name** from the CSV alongside its points. Run; confirm the points are absent or
      wrong.
- [ ] 4.2 GREEN: score each starter through `Week.Player` and `Points`, formatting the value into
      the view model. Re-run; confirm pass.
- [ ] 4.3 RED: test that a column of nine known scorers renders the total of their points. Run;
      confirm it fails on the total.
- [ ] 4.4 GREEN: sum the starters into the column's total. Re-run; confirm pass.
- [ ] 4.5 RED: test that a roster of twelve records renders only the first nine, and that the body
      contains none of the last three names. Run; confirm bench players currently appear or the
      total is wrong.
- [ ] 4.6 GREEN: take `Roster.Starters()`. Re-run; confirm pass.
- [ ] 4.7 RED: test that the `WeekSource` is called exactly **once** for a request that renders both
      columns — count calls in the fake. Run; confirm it is currently two.
- [ ] 4.8 GREEN: fetch once and score both columns from the same `Week`. Re-run; confirm pass.
      Comment that the two columns must not be read from different snapshots of the week.
- [ ] 4.9 RED: test that a `WeekSource` returning an error makes the page respond `502` with no
      scoreboard in the body. Run; confirm.
- [ ] 4.10 GREEN: map a fetch failure to `502`. Re-run; confirm pass. Comment that a zeroed page
      would read as "these players scored nothing" rather than "we do not know".

## 5. Starters the provider has nothing for

- [ ] 5.1 RED: test that a starter absent from the payload renders `no stats` and **not** `0`. Run;
      confirm a `0` is currently rendered.
- [ ] 5.2 GREEN: carry the placeholder in the view model, decided in Go rather than in the
      template. Re-run; confirm pass. Comment that `no stats` matches `scripts/scores.sh` while
      both UIs exist.
- [ ] 5.3 RED: test that a column with one absent starter and eight scorers totalling 40 renders a
      total of 40. Run; confirm.
- [ ] 5.4 GREEN: skip absent starters when summing. Re-run; confirm pass.
- [ ] 5.5 RED: test that a starter **present** in the payload with no scoring production renders
      `0` and not `no stats`. Run; confirm this distinguishes absence from a scoreless week.
- [ ] 5.6 GREEN: make it pass; confirm the two cases are decided by `Player`'s second return value
      and nothing else.

## 6. A page that implies no winner

- [ ] 6.1 RED: test that a page whose columns total 56 and 42 contains neither `14` as a rendered
      value nor any element carrying a difference. Run; confirm.
- [ ] 6.2 GREEN: confirm the view model has no margin field and the template computes nothing.
      Re-run.
- [ ] 6.3 Add the `<style>` block: a two-column grid, both columns equal width and sharing one
      class. Confirm by reading it that there is no per-column colour, no `.leading`/`.winner`
      class, no progress bar, and nothing keyed on which total is larger.
- [ ] 6.4 RED: test that the two columns' markup differs only in their content — assert the same
      class appears exactly twice and that no class name in the body matches a leader/winner
      vocabulary. Run; confirm.
- [ ] 6.5 GREEN: make it pass.
- [ ] 6.6 RED: test that a roster record whose name contains `<b>` renders escaped, and that the
      response body contains no `<b>` tag. Run; confirm — if it passes immediately, keep it as the
      regression test that pins contextual escaping and say so in the commit.
- [ ] 6.7 Confirm the response sets an HTML content type, and that the handler renders into a
      `bytes.Buffer` and writes only on success — add a test if the failure path can be provoked,
      otherwise confirm by reading.

## 7. Serving it

- [ ] 7.1 Register `GET /{season}/{week}` in `cmd/server/main.go`, constructing the lineup tree
      from a constant `scripts/lineups` root and handing the handler the same `sleeper.Client` the
      JSON route already gets. Comment that the root is interim until the tree is embedded.
- [ ] 7.2 Confirm `POST /scores` still responds as before — run the existing suite and hit it by
      hand against a locally running server.
- [ ] 7.3 Run the server against the real tree and load `/2025/14`, `/2025/15`, and `/2025/16` in a
      browser. Confirm each renders two columns with the expected opponents (`wood`, `aroma`,
      `gonads`) and that the totals match what `scripts/fantasycast.sh` prints for the same week.
- [ ] 7.4 Update the `main.go` package comment and the README to describe the page route alongside
      `POST /scores`.

## 8. Close out

- [ ] 8.1 Re-read the new code as a whole and delete any comment that restates it; keep the ones
      carrying a rule (the single fetch, why a fetch failure is not a zeroed page, why absence is
      not `0`, why the CSS has no leader class).
- [ ] 8.2 Run `gofmt -l ./...`, `go vet ./...`, and `go test ./...`; confirm clean and that
      `scripts/**` and the roster CSV format are untouched.
- [ ] 8.3 Walk the `matchup-page` spec scenario by scenario and confirm each has a test that
      exercises it; add any the loop above did not produce.
- [ ] 8.4 Tick the backlog's "Add `GET /{season}/{week}`…" line.
