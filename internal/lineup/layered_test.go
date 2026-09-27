package lineup

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

// addWeek puts a week of valid rosters into fsys, each roster's only record
// carrying the id `layer-team`, so a test can tell which layer a Read came from.
func addWeek(fsys fstest.MapFS, layer string, season, week int, teams ...string) fstest.MapFS {
	dir := fmt.Sprintf("%d/%d", season, week)
	fsys[dir] = &fstest.MapFile{Mode: fs.ModeDir}
	for _, team := range teams {
		roster := fmt.Sprintf("id,name,position,team\n%s-%s,Alpha One,QB,BUF\n", layer, team)
		fsys[dir+"/"+team+".csv"] = &fstest.MapFile{Data: []byte(roster)}
	}
	return fsys
}

func noLog(string, ...any) {}

// archiveThrough2026Week3 is the archive most scenarios layer over: weeks 1–3
// of 2026, each bojjaes against wood.
func archiveThrough2026Week3() fstest.MapFS {
	fsys := fstest.MapFS{}
	for week := 1; week <= 3; week++ {
		addWeek(fsys, "archive", 2026, week, "bojjaes", "wood")
	}
	return fsys
}

func TestLayeredServesALaterVolumeWeekAsTheLatest(t *testing.T) {
	volume := addWeek(fstest.MapFS{}, "volume", 2026, 4, "bojjaes", "aroma")
	tree := New(Layered(archiveThrough2026Week3(), volume, noLog))

	week, ok := tree.LatestWeek(2026)

	if !ok || week != 4 {
		t.Errorf("LatestWeek(2026) = %d, %v, want 4, true", week, ok)
	}
}

func TestLayeredReadsTheLaterVolumeWeekFromTheVolume(t *testing.T) {
	volume := addWeek(fstest.MapFS{}, "volume", 2026, 4, "bojjaes", "aroma")
	tree := New(Layered(archiveThrough2026Week3(), volume, noLog))

	ours, theirs, err := tree.Matchup(2026, 4)
	if err != nil {
		t.Fatalf("Matchup: %v", err)
	}
	if ours != "bojjaes" || theirs != "aroma" {
		t.Errorf("Matchup() = %q, %q, want %q, %q", ours, theirs, "bojjaes", "aroma")
	}

	assertFirstID(t, tree, 2026, 4, "aroma", "volume-aroma")
}

func assertFirstID(t *testing.T, tree *Tree, season, week int, team, want string) {
	t.Helper()
	l, err := tree.Read(season, week, team)
	if err != nil {
		t.Fatalf("Read(%d, %d, %q): %v", season, week, team, err)
	}
	if got := l.Starters()[0].ID; got != want {
		t.Errorf("Read(%d, %d, %q) first id = %q, want %q", season, week, team, got, want)
	}
}

func TestLayeredReadsWeeksTheVolumeDoesNotHoldFromTheArchive(t *testing.T) {
	volume := addWeek(fstest.MapFS{}, "volume", 2026, 4, "bojjaes", "aroma")
	tree := New(Layered(archiveThrough2026Week3(), volume, noLog))

	ours, theirs, err := tree.Matchup(2026, 2)
	if err != nil {
		t.Fatalf("Matchup: %v", err)
	}
	if ours != "bojjaes" || theirs != "wood" {
		t.Errorf("Matchup() = %q, %q, want %q, %q", ours, theirs, "bojjaes", "wood")
	}

	assertFirstID(t, tree, 2026, 2, "bojjaes", "archive-bojjaes")
}

func TestLayeredVolumeEditOfTheLatestArchivedWeekShadowsItWhole(t *testing.T) {
	volume := addWeek(fstest.MapFS{}, "volume", 2026, 3, "bojjaes", "aroma")
	tree := New(Layered(archiveThrough2026Week3(), volume, noLog))

	ours, theirs, err := tree.Matchup(2026, 3)
	if err != nil {
		t.Fatalf("Matchup: %v", err)
	}
	if ours != "bojjaes" || theirs != "aroma" {
		t.Errorf("Matchup() = %q, %q, want %q, %q", ours, theirs, "bojjaes", "aroma")
	}

	assertFirstID(t, tree, 2026, 3, "bojjaes", "volume-bojjaes")
}

