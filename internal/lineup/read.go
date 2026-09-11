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

	l := Lineup{records: records}
	// Starters and bench are resolved separately so a bench player sharing a
	// short name never pushes a starter back to their full name.
	fillShortNames(l.Starters())
	fillShortNames(l.Bench())
	return l, nil
}
