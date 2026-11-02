package lineup

import (
	"errors"
	"maps"
	"testing"
	"testing/fstest"
)

// treeOf merges single-week trees into one.
func treeOf(weeks ...fstest.MapFS) *Tree {
	fsys := fstest.MapFS{}
	for _, w := range weeks {
		maps.Copy(fsys, w)
	}
	return New(fsys)
}

func assertLatest(t *testing.T, tree *Tree, team string, wantSeason, wantWeek int) {
	t.Helper()
	season, week, err := tree.LatestLineup(team)
	if err != nil {
		t.Fatalf("LatestLineup(%q): %v", team, err)
	}
	if season != wantSeason || week != wantWeek {
		t.Errorf("LatestLineup(%q) = %d/%d, want %d/%d", team, season, week, wantSeason, wantWeek)
	}
}

func TestLatestLineupFindsTheTeamsOnlyWeek(t *testing.T) {
	tree := treeOf(weekFS(2026, 3, "bojjaes.csv", "gonads.csv"))

	assertLatest(t, tree, "gonads", 2026, 3)
}

func TestLatestLineupPrefersTheNewerWeek(t *testing.T) {
	tree := treeOf(
		weekFS(2026, 2, "bojjaes.csv", "renegades.csv"),
		weekFS(2026, 3, "bojjaes.csv", "vampyres.csv"),
	)

	assertLatest(t, tree, "bojjaes", 2026, 3)
}

// A listing sorts "10" before "9", so only a numeric order puts week 10 last.
func TestLatestLineupOrdersWeeksByNumber(t *testing.T) {
	tree := treeOf(
		weekFS(2026, 9, "bojjaes.csv", "jabs.csv"),
		weekFS(2026, 10, "bojjaes.csv", "saban.csv"),
	)

	assertLatest(t, tree, "bojjaes", 2026, 10)
}

func TestLatestLineupLooksBackIntoEarlierSeasons(t *testing.T) {
	tree := treeOf(
		weekFS(2025, 16, "bojjaes.csv", "gonads.csv"),
		weekFS(2026, 1, "bojjaes.csv", "fuego.csv"),
	)

	assertLatest(t, tree, "gonads", 2025, 16)
	assertLatest(t, tree, "bojjaes", 2026, 1)
}

func TestLatestLineupReportsATeamWithNoLineup(t *testing.T) {
	tree := treeOf(weekFS(2026, 3, "bojjaes.csv", "gonads.csv"))

	_, _, err := tree.LatestLineup("wood")

	if !errors.Is(err, ErrNoLineup) {
		t.Errorf("LatestLineup(%q) error = %v, want %v", "wood", err, ErrNoLineup)
	}
}

// A name that is not a plain file name is the caller's mistake, not a team
// that never filed a lineup.
func TestLatestLineupRejectsAPathAsATeam(t *testing.T) {
	tree := treeOf(weekFS(2026, 3, "bojjaes.csv", "gonads.csv"))

	_, _, err := tree.LatestLineup("../gonads")

	if err == nil || errors.Is(err, ErrNoLineup) {
		t.Errorf("LatestLineup(%q) error = %v, want an invalid team name", "../gonads", err)
	}
}

func TestLatestLineupSkipsEntriesThatAreNotNumbered(t *testing.T) {
	tree := treeOf(
		weekFS(2026, 3, "bojjaes.csv", "gonads.csv"),
		fstest.MapFS{
			"notes/1/gonads.csv":  &fstest.MapFile{},
			"2026/old/gonads.csv": &fstest.MapFile{},
			"2026/4":              &fstest.MapFile{},
		},
	)

	assertLatest(t, tree, "gonads", 2026, 3)
}
