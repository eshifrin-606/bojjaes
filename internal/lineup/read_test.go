package lineup

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
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
		{ID: "101", Name: "Alpha One", ShortName: "A One"},
		{ID: "102", Name: "Bravo Two", ShortName: "B Two"},
		{ID: "103", Name: "Charlie Three", ShortName: "C Three"},
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

	want := []Record{
		{ID: "101", Name: "Alpha One", ShortName: "A One"},
		{ID: "102", Name: "Bravo Two", ShortName: "B Two"},
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

// Collisions are resolved within starters and bench separately, so the two
// bench Allens leave the starting Josh Allen with his short name.
func TestTreeReadResolvesShortNameCollisionsWithinStarters(t *testing.T) {
	fsys := fstest.MapFS{}
	seedLineup(t, fsys, 2025, 14, "wood", "shortnames.csv")

	got, err := New(fsys).Read(2025, 14, "wood")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	wantStarters := map[int]string{
		0: "Jameson Williams",
		1: "J Allen",
		2: "Javonte Williams",
		3: "C Williams",
		4: "W McDonald",
		5: "A.J. Brown",
		6: "KC Concepcion",
	}
	for i, want := range wantStarters {
		if short := got.Starters()[i].ShortName; short != want {
			t.Errorf("Starters()[%d].ShortName = %q, want %q", i, short, want)
		}
	}
	if name := got.Starters()[4].Name; name != "Will McDonald IV" {
		t.Errorf("Starters()[4].Name = %q, want %q", name, "Will McDonald IV")
	}

	wantBench := []string{"Jaylen Allen", "Jordan Allen", "D Prescott"}
	for i, want := range wantBench {
		if short := got.Bench()[i].ShortName; short != want {
			t.Errorf("Bench()[%d].ShortName = %q, want %q", i, short, want)
		}
	}
}

// Moving a colliding player across the ninth line changes whether the
// collision counts, because only players in the same group collide.
func TestTreeReadShortNameCollisionFollowsTheStarterSplit(t *testing.T) {
	tests := []struct {
		name       string
		jaylenLine int
		want       string
	}{
		{name: "Jaylen Allen on the bench", jaylenLine: 10, want: "J Allen"},
		{name: "Jaylen Allen starting", jaylenLine: 9, want: "Josh Allen"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			names := []string{"Puka Nacua", "Josh Allen", "Bijan Robinson", "Dak Prescott",
				"A.J. Brown", "KC Concepcion", "Caleb Williams", "Will McDonald IV", "Jameson Williams"}
			names = slices.Insert(names, tt.jaylenLine-1, "Jaylen Allen")
			var csv strings.Builder
			for i, name := range names {
				fmt.Fprintf(&csv, "%d,%s\n", 300+i, name)
			}
			tree := New(fstest.MapFS{"2025/14/wood.csv": &fstest.MapFile{Data: []byte(csv.String())}})

			got, err := tree.Read(2025, 14, "wood")
			if err != nil {
				t.Fatalf("Read: %v", err)
			}

			if short := got.Starters()[1].ShortName; short != tt.want {
				t.Errorf("Josh Allen ShortName = %q, want %q", short, tt.want)
			}
		})
	}
}

func TestTreeReadDoesNotResolveShortNamesAcrossTeams(t *testing.T) {
	tree := New(fstest.MapFS{
		"2025/14/wood.csv":  &fstest.MapFile{Data: []byte("401,Josh Allen\n")},
		"2025/14/bojja.csv": &fstest.MapFile{Data: []byte("402,Jaylen Allen\n")},
	})

	for _, team := range []string{"wood", "bojja"} {
		got, err := tree.Read(2025, 14, team)
		if err != nil {
			t.Fatalf("Read %s: %v", team, err)
		}
		if short := got.Starters()[0].ShortName; short != "J Allen" {
			t.Errorf("%s: ShortName = %q, want %q", team, short, "J Allen")
		}
	}
}

func TestTreeReadLeavesEmptyNamesWithEmptyShortNames(t *testing.T) {
	tree := New(fstest.MapFS{
		"2025/14/wood.csv": &fstest.MapFile{Data: []byte("4984,\n4985\n")},
	})

	got, err := tree.Read(2025, 14, "wood")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	want := []Record{{ID: "4984"}, {ID: "4985"}}
	if len(got.records) != len(want) {
		t.Fatalf("Read() records = %v, want %v", got.records, want)
	}
	for i := range want {
		if got.records[i] != want[i] {
			t.Errorf("record %d = %+v, want %+v", i, got.records[i], want[i])
		}
	}
}
