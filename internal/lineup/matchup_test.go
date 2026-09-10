package lineup

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
)

func TestWeekDirIsRelativeToTheTree(t *testing.T) {
	tree := New(fstest.MapFS{})

	got := tree.weekDir(2025, 14)

	want := "2025/14"
	if got != want {
		t.Errorf("weekDir() = %q, want %q", got, want)
	}
}

// weekFS is one week of a lineup tree, holding each named entry with empty
// contents: resolution must never open them. The week itself is an explicit
// directory entry, so a week with no files in it is still a week.
func weekFS(season, week int, names ...string) fstest.MapFS {
	dir := fmt.Sprintf("%d/%d", season, week)
	fsys := fstest.MapFS{dir: &fstest.MapFile{Mode: fs.ModeDir}}
	for _, name := range names {
		fsys[dir+"/"+name] = &fstest.MapFile{}
	}
	return fsys
}

func seedWeek(t *testing.T, season, week int, names ...string) *Tree {
	t.Helper()
	return New(weekFS(season, week, names...))
}

func TestMatchupResolvesTwoLineups(t *testing.T) {
	tree := seedWeek(t, 2025, 14, "bojjaes.csv", "wood.csv")

	ours, theirs, err := tree.Matchup(2025, 14)
	if err != nil {
		t.Fatalf("Matchup: %v", err)
	}

	if ours != "bojjaes" || theirs != "wood" {
		t.Errorf("Matchup() = %q, %q, want %q, %q", ours, theirs, "bojjaes", "wood")
	}
}

// Directory listings are sorted, so an opponent whose name sorts before ours
// is what pins the order to the Bojjaes rather than to the filesystem.
func TestMatchupReturnsOursFirstWhenOpponentSortsBefore(t *testing.T) {
	tree := seedWeek(t, 2025, 15, "aroma.csv", "bojjaes.csv")

	ours, theirs, err := tree.Matchup(2025, 15)
	if err != nil {
		t.Fatalf("Matchup: %v", err)
	}

	if ours != "bojjaes" || theirs != "aroma" {
		t.Errorf("Matchup() = %q, %q, want %q, %q", ours, theirs, "bojjaes", "aroma")
	}
}

// Regression: resolution is a fact about the directory, so a lineup that
// parse would reject must not affect it. Passes without a code change; it is
// here to pin that Matchup never opens the files.
func TestMatchupIgnoresUnparseableLineup(t *testing.T) {
	fsys := weekFS(2025, 14, "bojjaes.csv")
	fsys["2025/14/wood.csv"] = &fstest.MapFile{Data: []byte(",Alpha\n")}
	tree := New(fsys)

	if _, err := tree.Read(2025, 14, "wood"); err == nil {
		t.Fatalf("Read() = nil error, want the fixture to be unparseable")
	}

	ours, theirs, err := tree.Matchup(2025, 14)
	if err != nil {
		t.Fatalf("Matchup: %v", err)
	}
	if ours != "bojjaes" || theirs != "wood" {
		t.Errorf("Matchup() = %q, %q, want %q, %q", ours, theirs, "bojjaes", "wood")
	}
}

func TestMatchupRefusesThreeLineups(t *testing.T) {
	tree := seedWeek(t, 2025, 14, "aroma.csv", "bojjaes.csv", "wood.csv")

	ours, theirs, err := tree.Matchup(2025, 14)
	if err == nil {
		t.Fatalf("Matchup() = %q, %q, err = nil, want an error", ours, theirs)
	}
	if ours != "" || theirs != "" {
		t.Errorf("Matchup() = %q, %q on error, want empty names", ours, theirs)
	}

	msg := err.Error()
	for _, want := range []string{tree.weekDir(2025, 14), "aroma.csv", "bojjaes.csv", "wood.csv"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error %q does not name %q", msg, want)
		}
	}
}

func TestMatchupRefusesLoneLineup(t *testing.T) {
	tree := seedWeek(t, 2025, 14, "bojjaes.csv")

	ours, theirs, err := tree.Matchup(2025, 14)
	if err == nil {
		t.Fatalf("Matchup() = %q, %q, err = nil, want an error", ours, theirs)
	}
	if !strings.Contains(err.Error(), tree.weekDir(2025, 14)) {
		t.Errorf("error %q does not name the week directory", err)
	}
}

