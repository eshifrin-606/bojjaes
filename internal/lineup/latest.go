package lineup

import (
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strconv"
)

// ErrNoLineup means no week in the tree holds a lineup for the team.
var ErrNoLineup = errors.New("no lineup")

// LatestLineup finds the newest season and week that hold a lineup for the
// team: the lineup a team is assumed to start until it files another.
func (t *Tree) LatestLineup(team string) (season, week int, err error) {
	if err := checkTeam(team); err != nil {
		return 0, 0, err
	}
	for _, s := range t.numberedDirsNewestFirst(".") {
		for _, w := range t.numberedDirsNewestFirst(strconv.Itoa(s)) {
			p, _ := t.Path(s, w, team) // team was checked above
			if _, err := fs.Stat(t.fsys, p); err == nil {
				return s, w, nil
			}
		}
	}
	return 0, 0, fmt.Errorf("%s: %w", team, ErrNoLineup)
}

func (t *Tree) numberedDirsNewestFirst(dir string) []int {
	entries, _ := fs.ReadDir(t.fsys, dir)
	var nums []int
	for _, e := range entries {
		if n, err := strconv.Atoi(e.Name()); err == nil && e.IsDir() {
			nums = append(nums, n)
		}
	}
	slices.Sort(nums)
	slices.Reverse(nums)
	return nums
}
