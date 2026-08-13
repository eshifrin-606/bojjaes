// Command server scores one hardcoded player for one hardcoded week.
//
// It is a walking skeleton: hitting GET /score fetches Puka Nacua's 2025 week
// 14 line from Sleeper, applies the rushing/receiving slice of the HMFFL rules,
// prints the result, and returns it as JSON. Player, season, and week are
// constants — parameterizing them is the next change's job.
package main

import (
	"log"
	"net/http"

	"github.com/eshifrin/bojjaes/internal/score"
)

const addr = ":8080"

func main() {
	fetch := func() (score.StatLine, error) {
		return score.FetchStatLine(
			score.SleeperBaseURL,
			score.TargetSeason,
			score.TargetWeek,
			score.NacuaPlayerID,
		)
	}

	http.Handle("/score", score.Handler(fetch))

	log.Printf("listening on %s; GET http://localhost%s/score", addr, addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