func TestMatchupRefusesEmptyWeek(t *testing.T) {
	tree := seedWeek(t, 2025, 14)

	_, _, err := tree.Matchup(2025, 14)
	if !errors.Is(err, ErrTooFewLineups) {
		t.Fatalf("Matchup() err = %v, want ErrTooFewLineups", err)
	}
	if !strings.Contains(err.Error(), tree.weekDir(2025, 14)) {
		t.Errorf("error %q does not name the week directory", err)
	}
}

// The caller asked about a week, not a path, so a missing directory is
// reported as an unstaged week rather than a bare ENOENT.
func TestMatchupRefusesMissingWeek(t *testing.T) {
	tree := New(fstest.MapFS{})

	_, _, err := tree.Matchup(2025, 99)
	if err == nil {
		t.Fatalf("Matchup() err = nil, want an error")
	}
	if !errors.Is(err, ErrNoWeek) {
		t.Errorf("Matchup() err = %v, want ErrNoWeek", err)
	}
	if !strings.Contains(err.Error(), tree.weekDir(2025, 99)) {
		t.Errorf("error %q does not name the week directory", err)
	}
}

func TestMatchupRefusesWeekWithoutUs(t *testing.T) {
	tree := seedWeek(t, 2025, 14, "aroma.csv", "fuego.csv")

	ours, theirs, err := tree.Matchup(2025, 14)
	if err == nil {
		t.Fatalf("Matchup() = %q, %q, err = nil, want an error", ours, theirs)
	}
	if !errors.Is(err, ErrNotOurMatchup) {
		t.Errorf("Matchup() err = %v, want ErrNotOurMatchup", err)
	}
	if !strings.Contains(err.Error(), tree.weekDir(2025, 14)) {
		t.Errorf("error %q does not name the week directory", err)
	}
}

// The four refusals are four different mistakes with four different fixes, so
// a caller must be able to tell them apart without reading the message.
func TestMatchupRefusalsAreDistinguishable(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		stage bool
		want  error
	}{
		{name: "too many", files: []string{"aroma.csv", "bojjaes.csv", "wood.csv"}, stage: true, want: ErrTooManyLineups},
		{name: "too few", files: []string{"bojjaes.csv"}, stage: true, want: ErrTooFewLineups},
		{name: "no bojjaes", files: []string{"aroma.csv", "fuego.csv"}, stage: true, want: ErrNotOurMatchup},
		{name: "missing directory", want: ErrNoWeek},
	}

	others := []error{ErrTooManyLineups, ErrTooFewLineups, ErrNotOurMatchup, ErrNoWeek}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := New(fstest.MapFS{})
			if tt.stage {
				tree = seedWeek(t, 2025, 14, tt.files...)
			}

			_, _, err := tree.Matchup(2025, 14)
			if !errors.Is(err, tt.want) {
				t.Fatalf("Matchup() err = %v, want %v", err, tt.want)
			}
			for _, other := range others {
				if other != tt.want && errors.Is(err, other) {
					t.Errorf("Matchup() err = %v also matches %v, want the cases distinguishable", err, other)
				}
			}
		})
	}
}

func TestMatchupIgnoresNonCSV(t *testing.T) {
	tree := seedWeek(t, 2025, 14, "bojjaes.csv", "notes.md", "wood.csv")

	ours, theirs, err := tree.Matchup(2025, 14)
	if err != nil {
		t.Fatalf("Matchup: %v", err)
	}
	if ours != "bojjaes" || theirs != "wood" {
		t.Errorf("Matchup() = %q, %q, want %q, %q", ours, theirs, "bojjaes", "wood")
	}
}

// The regression that matters most on a hand-edited tree: Finder writes
// .DS_Store into any directory it opens.
func TestMatchupIgnoresDSStore(t *testing.T) {
	tree := seedWeek(t, 2025, 14, ".DS_Store", "bojjaes.csv", "wood.csv")

	ours, theirs, err := tree.Matchup(2025, 14)
	if err != nil {
		t.Fatalf("Matchup: %v", err)
	}
	if ours != "bojjaes" || theirs != "wood" {
		t.Errorf("Matchup() = %q, %q, want %q, %q", ours, theirs, "bojjaes", "wood")
	}
}

