Each behaviour is one red-green pair. A RED is done only when the test has run and failed **on a
value mismatch or a missing error**, never a build error; the stub sections exist so that holds. A
"Guard" already passes: break the production code on purpose, see it fail, restore. Run the
package's tests after every step.

Out of scope: POST/persist, passphrase, home page, roster transactions, season/week pickers.

## 1. `lineup` stubs (all green)

- [x] 1.1 Add `Slots()` returning `slotOrder`, `func (Slot) Accepts(string) bool` returning
      `false`, exported `OurTeam` (keep `ourTeam` uses pointing at it), and `const LastWeek = 17`.
- [x] 1.2 Add `Rosters`, `NewRosters(fs.FS)`, `Teams()` returning nil, `Read(team)` returning
      `nil, nil`; and `FromSlots([StarterCount]Player) (Lineup, error)` returning `Lineup{}, nil`.
- [x] 1.3 Change `LatestLineup(team)` to `LatestLineup(team string, season, week int)`, ignoring the
      bound; pass `9999, 18` in the existing tests. `go build ./... && go test ./...`: green.

## 2. Slot acceptance

- [x] 2.1 RED: table — RB/FB yes, FLEX/TE yes, D/CB yes, QB/WR no, K/K yes. Fails on the yeses.
- [x] 2.2 GREEN: `Accepts` reads `slotAccepts`. Guard: make `newLineup` call `Accepts` and break
      it; see the slot tests in `record_test.go` fail; restore.

## 3. Rosters

- [x] 3.1 RED: `Teams()` over a MapFS of `wood.csv`, `aroma.csv`, `.DS_Store`, `notes.txt` →
      `[aroma wood]`. GREEN.
- [x] 3.2 RED: `Read("wood")` of a 12-record roster → 12 records in file order. GREEN via `parse`.
- [x] 3.3 RED: `Read("../wood")` → error. GREEN via `checkTeam`. Guard: a roster with an unknown
      position fails, naming the file.
- [x] 3.4 Data test in `internal/data`: `Teams()` over `data.Rosters` is the ten names, and every
      team's `Read` succeeds.

## 4. `FromSlots`

- [x] 4.1 RED: nine eligible records with an RB in RB and an RB in FLEX₁ → `Starters()` equals the
      input order. GREEN.
- [x] 4.2 RED: a WR in QB → error containing `QB`. GREEN via `Accepts`.
- [x] 4.3 RED: one ID in RB and FLEX₁ → error. GREEN. RED: an empty ID in K → error naming `K`.
      GREEN.
- [x] 4.4 Guard: `FromSlots` → `WriteCSV` → MapFS → `Tree.Read` gives the same starters.

## 5. `LatestLineup` bound

- [x] 5.1 RED: weeks 2026/7 and 2026/10, bound 2026/9 → 2026/7 (stub returns 10). GREEN.
- [x] 5.2 Guard: weeks 8 and 9, bound 2026/9 → 9.
- [x] 5.3 RED: 2025/16 and 2026/3, bound 2026/2 → 2025/16. GREEN (later seasons skipped,
      earlier seasons unbounded by week).
- [x] 5.4 RED: only 2026/10, bound 2026/9 → `ErrNoLineup`. GREEN if not already.

## 6. `web` stubs and fixtures (all green)

- [x] 6.1 Declare `CurrentWeekSource interface { AsOf(context.Context) (nflweek.Current, time.Time, error) }`.
- [x] 6.2 Add `lineup.html` (doctype, `<title>`, empty body) parsed at init as `lineupPage`, and
      `LineupHandler(rosters, tree, current, games)` / `LineupCSVHandler(rosters, tree, current)`
      that render the empty template / write nothing with 200.
