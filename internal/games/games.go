// Package games holds a week's NFL games independent of any provider, so the
// page can show each player's game state without importing the client that
// fetched it.
package games

import "time"

type State int

const (
	Upcoming State = iota
	InProgress
	Complete
	// Other carries a status never seen before rather than dropping it or
	// guessing which known state it means.
	Other
)

type Game struct {
	Home, Away string
	State      State
	Kickoff    time.Time
	Period     string // the provider's quarter text, as sent
	Status     string // the provider's status text, as sent
}

// Week is one season and week's games.
type Week struct {
	season, week int
	games        []Game
}

func NewWeek(season, week int, games []Game) Week {
	return Week{season: season, week: week, games: games}
}

// ForTeam returns the game the team plays in this week. The match is exact,
// case included, because team codes are keys, not display text.
func (w Week) ForTeam(team string) (Game, bool) {
	for _, g := range w.games {
		if g.Home == team || g.Away == team {
			return g, true
		}
	}
	return Game{}, false
}

// Empty reports whether the week holds no games at all, which a caller must
// tell apart from one team having a bye.
func (w Week) Empty() bool {
	return len(w.games) == 0
}
