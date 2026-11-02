package lineup

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

func TestFileResolvesWithinTheTree(t *testing.T) {
	tree := New(fstest.MapFS{})

	got, err := tree.file(2025, 14, "wood")
	if err != nil {
		t.Fatalf("file: %v", err)
	}

	want := "2025/14/wood.csv"
	if got != want {
		t.Errorf("file() = %q, want %q", got, want)
	}
}

// A team name is a URL segment once the page exists, so a name holding a
// separator must be refused rather than resolved into (or out of) the tree.
func TestFileRejectsTeamWithSeparator(t *testing.T) {
	tree := New(fstest.MapFS{})

	got, err := tree.file(2025, 14, "wood/../../etc")
	if err == nil {
		t.Fatalf("file() = %q, err = nil, want an error", got)
	}
	if got != "" {
		t.Errorf("file() = %q on error, want empty path so nothing is opened", got)
	}
}

func TestFileRejectsParentReference(t *testing.T) {
	tests := []string{"..", "wood..bad"}

	for _, team := range tests {
		tree := New(fstest.MapFS{})

		got, err := tree.file(2025, 14, team)
		if err == nil {
			t.Errorf("team %q: file() = %q, err = nil, want an error", team, got)
		}
		if got != "" {
			t.Errorf("team %q: file() = %q on error, want empty path", team, got)
		}
	}
}

func TestFileFromSuppliedFilesystemIsRelativeToIt(t *testing.T) {
	tree := New(fstest.MapFS{})

	got, err := tree.file(2025, 14, "wood")
	if err != nil {
		t.Fatalf("file: %v", err)
	}

	want := "2025/14/wood.csv"
	if got != want {
		t.Errorf("file() = %q, want %q", got, want)
	}
}

// seedLineup places a testdata fixture in a tree at the name file would
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

