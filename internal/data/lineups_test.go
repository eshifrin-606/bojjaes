package data_test

import (
	"fmt"
	"io/fs"
	"strconv"
	"testing"

	"github.com/eshifrin/bojjaes/internal/data"
	"github.com/eshifrin/bojjaes/internal/lineup"
)

// The embedded tree is the one the server serves, so it has to satisfy the
// reader whole: a week resolves to its two teams and both lineups parse.
func TestEmbeddedTreeResolvesAWeek(t *testing.T) {
	tree := lineup.New(data.Lineups)

	ours, theirs, err := tree.Matchup(2025, 15)
	if err != nil {
		t.Fatalf("Matchup: %v", err)
	}
	if ours != "bojjaes" || theirs != "aroma" {
		t.Errorf("Matchup() = %q, %q, want %q, %q", ours, theirs, "bojjaes", "aroma")
	}

	for _, team := range []string{ours, theirs} {
		got, err := tree.Read(2025, 15, team)
		if err != nil {
			t.Fatalf("Read(%q): %v", team, err)
		}
		if len(got.Starters()) != lineup.StarterCount {
			t.Errorf("%s starters = %d, want %d", team, len(got.Starters()), lineup.StarterCount)
		}
	}
}

// The unit tests all run on fstest.MapFS, so nothing else reads the real
// files: a malformed CSV, a roster that cannot fill the nine slots, or a week
// with three lineups in it would build clean and fail at request time. This
// walk is a test of the data, not the reader.
func TestEveryEmbeddedWeekIsAReadableMatchup(t *testing.T) {
	tree := lineup.New(data.Lineups)

	seasons, err := fs.ReadDir(data.Lineups, ".")
	if err != nil {
		t.Fatalf("reading the embedded tree: %v", err)
	}
	if len(seasons) == 0 {
		t.Fatal("the embedded tree holds no seasons")
	}

	for _, seasonEntry := range seasons {
		season, err := strconv.Atoi(seasonEntry.Name())
		if err != nil {
			t.Errorf("season directory %q is not a year", seasonEntry.Name())
			continue
		}

		weeks, err := fs.ReadDir(data.Lineups, seasonEntry.Name())
		if err != nil {
			t.Errorf("reading season %d: %v", season, err)
			continue
		}

		for _, weekEntry := range weeks {
			week, err := strconv.Atoi(weekEntry.Name())
			if err != nil {
				t.Errorf("week directory %s/%s is not a number", seasonEntry.Name(), weekEntry.Name())
				continue
			}

			t.Run(fmt.Sprintf("%d/%d", season, week), func(t *testing.T) {
				ours, theirs, err := tree.Matchup(season, week)
				if err != nil {
					t.Fatalf("Matchup: %v", err)
				}
				for _, team := range []string{ours, theirs} {
					if _, err := tree.Read(season, week, team); err != nil {
						t.Errorf("Read(%q): %v", team, err)
					}
				}
			})
		}
	}
}
