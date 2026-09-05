package roster

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// seedRoster copies a testdata fixture into a tree at the location Path
// would resolve for the given season/week/team, and returns that path.
func seedRoster(t *testing.T, root string, season, week int, team, fixture string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("testdata", fixture))
	if err != nil {
		t.Fatalf("reading fixture %s: %v", fixture, err)
	}

	dir := filepath.Join(root, strconv.Itoa(season), strconv.Itoa(week))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	path := filepath.Join(dir, team+".csv")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func TestTreeReadReturnsParsedRecords(t *testing.T) {
	root := t.TempDir()
	seedRoster(t, root, 2025, 14, "wood", "wood.csv")

	tree := New(root)
	got, err := tree.Read(2025, 14, "wood")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	want := []Record{
		{ID: "101", Name: "Alpha One"},
		{ID: "102", Name: "Bravo Two"},
		{ID: "103", Name: "Charlie Three"},
	}
	if len(got.records) != len(want) {
		t.Fatalf("Read() records = %v, want %v", got.records, want)
	}
	for i := range want {
		if got.records[i] != want[i] {
			t.Errorf("record %d = %v, want %v", i, got.records[i], want[i])
		}
	}
}

func TestTreeReadMissingFileNamesPath(t *testing.T) {
	root := t.TempDir()
	tree := New(root)

	_, err := tree.Read(2025, 14, "wood")
	if err == nil {
		t.Fatal("Read() = nil error, want an error naming the missing path")
	}

	wantPath := filepath.Join(root, "2025", "14", "wood.csv")
	if !strings.Contains(err.Error(), wantPath) {
		t.Errorf("Read() error = %q, want it to name path %q", err, wantPath)
	}
}

func TestTreeReadParseErrorNamesFile(t *testing.T) {
	root := t.TempDir()
	path := seedRoster(t, root, 2025, 14, "wood", "badline.csv")

	tree := New(root)
	_, err := tree.Read(2025, 14, "wood")
	if err == nil {
		t.Fatal("Read() = nil error, want a parse error")
	}

	if !strings.Contains(err.Error(), path) {
		t.Errorf("Read() error = %q, want it to name file %q", err, path)
	}
	if !strings.Contains(err.Error(), "3") {
		t.Errorf("Read() error = %q, want it to name line 3", err)
	}
}
