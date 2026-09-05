package roster

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWeekDirResolvesUnderRoot(t *testing.T) {
	tree := New("scripts/lineups")

	got := tree.weekDir(2025, 14)

	want := "scripts/lineups/2025/14"
	if got != want {
		t.Errorf("weekDir() = %q, want %q", got, want)
	}
}

// seedWeek creates a week directory under a temp root and writes each named
// file into it. Contents are empty: resolution must never open them.
func seedWeek(t *testing.T, season, week int, names ...string) *Tree {
	t.Helper()

	root := t.TempDir()
	dir := filepath.Join(root, fmt.Sprint(season), fmt.Sprint(week))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", name, err)
		}
	}

	return New(root)
}

func TestMatchupResolvesTwoRosters(t *testing.T) {
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

// Regression: resolution is a fact about the directory, so a roster that
// parse would reject must not affect it. Passes without a code change; it is
// here to pin that Matchup never opens the files.
func TestMatchupIgnoresUnparseableRoster(t *testing.T) {
	tree := seedWeek(t, 2025, 14, "bojjaes.csv", "wood.csv")

	bad := filepath.Join(tree.weekDir(2025, 14), "wood.csv")
	if err := os.WriteFile(bad, []byte(",Alpha\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
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

func TestMatchupRefusesThreeRosters(t *testing.T) {
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

func TestMatchupRefusesLoneRoster(t *testing.T) {
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
	if !errors.Is(err, ErrTooFewRosters) {
		t.Fatalf("Matchup() err = %v, want ErrTooFewRosters", err)
	}
	if !strings.Contains(err.Error(), tree.weekDir(2025, 14)) {
		t.Errorf("error %q does not name the week directory", err)
	}
}

// The caller asked about a week, not a path, so a missing directory is
// reported as an unstaged week rather than a bare ENOENT.
func TestMatchupRefusesMissingWeek(t *testing.T) {
	tree := New(t.TempDir())

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
		{name: "too many", files: []string{"aroma.csv", "bojjaes.csv", "wood.csv"}, stage: true, want: ErrTooManyRosters},
		{name: "too few", files: []string{"bojjaes.csv"}, stage: true, want: ErrTooFewRosters},
		{name: "no bojjaes", files: []string{"aroma.csv", "fuego.csv"}, stage: true, want: ErrNotOurMatchup},
		{name: "missing directory", want: ErrNoWeek},
	}

	others := []error{ErrTooManyRosters, ErrTooFewRosters, ErrNotOurMatchup, ErrNoWeek}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := New(t.TempDir())
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
	tree := seedWeek(t, 2025, 14, "bojjaes.csv", "wood.csv")
	for _, name := range []string{"archive", "aroma.csv"} {
		if err := os.Mkdir(filepath.Join(tree.weekDir(2025, 14), name), 0o755); err != nil {
			t.Fatalf("Mkdir %s: %v", name, err)
		}
	}

	ours, theirs, err := tree.Matchup(2025, 14)
	if err != nil {
		t.Fatalf("Matchup: %v", err)
	}
	if ours != "bojjaes" || theirs != "wood" {
		t.Errorf("Matchup() = %q, %q, want %q, %q", ours, theirs, "bojjaes", "wood")
	}
}

// Pins the filter and the count against each other: skipping junk must not
// weaken the refusal of a genuine third roster.
func TestMatchupRefusesThirdRosterAmongJunk(t *testing.T) {
	tree := seedWeek(t, 2025, 14, ".DS_Store", "aroma.csv", "bojjaes.csv", "notes.md", "wood.csv")

	_, _, err := tree.Matchup(2025, 14)
	if !errors.Is(err, ErrTooManyRosters) {
		t.Fatalf("Matchup() err = %v, want ErrTooManyRosters", err)
	}
	if strings.Contains(err.Error(), ".DS_Store") || strings.Contains(err.Error(), "notes.md") {
		t.Errorf("error %q names entries that are not rosters", err)
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

// The spec's scenario as written: every kind of non-roster entry at once.
func TestMatchupIgnoresEveryNonRosterEntry(t *testing.T) {
	tree := seedWeek(t, 2025, 14, ".DS_Store", "bojjaes.csv", "notes.md", "wood.csv")
	if err := os.Mkdir(filepath.Join(tree.weekDir(2025, 14), "archive"), 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	ours, theirs, err := tree.Matchup(2025, 14)
	if err != nil {
		t.Fatalf("Matchup: %v", err)
	}
	if ours != "bojjaes" || theirs != "wood" {
		t.Errorf("Matchup() = %q, %q, want %q, %q", ours, theirs, "bojjaes", "wood")
	}
}
