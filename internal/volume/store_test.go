package volume

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"testing"

	"github.com/eshifrin/bojjaes/internal/lineup"
)

// The positions are ordered so that a read, which assigns slots by position,
// puts each player back in the slot given here.
func testLineup(t *testing.T, idPrefix string) lineup.Lineup {
	t.Helper()
	positions := [lineup.StarterCount]string{"QB", "RB", "WR", "TE", "RB", "WR", "K", "LB", "CB"}
	var players [lineup.StarterCount]lineup.Player
	for i, pos := range positions {
		id := idPrefix + strconv.Itoa(i)
		players[i] = lineup.Player{ID: id, Name: "Player " + id, Position: pos, Team: "KC"}
	}
	l, err := lineup.FromSlots(players)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func ids(l lineup.Lineup) []string {
	var out []string
	for _, p := range l.Starters() {
		out = append(out, p.ID)
	}
	return out
}

func mustSave(t *testing.T, s *Store, team string, l lineup.Lineup) {
	t.Helper()
	if err := s.Save(2026, 12, team, l); err != nil {
		t.Fatalf("Save(2026, 12, %q): %v", team, err)
	}
}

func readSaved(t *testing.T, mount, team string) lineup.Lineup {
	t.Helper()
	got, err := lineup.New(os.DirFS(filepath.Join(mount, "lineups"))).Read(2026, 12, team)
	if err != nil {
		t.Fatalf("Read(2026, 12, %q): %v", team, err)
	}
	return got
}

func TestSaveWritesALineupThatReadsBack(t *testing.T) {
	mount := t.TempDir()
	want := testLineup(t, "b")

	mustSave(t, NewStore(mount), "bojjaes", want)

	if got := readSaved(t, mount, "bojjaes"); !slices.Equal(ids(got), ids(want)) {
		t.Errorf("read back %v, want %v", ids(got), ids(want))
	}
}

func TestSaveReplacesThatTeamsLineup(t *testing.T) {
	mount := t.TempDir()
	s := NewStore(mount)
	mustSave(t, s, "bojjaes", testLineup(t, "old"))
	want := testLineup(t, "new")

	mustSave(t, s, "bojjaes", want)

	if got := readSaved(t, mount, "bojjaes"); !slices.Equal(ids(got), ids(want)) {
		t.Errorf("read back %v, want %v", ids(got), ids(want))
	}
}

func weekFiles(t *testing.T, mount string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(mount, "lineups", "2026", "12"))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestSaveAddsTheOpponentBesideTheBojjaes(t *testing.T) {
	mount := t.TempDir()
	s := NewStore(mount)
	mustSave(t, s, "bojjaes", testLineup(t, "b"))
	want := testLineup(t, "w")

	mustSave(t, s, "wood", want)

	if got := readSaved(t, mount, "wood"); !slices.Equal(ids(got), ids(want)) {
		t.Errorf("wood read back %v, want %v", ids(got), ids(want))
	}
	if got := weekFiles(t, mount); !slices.Equal(got, []string{"bojjaes.csv", "wood.csv"}) {
		t.Errorf("week holds %v, want [bojjaes.csv wood.csv]", got)
	}
}

func weekFile(t *testing.T, mount, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(mount, "lineups", "2026", "12", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func wantOpponentSet(t *testing.T, err error, existing string) {
	t.Helper()
	var set *OpponentSetError
	if !errors.As(err, &set) || set.Existing != existing {
		t.Errorf("Save error = %v, want *OpponentSetError{Existing: %q}", err, existing)
	}
}

func TestSaveRefusesADifferentOpponent(t *testing.T) {
	mount := t.TempDir()
	s := NewStore(mount)
	mustSave(t, s, "bojjaes", testLineup(t, "b"))
	mustSave(t, s, "wood", testLineup(t, "w"))
	wood := weekFile(t, mount, "wood.csv")

	err := s.Save(2026, 12, "fuego", testLineup(t, "f"))

	wantOpponentSet(t, err, "wood")
	if got := weekFiles(t, mount); !slices.Equal(got, []string{"bojjaes.csv", "wood.csv"}) {
		t.Errorf("week holds %v, want [bojjaes.csv wood.csv]", got)
	}
	if got := weekFile(t, mount, "wood.csv"); !bytes.Equal(got, wood) {
		t.Errorf("wood.csv changed:\n%s\nwant:\n%s", got, wood)
	}
}

func TestSaveRefusesADifferentOpponentSavedFirst(t *testing.T) {
	mount := t.TempDir()
	s := NewStore(mount)
	mustSave(t, s, "wood", testLineup(t, "w"))

	wantOpponentSet(t, s.Save(2026, 12, "fuego", testLineup(t, "f")), "wood")
	mustSave(t, s, "bojjaes", testLineup(t, "b"))
}

func TestSaveReplacesTheWeeksOpponent(t *testing.T) {
	mount := t.TempDir()
	s := NewStore(mount)
	mustSave(t, s, "bojjaes", testLineup(t, "b"))
	mustSave(t, s, "wood", testLineup(t, "old"))
	want := testLineup(t, "new")

	mustSave(t, s, "wood", want)

	if got := readSaved(t, mount, "wood"); !slices.Equal(ids(got), ids(want)) {
		t.Errorf("wood read back %v, want %v", ids(got), ids(want))
	}
}

func TestSaveIgnoresDotfilesAndNonCSVAsOpponents(t *testing.T) {
	mount := t.TempDir()
	s := NewStore(mount)
	mustSave(t, s, "bojjaes", testLineup(t, "b"))
	dir := filepath.Join(mount, "lineups", "2026", "12")
	for _, name := range []string{".fuego.csv.tmp", ".fuego.csv", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	mustSave(t, s, "wood", testLineup(t, "w"))
}

func TestSaveLeavesOnlyTheTargetInTheWeek(t *testing.T) {
	mount := t.TempDir()

	mustSave(t, NewStore(mount), "bojjaes", testLineup(t, "b"))

	if got := weekFiles(t, mount); !slices.Equal(got, []string{"bojjaes.csv"}) {
		t.Errorf("week holds %v, want [bojjaes.csv]", got)
	}
}

func TestAFailedSaveKeepsThePreviousLineup(t *testing.T) {
	mount := t.TempDir()
	s := NewStore(mount)
	mustSave(t, s, "bojjaes", testLineup(t, "old"))
	previous := weekFile(t, mount, "bojjaes.csv")
	s.writeLineup = func(_ lineup.Lineup, w io.Writer) error {
		io.WriteString(w, "id,name")
		return errors.New("disk full")
	}

	if err := s.Save(2026, 12, "bojjaes", testLineup(t, "new")); err == nil {
		t.Error("Save with a failing write returned nil error")
	}

	if got := weekFile(t, mount, "bojjaes.csv"); !bytes.Equal(got, previous) {
		t.Errorf("bojjaes.csv = %q, want the previous %q", got, previous)
	}
	if got := weekFiles(t, mount); !slices.Equal(got, []string{"bojjaes.csv"}) {
		t.Errorf("week holds %v, want [bojjaes.csv]", got)
	}
}

func TestConcurrentOpponentSavesLeaveOneOpponent(t *testing.T) {
	mount := t.TempDir()
	s := NewStore(mount)
	mustSave(t, s, "bojjaes", testLineup(t, "b"))
	start := make(chan struct{})
	errs := make([]error, 20)
	var wg sync.WaitGroup
	for i := range errs {
		team := []string{"wood", "fuego"}[i%2]
		l := testLineup(t, team)
		wg.Go(func() {
			<-start
			errs[i] = s.Save(2026, 12, team, l)
		})
	}

	close(start)
	wg.Wait()

	var csvs []string
	for _, name := range weekFiles(t, mount) {
		if filepath.Ext(name) == ".csv" {
			csvs = append(csvs, name)
		}
	}
	if len(csvs) != 2 {
		t.Errorf("week holds %v, want bojjaes.csv and one opponent", csvs)
	}
	for _, err := range errs {
		var set *OpponentSetError
		if err != nil && !errors.As(err, &set) {
			t.Errorf("refused save got %v, want *OpponentSetError", err)
		}
	}
}
