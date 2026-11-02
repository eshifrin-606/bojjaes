package lineup

import (
	"errors"
	"io/fs"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

func TestRostersTeamsAreTheCSVFilesInNameOrder(t *testing.T) {
	rosters := NewRosters(fstest.MapFS{
		"wood.csv":  &fstest.MapFile{},
		"aroma.csv": &fstest.MapFile{},
		".DS_Store": &fstest.MapFile{},
		"notes.txt": &fstest.MapFile{},
	})

	got := rosters.Teams()

	want := []string{"aroma", "wood"}
	if !slices.Equal(got, want) {
		t.Errorf("Teams() = %q, want %q", got, want)
	}
}

// A roster is not a lineup: it holds more than nine, and no slot reorders it.
func TestRostersReadKeepsEveryRecordInFileOrder(t *testing.T) {
	csv := rosterCSV(t,
		"1", "K", "2", "QB", "3", "RB", "4", "RB", "5", "WR", "6", "WR",
		"7", "TE", "8", "FB", "9", "CB", "10", "LB", "11", "QB", "12", "DE",
	)
	rosters := NewRosters(fstest.MapFS{"wood.csv": &fstest.MapFile{Data: []byte(csv)}})

	got, err := rosters.Read("wood")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	var ids []string
	for _, p := range got {
		ids = append(ids, p.ID)
	}
	want := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12"}
	if !slices.Equal(ids, want) {
		t.Errorf("Read ids = %q, want %q", ids, want)
	}
}

// A path is the caller's mistake, not a team without a roster, so it is
// refused before the filesystem is asked.
func TestRostersReadRejectsAPathAsATeam(t *testing.T) {
	rosters := NewRosters(fstest.MapFS{"wood.csv": &fstest.MapFile{Data: []byte(rosterCSV(t, "1", "QB"))}})

	_, err := rosters.Read("../wood")

	if err == nil || errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Read(%q) error = %v, want an invalid team name", "../wood", err)
	}
}

func TestRostersReadRefusesAnUnknownPositionNamingTheFile(t *testing.T) {
	rosters := NewRosters(fstest.MapFS{"wood.csv": &fstest.MapFile{Data: []byte(rosterCSV(t, "1", "QB", "2", "XX"))}})

	_, err := rosters.Read("wood")

	if err == nil || !strings.Contains(err.Error(), "wood.csv") {
		t.Errorf("Read error = %v, want one naming wood.csv", err)
	}
}
