package lineup

import (
	"fmt"
	"io/fs"
	"strconv"
	"testing"
)

// The embedded tree starts at the seasons, not at the directory the embed
// directive named: a Tree over it has no prefix to strip.
func TestEmbeddedTreeStartsAtTheSeasons(t *testing.T) {
	name := "2025/15/bojjaes.csv"

	if _, err := fs.Stat(Embedded, name); err != nil {
		t.Errorf("Stat(%q): %v", name, err)
	}
}

// The embedded tree is the one the server serves, so it has to satisfy the
// reader whole: a week resolves to its two teams and both lineups parse.
func TestEmbeddedTreeResolvesAWeek(t *testing.T) {
	tree := New(Embedded)

	ours, theirs, err := tree.Matchup(2025, 15)
	if err != nil {
		t.Fatalf("Matchup: %v", err)
	}
	if ours != "bojjaes" || theirs != "aroma" {
		t.Errorf("Matchup() = %q, %q, want %q, %q", ours, theirs, "bojjaes", "aroma")
	}

	for _, team := range []string{ours, theirs} {
		lineup, err := tree.Read(2025, 15, team)
		if err != nil {
			t.Fatalf("Read(%q): %v", team, err)
		}
		if len(lineup.Starters()) != starterCount {
			t.Errorf("%s starters = %d, want %d", team, len(lineup.Starters()), starterCount)
		}
	}
}

// The unit tests all run on fstest.MapFS, so nothing else reads the real
// files: a malformed CSV or a week with three lineups in it would build clean
// and fail at request time. This walk is a test of the data, not the reader.
func TestEveryEmbeddedWeekIsAReadableMatchup(t *testing.T) {
	tree := New(Embedded)

	seasons, err := fs.ReadDir(Embedded, ".")
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

		weeks, err := fs.ReadDir(Embedded, seasonEntry.Name())
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
