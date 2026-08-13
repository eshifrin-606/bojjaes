## 1. Setup

- [x] 1.1 `go mod init github.com/eshifrin/bojjaes`; confirm `go version` and that `go test ./...` runs clean on an empty tree
- [x] 1.2 Look up Puka Nacua's Sleeper `player_id` and record it, with season 2025 / week 14, as consts in `internal/score/sleeper.go`
- [x] 1.3 Fetch `https://api.sleeper.app/v1/stats/nfl/regular/2025/14` once by hand and save Nacua's entry as `internal/score/testdata/week14.json`, trimmed to two players (Nacua plus one other, so "wrong player" is a possible test failure)
- [x] 1.4 Eyeball that entry against a public box score to confirm the ID is really Nacua

## 2. Domain object

- [x] 2.1 Write `StatLine` in `internal/score/stats.go`: PlayerID, Season, Week, RushYd, RecYd, RushTD, RecTD, TD40Plus, FumLost

## 3. Calculator (red-green)

- [x] 3.1 RED: table test in `calc_test.go` for the below-threshold case (60 rec / 20 rush → 0); watch it fail to compile, add `Points(StatLine) float64` returning 0, green
- [x] 3.2 RED: 99 rec yards → 3.5; implement the receiving clause with floored 10-yard increments, green
- [x] 3.3 RED: 80 rush + 80 rec → 6.0 (single award, combined clause wins); generalize to max-over-three-clauses, green
- [x] 3.4 RED: 105 rec / 0 rush → 4.0 (receiving clause beats combined); confirm green without further change, or fix the max
- [x] 3.5 RED: 2 rec TD with one 40+ → +13; implement TD and long-TD bonus, green
- [x] 3.6 RED: 85 rec + 1 fumble lost → 0 (3-pt award, no increment, minus 3); implement the -3, green

## 4. Sleeper transform (red-green)

- [x] 4.1 RED: test that `FetchStatLine(baseURL, season, week, playerID)` against an `httptest.Server` serving `week14.json` returns Nacua's mapped stats; implement decode into `map[string]map[string]float64` and map the six stats, green
- [x] 4.2 RED: test that a player ID absent from the payload returns an error naming ID, season, and week; implement, green
- [x] 4.3 Confirm a stat key missing from the fixture entry maps to 0 rather than erroring (add the case to 4.1's table)

## 5. Server (red-green)

- [x] 5.1 RED: test the handler via `httptest.NewRecorder` against a stubbed fetch, asserting 200 and a JSON body carrying both stats and points; implement the handler and the `{stats, points}` response struct, green
- [x] 5.2 RED: test that a fetch error yields 5xx and a message rather than a zero score; implement, green
- [x] 5.3 Add the `fmt.Println` of the score in the handler path
- [x] 5.4 Write `cmd/server/main.go`: wire the real Sleeper base URL, listen on `:8080`, log the listen address

## 6. Verify

- [x] 6.1 `go test ./...` all green; `go vet ./...` clean
- [x] 6.2 Run the server, hit the endpoint, confirm a real score prints to stdout and returns as JSON
- [x] 6.3 Hand-check the returned stat line against a box score and sanity-check the point total by the rules in `docs/scoring.md`
