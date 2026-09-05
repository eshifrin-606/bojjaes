package roster

import (
	"fmt"
	"os"
)

// Read locates and parses the roster for a season, week, and team.
func (t *Tree) Read(season int, week int, team string) (Roster, error) {
	path, err := t.Path(season, week, team)
	if err != nil {
		return Roster{}, err
	}

	f, err := os.Open(path)
	if err != nil {
		return Roster{}, err
	}
	defer f.Close()

	records, err := parse(f)
	if err != nil {
		return Roster{}, fmt.Errorf("%s: %w", path, err)
	}

	return Roster{records: records}, nil
}
