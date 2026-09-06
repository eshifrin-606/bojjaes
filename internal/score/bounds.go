package score

import "fmt"

// The season and week a request may ask about. They live in the domain rather
// than in one transport because every way into the scoring path — the JSON
// endpoint, the page's URL — has to refuse the same nonsense, and two copies
// of the bounds would eventually disagree.
const (
	minSeason = 2009 // Sleeper's stats do not reach further back.
	maxSeason = 2099
	minWeek   = 1
	maxWeek   = 18
)

// ValidateSeasonWeek reports whether a season and week are ones the league
// could have played.
func ValidateSeasonWeek(season, week int) error {
	if season < minSeason || season > maxSeason {
		return fmt.Errorf("season %d outside %d-%d", season, minSeason, maxSeason)
	}
	if week < minWeek || week > maxWeek {
		return fmt.Errorf("week %d outside %d-%d", week, minWeek, maxWeek)
	}
	return nil
}
