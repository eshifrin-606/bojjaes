package games

import (
	"testing"
	"time"
)

var detAtAri = Game{
	Home:    "ARI",
	Away:    "DET",
	State:   Complete,
	Kickoff: time.Date(2026, 10, 11, 20, 25, 0, 0, time.UTC),
	Period:  "F",
	Status:  "complete",
}

func TestAHomeTeamFindsItsGame(t *testing.T) {
	week := NewWeek(2026, 6, []Game{detAtAri})

	got, ok := week.ForTeam("ARI")
	if !ok {
		t.Fatal("ForTeam(ARI) reported no game")
	}
	if got != detAtAri {
		t.Errorf("ForTeam(ARI) = %+v, want %+v", got, detAtAri)
	}
}

func TestAnAwayTeamFindsItsGame(t *testing.T) {
	week := NewWeek(2026, 6, []Game{detAtAri})

	got, ok := week.ForTeam("DET")
	if !ok {
		t.Fatal("ForTeam(DET) reported no game")
	}
	if got != detAtAri {
		t.Errorf("ForTeam(DET) = %+v, want %+v", got, detAtAri)
	}
}

func TestATeamOutsideTheWeekHasNoGame(t *testing.T) {
	week := NewWeek(2026, 6, []Game{detAtAri})

	if got, ok := week.ForTeam("KC"); ok {
		t.Errorf("ForTeam(KC) = %+v, want no game", got)
	}
}

func TestTheTeamMatchIsCaseSensitive(t *testing.T) {
	week := NewWeek(2026, 6, []Game{detAtAri})

	if got, ok := week.ForTeam("ari"); ok {
		t.Errorf("ForTeam(ari) = %+v, want no game", got)
	}
}

func TestAWeekBuiltFromNoGamesIsEmpty(t *testing.T) {
	week := NewWeek(2026, 19, nil)

	if !week.Empty() {
		t.Error("Empty() = false for a week with no games, want true")
	}
}

func TestAWeekWithAGameIsNotEmpty(t *testing.T) {
	week := NewWeek(2026, 6, []Game{detAtAri})

	if week.Empty() {
		t.Error("Empty() = true for a week with a game, want false")
	}
}