func TestMatchupIgnoresDotfileCSVs(t *testing.T) {
	tree := seedWeek(t, 2025, 14, ".wood.csv", ".wood.csv.swp", "bojjaes.csv", "wood.csv")

	ours, theirs, err := tree.Matchup(2025, 14)
	if err != nil {
		t.Fatalf("Matchup: %v", err)
	}
	if ours != "bojjaes" || theirs != "wood" {
		t.Errorf("Matchup() = %q, %q, want %q, %q", ours, theirs, "bojjaes", "wood")
	}
}

func TestMatchupIgnoresSubdirectories(t *testing.T) {
	fsys := weekFS(2025, 14, "bojjaes.csv", "wood.csv")
	for _, name := range []string{"archive", "aroma.csv"} {
		fsys["2025/14/"+name] = &fstest.MapFile{Mode: fs.ModeDir}
	}
	tree := New(fsys)

	ours, theirs, err := tree.Matchup(2025, 14)
	if err != nil {
		t.Fatalf("Matchup: %v", err)
	}
	if ours != "bojjaes" || theirs != "wood" {
		t.Errorf("Matchup() = %q, %q, want %q, %q", ours, theirs, "bojjaes", "wood")
	}
}

// Pins the filter and the count against each other: skipping junk must not
// weaken the refusal of a genuine third lineup.
func TestMatchupRefusesThirdLineupAmongJunk(t *testing.T) {
	tree := seedWeek(t, 2025, 14, ".DS_Store", "aroma.csv", "bojjaes.csv", "notes.md", "wood.csv")

	_, _, err := tree.Matchup(2025, 14)
	if !errors.Is(err, ErrTooManyLineups) {
		t.Fatalf("Matchup() err = %v, want ErrTooManyLineups", err)
	}
	if strings.Contains(err.Error(), ".DS_Store") || strings.Contains(err.Error(), "notes.md") {
		t.Errorf("error %q names entries that are not lineups", err)
	}
}

// The comparison against ourTeam is an exact, lowercase one, so a mixed-case
// file fails the same way on a case-insensitive filesystem as on a sensitive
// one rather than resolving on one machine and not the other.
func TestMatchupRefusesMixedCaseBojjaes(t *testing.T) {
	tree := seedWeek(t, 2025, 14, "BOJJAES.csv", "wood.csv")

	_, _, err := tree.Matchup(2025, 14)
	if !errors.Is(err, ErrNotOurMatchup) {
		t.Fatalf("Matchup() err = %v, want ErrNotOurMatchup", err)
	}
}

// The spec's scenario as written: every kind of non-lineup entry at once.
func TestMatchupIgnoresEveryNonLineupEntry(t *testing.T) {
	fsys := weekFS(2025, 14, ".DS_Store", "bojjaes.csv", "notes.md", "wood.csv")
	fsys["2025/14/archive"] = &fstest.MapFile{Mode: fs.ModeDir}
	tree := New(fsys)

	ours, theirs, err := tree.Matchup(2025, 14)
	if err != nil {
		t.Fatalf("Matchup: %v", err)
	}
	if ours != "bojjaes" || theirs != "wood" {
		t.Errorf("Matchup() = %q, %q, want %q, %q", ours, theirs, "bojjaes", "wood")
	}
}

func TestMatchupListsThroughTheSuppliedFilesystem(t *testing.T) {
	tree := New(fstest.MapFS{
		"2025/14/bojjaes.csv": &fstest.MapFile{},
		"2025/14/wood.csv":    &fstest.MapFile{},
	})

	ours, theirs, err := tree.Matchup(2025, 14)
	if err != nil {
		t.Fatalf("Matchup: %v", err)
	}
	if ours != "bojjaes" || theirs != "wood" {
		t.Errorf("Matchup() = %q, %q, want %q, %q", ours, theirs, "bojjaes", "wood")
	}
}
