package roster

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Tree locates roster files within a lineup tree rooted at a caller-supplied
// directory. The layout is <root>/<season>/<week>/<team>.csv, stated here and
// nowhere else, so callers never construct the path themselves.
type Tree struct {
	root string
}

// New returns a Tree rooted at root.
func New(root string) *Tree {
	return &Tree{root: root}
}

// Path resolves the roster file for a season, week, and team.
func (t *Tree) Path(season, week int, team string) (string, error) {
	// A team name arrives as a URL segment once the page exists; refusing
	// anything but a single, plain segment here keeps it from reaching
	// outside the lineup tree, in the one place that knows the layout.
	if team != filepath.Base(team) || strings.Contains(team, "..") {
		return "", fmt.Errorf("roster: invalid team name %q", team)
	}

	return filepath.Join(t.root, fmt.Sprint(season), fmt.Sprint(week), team+".csv"), nil
}
