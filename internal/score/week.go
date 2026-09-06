package score

// Week is one season and week's stat lines, keyed by player ID, in
// provider-neutral terms. It is built by an adapter and read by anyone: the
// provider's payload shape ends at construction.
type Week struct {
	season, week int
	players      map[string]StatLine
}

// NewWeek is exported because the adapter that builds a Week lives in another
// package; it is the only way to fill one. The caller must not retain players
// after the call — a Week is read concurrently, so its map is never written
// again once construction returns.
func NewWeek(season, week int, players map[string]StatLine) Week {
	return Week{season: season, week: week, players: players}
}

// Player reads one player's stat line out of the week.
//
// Absence is reported as a value rather than an error for the same reason the
// transform reports it that way: the payload cannot say whether a missing
// player has not kicked off, was inactive, or does not exist. Only the caller
// can decide what that means for its request.
func (w Week) Player(id string) (StatLine, bool) {
	line, ok := w.players[id]
	if !ok {
		return StatLine{}, false
	}

	// Stamped on the way out rather than trusted from the map, so a line
	// cannot be attributed to a week other than the one this Week holds.
	line.Season, line.Week = w.season, w.week
	return line, true
}
