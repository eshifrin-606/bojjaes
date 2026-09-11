package lineup

import "fmt"

// Read locates and parses the lineup for a season, week, and team.
func (t *Tree) Read(season int, week int, team string) (Lineup, error) {
	path, err := t.Path(season, week, team)
	if err != nil {
		return Lineup{}, err
	}

	f, err := t.fsys.Open(path)
	if err != nil {
		return Lineup{}, err
	}
	defer f.Close()

	records, err := parse(f)
	if err != nil {
		return Lineup{}, fmt.Errorf("%s: %w", path, err)
	}

	return newLineup(records), nil
}
