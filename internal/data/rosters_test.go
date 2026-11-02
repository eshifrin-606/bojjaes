package data_test

import (
	"slices"
	"testing"

	"github.com/eshifrin/bojjaes/internal/data"
	"github.com/eshifrin/bojjaes/internal/lineup"
)

// Like the lineup walk, this is a test of the data: the unit tests never read
// the real roster files.
func TestEmbeddedRostersAreTheLeaguesTenAndAllRead(t *testing.T) {
	rosters := lineup.NewRosters(data.Rosters)

	teams := rosters.Teams()
	want := []string{"aroma", "bojjaes", "fuego", "gonads", "jabs", "renegades", "saban", "trogs", "vampyres", "wood"}
	if !slices.Equal(teams, want) {
		t.Errorf("Teams() = %q, want %q", teams, want)
	}

	for _, team := range teams {
		if _, err := rosters.Read(team); err != nil {
			t.Errorf("Read(%q): %v", team, err)
		}
	}
}