- [x] 6.3 Test fixtures in `lineup_test.go`: a counting `stubCurrent`, a counting games stub, a
      MapFS roster set (`bojjaes`, `wood`; wood's covers every slot plus a spare RB, FB, WR, CB),
      and a MapFS lineup tree. Helper `get(url) *httptest.ResponseRecorder`.

## 7. Current week

- [x] 7.1 RED: current 2026 regular week 9 → body contains `2026 · Week 9`. GREEN.
- [x] 7.2 RED: offseason → "No week to set", no `<select name="qb"`, games stub calls 0. GREEN.
- [x] 7.3 RED: week 18 → "No week to set". GREEN via `lineup.LastWeek`. Guard: week 17 shows the
      selects.
- [x] 7.4 RED: source error → 502. GREEN.
- [x] 7.5 Guard: `?season=2025&week=3` still shows `2026 · Week 9`.

## 8. Team

- [x] 8.1 RED: `?team=nobody` → 400, current and games calls 0. GREEN. Guard: `?team=../wood` → 400.
- [x] 8.2 RED: no `team` → picker lists every roster team and marks `bojjaes` selected. GREEN.
- [x] 8.3 RED: `?team=wood` → `wood` selected. GREEN. Assert the picker form is `method="get"`
      `action="/lineup"`.

## 9. Slot selects and options

- [x] 9.1 RED: nine selects named `qb rb wr te flex flex k d d` in order. GREEN.
- [x] 9.2 RED: QB select holds the QB's ID and not a WR's. GREEN via `Accepts`.
- [x] 9.3 Guard: FLEX holds RB, FB, WR, TE, not QB or CB; D holds CB and DE.
- [x] 9.4 RED: games stub with DET upcoming Sun 12:00 Central → option text
      `Amon-Ra St. Brown · WR · DET · Sun 12p`. GREEN via `gameState` + `kickoffText`.
- [x] 9.5 Guard: no game in a non-empty week → ends `· Bye`; FA → ends `· FA`; games error → 200,
      option ends at the team.

## 10. Starting lineup

- [x] 10.1 RED: wood has 2026/9 → each select's `selected` option is that file's player. GREEN.
- [x] 10.2 RED: wood has 2026/7 and 2026/10, current week 9 → week 7's players selected. GREEN
      via the bounded `LatestLineup`.
- [x] 10.3 RED: no lineup → every select's empty option selected. GREEN.
- [x] 10.4 RED: the starting TE is not on wood's roster → he is offered and selected in TE. GREEN
      (pool = roster then off-roster starters).
- [x] 10.5 RED: starting file malformed → 200, empty slots, notice text. GREEN.

## 11. Proposal

- [x] 11.1 RED: a full valid proposal URL → each select shows its proposed ID selected (differs
      from the starting lineup), and no error list. GREEN.
- [x] 11.2 RED: `flex=A&flex=B` → FLEX₁ A, FLEX₂ B. GREEN-or-guard (swap order to confirm).
- [x] 11.3 RED: `qb=99999` → 200, error names `99999`, QB empty. GREEN.
- [x] 11.4 RED: WR as `qb` → error "cannot play QB", WR selected in QB. GREEN.
- [x] 11.5 RED: same RB as `rb` and `flex` → "in two slots". GREEN.
- [x] 11.6 RED: no `k` → "K slot is empty". GREEN. RED: three `flex` → "too many FLEX". GREEN.
- [x] 11.7 Guard: `qb=<script>` → 200, escaped, unknown.
- [x] 11.8 RED: slots form is `method="get" action="/lineup"` with hidden `team=wood` and a
      "Check" button. GREEN.

## 12. Summary and CSV

- [x] 12.1 RED: valid proposal → textarea text exactly `wood · 2026 week 9` then nine
      `<SLOT> <name> (<team>)` lines. GREEN via `FromSlots`.
- [x] 12.2 Guard: duplicate proposal → no textarea and no `/lineup.csv` link; starting lineup with
      an empty slot → none either.
- [x] 12.3 RED: valid page links `/lineup.csv?team=wood&qb=…&…&d=…` in slot order and names
      `internal/data/lineups/2026/9/wood.csv`. GREEN.
- [x] 12.4 RED: `LineupCSVHandler` valid → 200, `text/csv; charset=utf-8`,
      `attachment; filename="wood.csv"`, body equal to `FromSlots(...).WriteCSV`. GREEN, sharing the
      team/week/pool/proposal steps with the page handler.
- [x] 12.5 RED: CSV with a duplicate → 400; no week to set → 404; unknown team → 400; source error
      → 502. GREEN each.
- [x] 12.6 Guard: a roster name `<b>X</b>` renders escaped in options and summary.

## 13. Mux and wiring

- [x] 13.1 Add `stubCurrent` to `mux_test.go` and the two new `newMux` parameters (handlers not yet
      registered). Green.
- [x] 13.2 RED: `GET /lineup` → 200 with `name="qb"` (today the season handler's 400). GREEN:
      register `GET /lineup`.
- [x] 13.3 RED: `GET /lineup.csv?…valid` → `text/csv`. GREEN: register `GET /lineup.csv`.
- [x] 13.4 Guard: `GET /2025` still the season page; `POST /lineup` → 405.
- [x] 13.5 `main`: set `V1BaseURL`, build `weekcache.NewSingle("nflstate", client.CurrentWeek,
      weekcache.CurrentWeekTTL)`, pass it and `lineup.NewRosters(data.Rosters)` to `newMux`.
      `go build ./... && go test ./...`.
- [x] 13.6 Run the server and load `/lineup`, a team change, a Check, and the CSV on a phone-width
      viewport; confirm the layout matches the matchup page's style.
- [x] 13.7 Tick the backlog row.
