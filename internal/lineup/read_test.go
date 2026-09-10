package lineup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

// seedLineup places a testdata fixture in a tree at the name Path would
// resolve for the given season/week/team, and returns that name.
func seedLineup(t *testing.T, fsys fstest.MapFS, season, week int, team, fixture string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("testdata", fixture))
	if err != nil {
		t.Fatalf("reading fixture %s: %v", fixture, err)
	}

	name := fmt.Sprintf("%d/%d/%s.csv", season, week, team)
	fsys[name] = &fstest.MapFile{Data: content}
	return name
}

func TestTreeReadReturnsParsedRecords(t *testing.T) {
	fsys := fstest.MapFS{}
	seedLineup(t, fsys, 2025, 14, "wood", "wood.csv")

	tree := New(fsys)
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
	tree := New(fstest.MapFS{})

	_, err := tree.Read(2025, 14, "wood")
	if err == nil {
		t.Fatal("Read() = nil error, want an error naming the missing path")
	}

	wantPath := "2025/14/wood.csv"
	if !strings.Contains(err.Error(), wantPath) {
		t.Errorf("Read() error = %q, want it to name path %q", err, wantPath)
	}
}

func TestTreeReadParseErrorNamesFile(t *testing.T) {
	fsys := fstest.MapFS{}
	path := seedLineup(t, fsys, 2025, 14, "wood", "badline.csv")

	tree := New(fsys)
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

func TestTreeReadReadsThroughTheSuppliedFilesystem(t *testing.T) {
	tree := New(fstest.MapFS{
		"2025/14/wood.csv": &fstest.MapFile{Data: []byte("101,Alpha One\n102,Bravo Two\n")},
	})

	got, err := tree.Read(2025, 14, "wood")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	want := []Record{{ID: "101", Name: "Alpha One"}, {ID: "102", Name: "Bravo Two"}}
	if len(got.records) != len(want) {
		t.Fatalf("Read() records = %v, want %v", got.records, want)
	}
	for i := range want {
		if got.records[i] != want[i] {
			t.Errorf("record %d = %v, want %v", i, got.records[i], want[i])
		}
	}
}

// Two trees, the same season, week, and team: each read yields its own tree's
// lineup. The guard is against any read reaching past the supplied filesystem
// — to the working directory, or to a root remembered from somewhere else.
func TestTreeReadIsConfinedToItsOwnFilesystem(t *testing.T) {
	trees := map[string]*Tree{
		"Alpha One": New(fstest.MapFS{"2025/14/wood.csv": &fstest.MapFile{Data: []byte("101,Alpha One\n")}}),
		"Bravo Two": New(fstest.MapFS{"2025/14/wood.csv": &fstest.MapFile{Data: []byte("102,Bravo Two\n")}}),
	}

	for want, tree := range trees {
		got, err := tree.Read(2025, 14, "wood")
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		if len(got.records) != 1 || got.records[0].Name != want {
			t.Errorf("Read() records = %v, want the single record %q", got.records, want)
		}
	}
}
