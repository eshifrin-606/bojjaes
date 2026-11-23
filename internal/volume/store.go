package volume

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/eshifrin/bojjaes/internal/lineup"
)

// Store writes lineups into the lineup tree on the mount.
type Store struct {
	mount string
	// writeLineup lets a test fail a write partway.
	writeLineup func(lineup.Lineup, io.Writer) error
	// mu holds the opponent check and the write together, so concurrent
	// saves of two opponents cannot both see an empty slot.
	mu sync.Mutex
}

func NewStore(mount string) *Store {
	return &Store{mount: mount, writeLineup: lineup.Lineup.WriteCSV}
}

// OpponentSetError refuses a save because the week already holds a different
// opponent for the Bojjaes.
type OpponentSetError struct {
	Existing string
}

func (e *OpponentSetError) Error() string {
	return fmt.Sprintf("bojjaes' opponent is already %s", e.Existing)
}

func (s *Store) Save(season, week int, team string, l lineup.Lineup) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	dir := filepath.Join(s.mount, lineupsDir, strconv.Itoa(season), strconv.Itoa(week))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if team != lineup.OurTeam {
		existing, err := opponent(dir)
		if err != nil {
			return err
		}
		if existing != "" && existing != team {
			return &OpponentSetError{Existing: existing}
		}
	}
	return s.writeAtomically(dir, team, l)
}

// writeAtomically renames a finished temp file over the target, so a reader
// sees the old lineup or the new one, never part of one. The dot prefix keeps
// a temp file left by a crash out of the matchup and the opponent rule.
func (s *Store) writeAtomically(dir, team string, l lineup.Lineup) (err error) {
	f, err := os.CreateTemp(dir, "."+team+".csv.*")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			f.Close()
			os.Remove(f.Name())
		}
	}()
	if err := s.writeLineup(l, f); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), filepath.Join(dir, team+".csv"))
}

// opponent names the week's lineup that isn't the Bojjaes', or "" for none.
func opponent(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		team, ok := strings.CutSuffix(e.Name(), ".csv")
		if ok && !strings.HasPrefix(team, ".") && team != lineup.OurTeam {
			return team, nil
		}
	}
	return "", nil
}