func TestLayeredHidesTheArchivedCopyOfAShadowedWeek(t *testing.T) {
	volume := addWeek(fstest.MapFS{}, "volume", 2026, 3, "bojjaes", "aroma")
	tree := New(Layered(archiveThrough2026Week3(), volume, noLog))

	_, err := tree.Read(2026, 3, "wood")

	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Read(2026, 3, %q) error = %v, want %v", "wood", err, fs.ErrNotExist)
	}
}

func TestLayeredVolumeWeekCanOpenANewSeason(t *testing.T) {
	archive := fstest.MapFS{}
	for week := 1; week <= 16; week++ {
		addWeek(archive, "archive", 2025, week, "bojjaes", "wood")
	}
	volume := addWeek(fstest.MapFS{}, "volume", 2026, 1, "bojjaes", "aroma")
	tree := New(Layered(archive, volume, noLog))

	if !tree.HasWeek(2026, 1) {
		t.Errorf("HasWeek(2026, 1) = false, want true")
	}
	if week, ok := tree.LatestWeek(2026); !ok || week != 1 {
		t.Errorf("LatestWeek(2026) = %d, %v, want 1, true", week, ok)
	}
	if week, ok := tree.LatestWeek(2025); !ok || week != 16 {
		t.Errorf("LatestWeek(2025) = %d, %v, want 16, true", week, ok)
	}
	ours, theirs, err := tree.Matchup(2026, 1)
	if err != nil {
		t.Fatalf("Matchup: %v", err)
	}
	if ours != "bojjaes" || theirs != "aroma" {
		t.Errorf("Matchup() = %q, %q, want %q, %q", ours, theirs, "bojjaes", "aroma")
	}
	assertFirstID(t, tree, 2026, 1, "aroma", "volume-aroma")
}

func TestLayeredDoesNotServeAVolumeWeekOlderThanTheArchiveLatest(t *testing.T) {
	volume := addWeek(fstest.MapFS{}, "volume", 2026, 2, "bojjaes", "aroma")
	tree := New(Layered(archiveThrough2026Week3(), volume, noLog))

	ours, theirs, err := tree.Matchup(2026, 2)
	if err != nil {
		t.Fatalf("Matchup: %v", err)
	}
	if ours != "bojjaes" || theirs != "wood" {
		t.Errorf("Matchup() = %q, %q, want %q, %q", ours, theirs, "bojjaes", "wood")
	}
}

func TestLayeredLogsAVolumeWeekOlderThanTheArchiveLatest(t *testing.T) {
	volume := addWeek(fstest.MapFS{}, "volume", 2026, 2, "bojjaes", "aroma")
	var logged []string
	logf := func(format string, args ...any) { logged = append(logged, fmt.Sprintf(format, args...)) }

	Layered(archiveThrough2026Week3(), volume, logf)

	if len(logged) != 1 || !strings.Contains(logged[0], "2026/2") {
		t.Errorf("logged %q, want one line naming 2026/2", logged)
	}
}

func TestLayeredServesTheLaterOfAStaleAndACurrentVolumeWeek(t *testing.T) {
	volume := addWeek(fstest.MapFS{}, "volume", 2026, 2, "bojjaes", "aroma")
	addWeek(volume, "volume", 2026, 4, "bojjaes", "aroma")
	var logged []string
	logf := func(format string, args ...any) { logged = append(logged, fmt.Sprintf(format, args...)) }
	tree := New(Layered(archiveThrough2026Week3(), volume, logf))

	if week, ok := tree.LatestWeek(2026); !ok || week != 4 {
		t.Errorf("LatestWeek(2026) = %d, %v, want 4, true", week, ok)
	}
	assertFirstID(t, tree, 2026, 4, "bojjaes", "volume-bojjaes")
	assertFirstID(t, tree, 2026, 2, "bojjaes", "archive-bojjaes")
	if len(logged) != 1 || !strings.Contains(logged[0], "2026/2") {
		t.Errorf("logged %q, want one line naming 2026/2", logged)
	}
}

func TestLayeredServesTheGreatestOfSeveralCurrentVolumeWeeks(t *testing.T) {
	volume := addWeek(fstest.MapFS{}, "volume", 2026, 4, "bojjaes", "aroma")
	addWeek(volume, "volume", 2026, 5, "bojjaes", "aroma")
	tree := New(Layered(archiveThrough2026Week3(), volume, noLog))

	if week, ok := tree.LatestWeek(2026); !ok || week != 5 {
		t.Errorf("LatestWeek(2026) = %d, %v, want 5, true", week, ok)
	}
	assertFirstID(t, tree, 2026, 5, "bojjaes", "volume-bojjaes")
}

