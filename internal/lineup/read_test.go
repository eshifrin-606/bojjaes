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
		{ID: "101", Name: "Alpha One", ShortName: "A One", Position: "QB", Team: "BUF"},
		{ID: "102", Name: "Bravo Two", ShortName: "B Two", Position: "RB", Team: "KC"},
		{ID: "103", Name: "Charlie Three", ShortName: "C Three", Position: "WR", Team: "SF"},
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
	if !strings.Contains(err.Error(), "4") {
		t.Errorf("Read() error = %q, want it to name line 4", err)
	}
}

func TestTreeReadReadsThroughTheSuppliedFilesystem(t *testing.T) {
	tree := New(fstest.MapFS{
		"2025/14/wood.csv": &fstest.MapFile{Data: []byte(header + "101,Alpha One,QB,BUF\n102,Bravo Two,RB,KC\n")},
	})

	got, err := tree.Read(2025, 14, "wood")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	want := []Record{
		{ID: "101", Name: "Alpha One", ShortName: "A One", Position: "QB", Team: "BUF"},
		{ID: "102", Name: "Bravo Two", ShortName: "B Two", Position: "RB", Team: "KC"},
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
		"Alpha One": New(fstest.MapFS{"2025/14/wood.csv": &fstest.MapFile{Data: []byte(header + "101,Alpha One,QB,BUF\n")}}),
		"Bravo Two": New(fstest.MapFS{"2025/14/wood.csv": &fstest.MapFile{Data: []byte(header + "102,Bravo Two,RB,KC\n")}}),
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
func TestTreeReadResolvesShortNameCollisionsWithinEachGroup(t *testing.T) {
	fsys := fstest.MapFS{}
	seedLineup(t, fsys, 2025, 14, "wood", "shortnames.csv")

	got, err := New(fsys).Read(2025, 14, "wood")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	wantBench := []string{"Jaylen Allen", "Jordan Allen", "D Prescott"}
	if len(got.Starters()) != starterCount || len(got.Bench()) != len(wantBench) {
		t.Fatalf("Read() split = %d starters, %d bench, want %d starters, %d bench",
			len(got.Starters()), len(got.Bench()), starterCount, len(wantBench))
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
		wantJosh   string
		wantJaylen string
	}{
		{name: "Jaylen Allen on the bench", jaylenLine: 10, wantJosh: "J Allen", wantJaylen: "J Allen"},
		{name: "Jaylen Allen starting", jaylenLine: 9, wantJosh: "Josh Allen", wantJaylen: "Jaylen Allen"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			names := []string{"Puka Nacua", "Josh Allen", "Bijan Robinson", "Dak Prescott",
				"A.J. Brown", "KC Concepcion", "Caleb Williams", "Will McDonald IV", "Jameson Williams"}
			names = slices.Insert(names, tt.jaylenLine-1, "Jaylen Allen")
			var csv strings.Builder
			csv.WriteString(header)
			for i, name := range names {
				fmt.Fprintf(&csv, "%d,%s,WR,BUF\n", 300+i, name)
			}
			tree := New(fstest.MapFS{"2025/14/wood.csv": &fstest.MapFile{Data: []byte(csv.String())}})

			got, err := tree.Read(2025, 14, "wood")
			if err != nil {
				t.Fatalf("Read: %v", err)
			}

			if short := got.records[1].ShortName; short != tt.wantJosh {
				t.Errorf("Josh Allen ShortName = %q, want %q", short, tt.wantJosh)
			}
			if short := got.records[tt.jaylenLine-1].ShortName; short != tt.wantJaylen {
				t.Errorf("Jaylen Allen ShortName = %q, want %q", short, tt.wantJaylen)
			}
		})
	}
}

func TestTreeReadDoesNotResolveShortNamesAcrossTeams(t *testing.T) {
	tree := New(fstest.MapFS{
		"2025/14/wood.csv":  &fstest.MapFile{Data: []byte(header + "401,Josh Allen,QB,BUF\n")},
		"2025/14/bojja.csv": &fstest.MapFile{Data: []byte(header + "402,Jaylen Allen,WR,FA\n")},
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
		"2025/14/wood.csv": &fstest.MapFile{Data: []byte(header + "4984,,QB,BUF\n4985,,RB,KC\n")},
	})

	got, err := tree.Read(2025, 14, "wood")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	want := []Record{{ID: "4984", Position: "QB", Team: "BUF"}, {ID: "4985", Position: "RB", Team: "KC"}}
	if len(got.records) != len(want) {
		t.Fatalf("Read() records = %v, want %v", got.records, want)
	}
	for i := range want {
		if got.records[i] != want[i] {
			t.Errorf("record %d = %+v, want %+v", i, got.records[i], want[i])
		}
	}
}

func TestTreeReadRefusesHeaderWithNoRecordsNamingFile(t *testing.T) {
	path := "2025/14/wood.csv"
	tree := New(fstest.MapFS{path: &fstest.MapFile{Data: []byte("# lineup\n" + header + "# bench\n  # nothing here\n")}})

	_, err := tree.Read(2025, 14, "wood")
	if err == nil {
		t.Fatal("Read() = nil error, want an error for a header with no records")
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("Read() error = %q, want it to name file %q", err, path)
	}
}
