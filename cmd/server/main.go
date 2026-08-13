// Command server is a walking skeleton: GET /score fetches Puka Nacua's 2025
// week 14 line from Sleeper, applies the rushing/receiving slice of the HMFFL
// rules, and returns it as JSON. Turning the hardcoded player, season, and week
// into request parameters is the next change's job.
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/eshifrin/bojjaes/internal/score"
)

const addr = ":8080"

func main() {
	fetch := func(ctx context.Context) (score.StatLine, error) {
		return score.FetchStatLine(
			ctx,
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
