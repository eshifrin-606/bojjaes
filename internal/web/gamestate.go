package web

import (
	"strings"
	"time"

	"github.com/eshifrin/bojjaes/internal/games"
)

// gameState is what a starter's line says about the game their roster team
// plays. Kickoff is returned apart from the text so the template can put it in
// a <time> element; a zero kickoff and empty text mean the line shows nothing.
func gameState(team string, week games.Week, available bool) (text string, kickoff time.Time) {
	// A week with no games at all is missing data, not a league-wide bye.
	// A free agent has no team, so no game, and is not on a bye either.
	if !available || week.Empty() || team == "FA" {
		return "", time.Time{}
	}
	g, ok := week.ForTeam(team)
	if !ok {
		return "Bye", time.Time{}
	}
	switch g.State {
	case games.Upcoming:
		return "", g.Kickoff
	case games.InProgress:
		if g.Period == "" {
			return "Live", time.Time{}
		}
		if allDigits(g.Period) {
			return "Q" + g.Period, time.Time{}
		}
		return g.Period, time.Time{}
	case games.Complete:
		if g.Period == "" {
			return "Final", time.Time{}
		}
		if rest, ok := strings.CutPrefix(g.Period, "F"); ok {
			return "Final" + rest, time.Time{}
		}
		return g.Period, time.Time{}
	case games.Other:
		return g.Status, time.Time{}
	}
	return "", time.Time{}
}

func allDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
