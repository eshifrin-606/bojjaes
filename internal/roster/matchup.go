package roster

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ourTeam is the Bojjaes' roster file name, without the extension. This repo
// is a tool for one team, so the name lives here rather than in every caller.
const ourTeam = "bojjaes"

// The four ways a week directory can fail to be a matchup. They are separate
// values because they are four different mistakes with four different fixes,
// and a caller that renders one needs to tell them apart.
var (
	ErrNoWeek         = errors.New("no week directory")
	ErrTooFewRosters  = errors.New("fewer than two rosters")
	ErrTooManyRosters = errors.New("more than two rosters")
	ErrNotOurMatchup  = errors.New("no bojjaes roster")
)

func found(names []string) string {
	if len(names) == 0 {
		return "no roster files"
	}
	return strings.Join(names, ", ")
}

// weekDir is where Path gets the directory it puts a roster file in, so the
// week and the files inside it cannot disagree about where the tree is.
func (t *Tree) weekDir(season, week int) string {
	return filepath.Join(t.root, fmt.Sprint(season), fmt.Sprint(week))
}

// Matchup resolves a season and week to that week's two team names, ours
// first. It never opens either roster: whether a roster is usable is a fact
// about the file, reported by Read.
func (t *Tree) Matchup(season, week int) (ours, theirs string, err error) {
	dir := t.weekDir(season, week)

	entries, err := os.ReadDir(dir)
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

	// No rule picks two of three rosters as the matchup, and a guess would
	// render a full, plausible page for a game nobody is playing.
	if len(teams) > 2 {
		return "", "", fmt.Errorf("%s: %w: found %s", dir, ErrTooManyRosters, found(names))
	}

	if len(teams) < 2 {
		return "", "", fmt.Errorf("%s: %w: found %s", dir, ErrTooFewRosters, found(names))
	}

	// This resolver answers who the Bojjaes are playing, which a directory of
	// two other teams cannot. The opponent is then found by elimination — it
	// is the roster that is not ours, never inferred from its name, size, or
	// position in the listing.
	switch ourTeam {
	case teams[0]:
		return teams[0], teams[1], nil
	case teams[1]:
		return teams[1], teams[0], nil
	}
	return "", "", fmt.Errorf("%s: %w: found %s", dir, ErrNotOurMatchup, found(names))
}
