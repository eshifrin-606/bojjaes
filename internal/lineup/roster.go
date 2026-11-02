package lineup

import (
	"fmt"
	"io/fs"
	"strings"
)

// Rosters locates each team's roster file, <team>.csv, within a filesystem
// already rooted at the roster directory.
type Rosters struct {
	fsys fs.FS
}

func NewRosters(fsys fs.FS) *Rosters {
	return &Rosters{fsys: fsys}
}

// Teams lists the roster files in name order, which fs.ReadDir guarantees.
// Dotfiles are skipped for the reason Matchup skips them.
func (r *Rosters) Teams() []string {
	entries, _ := fs.ReadDir(r.fsys, ".")
	var teams []string
	for _, e := range entries {
		if !e.Type().IsRegular() || strings.HasPrefix(e.Name(), ".") || !strings.HasSuffix(e.Name(), ".csv") {
			continue
		}
		teams = append(teams, strings.TrimSuffix(e.Name(), ".csv"))
	}
	return teams
}

func (r *Rosters) Read(team string) ([]Player, error) {
	if err := checkTeam(team); err != nil {
		return nil, err
	}
	name := team + ".csv"
	f, err := r.fsys.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	players, err := parse(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return players, nil
}
