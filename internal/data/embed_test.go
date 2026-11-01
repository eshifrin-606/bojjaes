package data

import (
	"io/fs"
	"testing"
)

// The embedded trees start below the directory the embed directive named: a
// reader over them has no prefix to strip.
func TestLineupsStartAtTheSeasons(t *testing.T) {
	name := "2025/15/bojjaes.csv"

	if _, err := fs.Stat(Lineups, name); err != nil {
		t.Errorf("Stat(%q): %v", name, err)
	}
}

func TestRostersStartAtTheTeamFiles(t *testing.T) {
	name := "bojjaes.csv"

	if _, err := fs.Stat(Rosters, name); err != nil {
		t.Errorf("Stat(%q): %v", name, err)
	}
}