// Colliding short names, whether within starters, within bench, or between
// them, all keep their derived short form.
func TestTreeReadKeepsCollidingShortNames(t *testing.T) {
	fsys := fstest.MapFS{}
	seedLineup(t, fsys, 2025, 14, "wood", "shortnames.csv")

	got, err := New(fsys).Read(2025, 14, "wood")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	wantBench := []string{"J Allen", "J Allen", "D Prescott"}
	if len(got.Starters()) != starterCount || len(got.Bench()) != len(wantBench) {
		t.Fatalf("Read() split = %d starters, %d bench, want %d starters, %d bench",
			len(got.Starters()), len(got.Bench()), starterCount, len(wantBench))
	}

	wantStarters := map[int]string{
		0: "J Williams",
		1: "J Allen",
		2: "J Williams",
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

// A regression pin: two starters with different ids and the same name both
// keep the short form, rather than falling back to the full name.
func TestTreeReadKeepsShortNameForRepeatedFullName(t *testing.T) {
	tree := New(fstest.MapFS{
		"2025/14/wood.csv": &fstest.MapFile{Data: []byte(header +
			"401,Josh Allen,QB,BUF\n402,Josh Allen,WR,KC\n")},
	})

	got, err := tree.Read(2025, 14, "wood")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	for i, rec := range got.records {
		if rec.ShortName != "J Allen" {
			t.Errorf("Starters()[%d].ShortName = %q, want %q", i, rec.ShortName, "J Allen")
		}
	}
}

// Moving a colliding player across the ninth line does not change another
// record's short name: each record's short name depends only on its own name.
func TestTreeReadShortNameCollisionFollowsTheStarterSplit(t *testing.T) {
	tests := []struct {
		name       string
		jaylenLine int
		wantJosh   string
		wantJaylen string
	}{
		{name: "Jaylen Allen on the bench", jaylenLine: 10, wantJosh: "J Allen", wantJaylen: "J Allen"},
		{name: "Jaylen Allen starting", jaylenLine: 9, wantJosh: "J Allen", wantJaylen: "J Allen"},
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

func TestLatestWeekIsTheGreatestPresent(t *testing.T) {
	fsys := weekFS(2026, 1, "bojjaes.csv", "wood.csv")
	maps.Copy(fsys, weekFS(2026, 2, "bojjaes.csv", "renegades.csv"))
	tree := New(fsys)

	week, ok := tree.LatestWeek(2026)
	if !ok {
		t.Fatalf("LatestWeek(2026) ok = false, want true")
	}
	if week != 2 {
		t.Errorf("LatestWeek(2026) = %d, want 2", week)
	}
}

func TestLatestWeekIsNotLoweredByAGap(t *testing.T) {
	fsys := weekFS(2026, 1, "bojjaes.csv", "wood.csv")
	maps.Copy(fsys, weekFS(2026, 3, "bojjaes.csv", "renegades.csv"))
	tree := New(fsys)

	week, ok := tree.LatestWeek(2026)
	if !ok {
		t.Fatalf("LatestWeek(2026) ok = false, want true")
	}
	if week != 3 {
		t.Errorf("LatestWeek(2026) = %d, want 3", week)
	}
}

func TestLatestWeekIsFalseForASeasonWithNoWeeks(t *testing.T) {
	tree := New(fstest.MapFS{})

	if _, ok := tree.LatestWeek(2027); ok {
		t.Errorf("LatestWeek(2027) ok = true, want false")
	}
}

func TestLatestWeekIgnoresANonIntegerEntry(t *testing.T) {
	fsys := weekFS(2026, 1, "bojjaes.csv", "wood.csv")
	fsys["2026/notaweek"] = &fstest.MapFile{Mode: fs.ModeDir}
	tree := New(fsys)

	week, ok := tree.LatestWeek(2026)
	if !ok {
		t.Fatalf("LatestWeek(2026) ok = false, want true")
	}
	if week != 1 {
		t.Errorf("LatestWeek(2026) = %d, want 1", week)
	}
}

func TestLatestWeekCountsAWeekThatIsNotAMatchup(t *testing.T) {
	tree := seedWeek(t, 2026, 3, "aroma.csv", "bojjaes.csv", "wood.csv")

	week, ok := tree.LatestWeek(2026)
	if !ok {
		t.Fatalf("LatestWeek(2026) ok = false, want true")
	}
	if week != 3 {
		t.Errorf("LatestWeek(2026) = %d, want 3", week)
	}
}

// Regression: LatestWeek must answer from the directory listing alone, never
// by opening a roster file.
func TestLatestWeekNeverOpensARosterFile(t *testing.T) {
	fsys := weekFS(2026, 2, "bojjaes.csv")
	fsys["2026/2/wood.csv"] = &fstest.MapFile{Data: []byte(",Alpha\n")}
	tree := New(fsys)

	week, ok := tree.LatestWeek(2026)
	if !ok {
		t.Fatalf("LatestWeek(2026) ok = false, want true")
	}
	if week != 2 {
		t.Errorf("LatestWeek(2026) = %d, want 2", week)
	}
}

func TestHasWeekFindsAWeekDirectory(t *testing.T) {
	tree := seedWeek(t, 2026, 2, "bojjaes.csv", "renegades.csv")

	if !tree.HasWeek(2026, 2) {
		t.Errorf("HasWeek(2026, 2) = false, want true")
	}
}

func TestHasWeekIsFalseForAMissingWeek(t *testing.T) {
	fsys := weekFS(2026, 1, "bojjaes.csv", "wood.csv")
	maps.Copy(fsys, weekFS(2026, 2, "bojjaes.csv", "renegades.csv"))
	tree := New(fsys)

	if tree.HasWeek(2026, 3) {
		t.Errorf("HasWeek(2026, 3) = true, want false")
	}
}

// Stat succeeds on a regular file, so only the directory check keeps a stray
// file from being linked to as a week that Matchup would then call missing.
func TestHasWeekIsFalseForAFileWhereAWeekShouldBe(t *testing.T) {
	tree := New(fstest.MapFS{"2026/3": &fstest.MapFile{}})

	if tree.HasWeek(2026, 3) {
		t.Errorf("HasWeek(2026, 3) = true, want false")
	}
}

// A week that is not a matchup is a mistake in the tree, and its link should
// lead to the 500 that exposes it rather than vanish.
func TestHasWeekIsTrueForAWeekThatIsNotAMatchup(t *testing.T) {
	unparseable := weekFS(2026, 3, "bojjaes.csv")
	unparseable["2026/3/wood.csv"] = &fstest.MapFile{Data: []byte(",Alpha\n")}

	tests := []struct {
		name string
		fsys fstest.MapFS
	}{
		{"three rosters", weekFS(2026, 3, "aroma.csv", "bojjaes.csv", "wood.csv")},
		{"lone roster", weekFS(2026, 3, "bojjaes.csv")},
		{"no bojjaes", weekFS(2026, 3, "aroma.csv", "wood.csv")},
		{"unparseable roster", unparseable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !New(tt.fsys).HasWeek(2026, 3) {
				t.Errorf("HasWeek(2026, 3) = false, want true")
			}
		})
	}
}

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
