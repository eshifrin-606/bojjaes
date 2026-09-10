package lineup

import (
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strings"
)

// Tree locates lineup files within a filesystem already rooted at the lineup
// tree. The layout is <season>/<week>/<team>.csv, stated here and nowhere else,
// so callers never construct the name themselves.
type Tree struct {
	fsys fs.FS
}

// New returns a Tree over a filesystem already rooted at the lineup tree.
func New(fsys fs.FS) *Tree {
	return &Tree{fsys: fsys}
}

// Path resolves the lineup file for a season, week, and team.
func (t *Tree) Path(season, week int, team string) (string, error) {
	// A team name arrives as a URL segment once the page exists; refusing
	// anything but a single, plain segment here keeps it from reaching
	// outside the lineup tree, in the one place that knows the layout.
	if team != filepath.Base(team) || strings.Contains(team, "..") {
		return "", fmt.Errorf("lineup: invalid team name %q", team)
	}

	return path.Join(t.weekDir(season, week), team+".csv"), nil
}
