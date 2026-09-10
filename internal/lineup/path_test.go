package lineup

import (
	"testing"
	"testing/fstest"
)

func TestPathResolvesWithinTheTree(t *testing.T) {
	tree := New(fstest.MapFS{})

	got, err := tree.Path(2025, 14, "wood")
	if err != nil {
		t.Fatalf("Path: %v", err)
	}

	want := "2025/14/wood.csv"
	if got != want {
		t.Errorf("Path() = %q, want %q", got, want)
	}
}

// A team name is a URL segment once the page exists, so a name holding a
// separator must be refused rather than resolved into (or out of) the tree.
func TestPathRejectsTeamWithSeparator(t *testing.T) {
	tree := New(fstest.MapFS{})

	got, err := tree.Path(2025, 14, "wood/../../etc")
	if err == nil {
		t.Fatalf("Path() = %q, err = nil, want an error", got)
	}
	if got != "" {
		t.Errorf("Path() = %q on error, want empty path so nothing is opened", got)
	}
}

func TestPathRejectsParentReference(t *testing.T) {
	tests := []string{"..", "wood..bad"}

	for _, team := range tests {
		tree := New(fstest.MapFS{})

		got, err := tree.Path(2025, 14, team)
		if err == nil {
			t.Errorf("team %q: Path() = %q, err = nil, want an error", team, got)
		}
		if got != "" {
			t.Errorf("team %q: Path() = %q on error, want empty path", team, got)
		}
	}
}

func TestPathFromSuppliedFilesystemIsRelativeToIt(t *testing.T) {
	tree := New(fstest.MapFS{})

	got, err := tree.Path(2025, 14, "wood")
	if err != nil {
		t.Fatalf("Path: %v", err)
	}

	want := "2025/14/wood.csv"
	if got != want {
		t.Errorf("Path() = %q, want %q", got, want)
	}
}
