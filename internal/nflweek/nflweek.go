// Package nflweek is the current point in the NFL calendar, independent of any
// provider, so a caller can ask which week a lineup is set for without
// importing the client that read it.
package nflweek

type Phase int

const (
	Preseason Phase = iota
	Regular
	Postseason
	Offseason
	// Other carries a phase never seen before rather than refusing it or
	// guessing which known phase it means.
	Other
)

// Current is the provider's season, week, and phase as last read.
type Current struct {
	season, week int
	phase        Phase
}

func New(season, week int, phase Phase) Current {
	return Current{season: season, week: week, phase: phase}
}

// Settable reports the season and week a lineup can be set for. The league's
// own last week is a league rule and is not applied here.
func (c Current) Settable() (season, week int, ok bool) {
	if c.phase != Regular || c.week < 1 {
		return 0, 0, false
	}
	return c.season, c.week, true
}
