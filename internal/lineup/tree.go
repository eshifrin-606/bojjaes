// Package lineup reads weekly lineup files: the CSV format of one lineup, and
// the <season>/<week>/<team>.csv tree that locates them.
package lineup

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strconv"
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

// file resolves the lineup file for a season, week, and team.
func (t *Tree) file(season, week int, team string) (string, error) {
	if err := checkTeam(team); err != nil {
		return "", err
	}

	return path.Join(t.weekDir(season, week), team+".csv"), nil
}

// A team name arrives as a URL segment once the page exists; refusing
// anything but a single, plain segment keeps it from reaching outside the
// lineup tree.
func checkTeam(team string) error {
	if team != filepath.Base(team) || strings.Contains(team, "..") {
		return fmt.Errorf("lineup: invalid team name %q", team)
	}
	return nil
}

// Read locates and parses the lineup for a season, week, and team.
func (t *Tree) Read(season int, week int, team string) (Lineup, error) {
	path, err := t.file(season, week, team)
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

	lineup, err := newLineup(records)
	if err != nil {
		return Lineup{}, fmt.Errorf("%s: %w", path, err)
	}
	return lineup, nil
}

// ourTeam is the Bojjaes' lineup file name, without the extension. This repo
// is a tool for one team, so the name lives here rather than in every caller.
const ourTeam = "bojjaes"

// The four ways a week directory can fail to be a matchup. They are separate
// values because they are four different mistakes with four different fixes,
// and a caller that renders one needs to tell them apart.
var (
	ErrNoWeek         = errors.New("no week directory")
	ErrTooFewLineups  = errors.New("fewer than two lineups")
	ErrTooManyLineups = errors.New("more than two lineups")
	ErrNotOurMatchup  = errors.New("no bojjaes lineup")
)

func found(names []string) string {
	if len(names) == 0 {
		return "no lineup files"
	}
	return strings.Join(names, ", ")
}

// weekDir is where file gets the directory it puts a lineup file in, so the
// week and the files inside it cannot disagree about where the tree is.
func (t *Tree) weekDir(season, week int) string {
	return path.Join(fmt.Sprint(season), fmt.Sprint(week))
}

// HasWeek reports whether the tree holds a directory for the week. It does not
// ask whether that directory is a matchup: a broken week still exists, and
// hiding it would hide the mistake.
func (t *Tree) HasWeek(season, week int) bool {
	info, err := fs.Stat(t.fsys, t.weekDir(season, week))
	return err == nil && info.IsDir()
}

// LatestWeek reports the greatest week number the tree holds a directory for,
// in a season. It reads only the season directory's listing: a week that is
// not a well-formed matchup still counts, and no roster file is opened.
func (t *Tree) LatestWeek(season int) (week int, ok bool) {
	entries, err := fs.ReadDir(t.fsys, fmt.Sprint(season))
	if err != nil {
		return 0, false
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		n, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		if !ok || n > week {
			week, ok = n, true
		}
	}
	return week, ok
}

// Matchup resolves a season and week to that week's two team names, ours
// first. It never opens either lineup: whether a lineup is usable is a fact
// about the file, reported by Read.
func (t *Tree) Matchup(season, week int) (ours, theirs string, err error) {
	dir := t.weekDir(season, week)

	entries, err := fs.ReadDir(t.fsys, dir)
	if err != nil {
		return "", "", fmt.Errorf("%s: %w", dir, ErrNoWeek)
	}

	var names, teams []string
	for _, e := range entries {
		// Finder writes .DS_Store into any directory it opens, and an editor
		// leaves .wood.csv.swp behind, so counting dotfiles would break weeks
		// that are perfectly well formed.
		if !e.Type().IsRegular() || strings.HasPrefix(e.Name(), ".") || !strings.HasSuffix(e.Name(), ".csv") {
			continue
		}
		names = append(names, e.Name())
		teams = append(teams, strings.TrimSuffix(e.Name(), ".csv"))
	}

	// No rule picks two of three lineups as the matchup, and a guess would
	// render a full, plausible page for a game nobody is playing.
	if len(teams) > 2 {
		return "", "", fmt.Errorf("%s: %w: found %s", dir, ErrTooManyLineups, found(names))
	}

	if len(teams) < 2 {
		return "", "", fmt.Errorf("%s: %w: found %s", dir, ErrTooFewLineups, found(names))
	}

	// This resolver answers who the Bojjaes are playing, which a directory of
	// two other teams cannot. The opponent is then found by elimination — it
	// is the lineup that is not ours, never inferred from its name, size, or
	// position in the listing.
	switch ourTeam {
	case teams[0]:
		return teams[0], teams[1], nil
	case teams[1]:
		return teams[1], teams[0], nil
	}
	return "", "", fmt.Errorf("%s: %w: found %s", dir, ErrNotOurMatchup, found(names))
}

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
			p, _ := t.file(s, w, team) // team was checked above
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