func TestLayeredLogsASmallerCurrentVolumeWeekAsIgnored(t *testing.T) {
	volume := addWeek(fstest.MapFS{}, "volume", 2026, 4, "bojjaes", "aroma")
	addWeek(volume, "volume", 2026, 5, "bojjaes", "aroma")
	var logged []string
	logf := func(format string, args ...any) { logged = append(logged, fmt.Sprintf(format, args...)) }

	Layered(archiveThrough2026Week3(), volume, logf)

	if len(logged) != 1 || !strings.Contains(logged[0], "2026/4") {
		t.Errorf("logged %q, want one line naming 2026/4", logged)
	}
}

// assertReadsAsArchive checks the layered tree answers every Tree question
// about archiveThrough2026Week3 as the archive alone does.
func assertReadsAsArchive(t *testing.T, layered fs.FS) {
	t.Helper()
	want := New(archiveThrough2026Week3())
	got := New(layered)

	gotWeek, gotOK := got.LatestWeek(2026)
	wantWeek, wantOK := want.LatestWeek(2026)
	if gotWeek != wantWeek || gotOK != wantOK {
		t.Errorf("LatestWeek(2026) = %d, %v, want %d, %v", gotWeek, gotOK, wantWeek, wantOK)
	}
	for week := 1; week <= 4; week++ {
		if got.HasWeek(2026, week) != want.HasWeek(2026, week) {
			t.Errorf("HasWeek(2026, %d) = %v, want %v", week, got.HasWeek(2026, week), want.HasWeek(2026, week))
		}
	}
	ours, theirs, err := got.Matchup(2026, 3)
	if err != nil || ours != "bojjaes" || theirs != "wood" {
		t.Errorf("Matchup(2026, 3) = %q, %q, %v, want %q, %q, nil", ours, theirs, err, "bojjaes", "wood")
	}
	assertFirstID(t, got, 2026, 3, "wood", "archive-wood")
}

func TestLayeredOverAnEmptyVolumeReadsAsTheArchive(t *testing.T) {
	assertReadsAsArchive(t, Layered(archiveThrough2026Week3(), fstest.MapFS{}, noLog))
}

func TestLayeredOverAMissingVolumeReadsAsTheArchiveAndCreatesNothing(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "lineups")

	assertReadsAsArchive(t, Layered(archiveThrough2026Week3(), os.DirFS(missing), noLog))

	if _, err := os.Stat(missing); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Stat(%s) error = %v, want %v", missing, err, fs.ErrNotExist)
	}
}

func TestLayeredIgnoresStrayVolumeEntries(t *testing.T) {
	volume := fstest.MapFS{
		"lost+found":     &fstest.MapFile{Mode: fs.ModeDir},
		"notes.txt":      &fstest.MapFile{Data: []byte("x")},
		"2026/draft":     &fstest.MapFile{Mode: fs.ModeDir},
		"2026/notes.txt": &fstest.MapFile{Data: []byte("x")},
	}
	var logged []string
	logf := func(format string, args ...any) { logged = append(logged, fmt.Sprintf(format, args...)) }

	assertReadsAsArchive(t, Layered(archiveThrough2026Week3(), volume, logf))

	if len(logged) != 0 {
		t.Errorf("logged %q, want nothing", logged)
	}
}

func TestLayeredIsAConformingFS(t *testing.T) {
	t.Run("volume shadows the archive latest", func(t *testing.T) {
		volume := addWeek(fstest.MapFS{}, "volume", 2026, 3, "bojjaes", "aroma")
		fsys := Layered(archiveThrough2026Week3(), volume, noLog)

		if err := fstest.TestFS(fsys, "2026/1/wood.csv", "2026/3/bojjaes.csv", "2026/3/aroma.csv"); err != nil {
			t.Error(err)
		}
	})
	t.Run("volume opens a new season", func(t *testing.T) {
		archive := addWeek(fstest.MapFS{}, "archive", 2025, 16, "bojjaes", "wood")
		volume := addWeek(fstest.MapFS{}, "volume", 2026, 1, "bojjaes", "aroma")
		fsys := Layered(archive, volume, noLog)

		if err := fstest.TestFS(fsys, "2025/16/wood.csv", "2026/1/bojjaes.csv", "2026/1/aroma.csv"); err != nil {
			t.Error(err)
		}
	})
}
