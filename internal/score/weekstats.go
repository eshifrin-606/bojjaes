package score

// WeekStats is one season and week's stat lines, keyed by player ID, in
// provider-neutral terms. It is built by an adapter and read by anyone: the
// provider's payload shape ends at construction.
type WeekStats struct {
	season, week int
	players      map[string]StatLine
}

// NewWeekStats is exported because the adapter that builds a WeekStats lives
// in another package; it is the only way to fill one. The caller must not
// retain players after the call — a WeekStats is read concurrently, so its map
// is never written again once construction returns.
func NewWeekStats(season, week int, players map[string]StatLine) WeekStats {
	return WeekStats{season: season, week: week, players: players}
}

// Player reads one player's stat line out of the week.
//
// Absence is reported as a value rather than an error for the same reason the
// transform reports it that way: the payload cannot say whether a missing
// player has not kicked off, was inactive, or does not exist. Only the caller
// can decide what that means for its request.
func (w WeekStats) Player(id string) (StatLine, bool) {
	line, ok := w.players[id]
	if !ok {
		return StatLine{}, false
	}

	// Stamped on the way out rather than trusted from the map, so a line
	// cannot be attributed to a week other than the one this WeekStats holds.
	line.Season, line.Week = w.season, w.week
	return line, true
}
