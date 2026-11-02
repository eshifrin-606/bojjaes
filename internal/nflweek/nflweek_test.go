package nflweek

import "testing"

func TestRegularSeasonWeekIsSettable(t *testing.T) {
	season, week, ok := New(2026, 9, Regular).Settable()

	if season != 2026 || week != 9 || !ok {
		t.Errorf("Settable() = %d, %d, %t; want 2026, 9, true", season, week, ok)
	}
}

func TestLastNFLRegularSeasonWeekIsSettable(t *testing.T) {
	if _, week, ok := New(2026, 18, Regular).Settable(); week != 18 || !ok {
		t.Errorf("Settable() = week %d, %t; want 18, true", week, ok)
	}
}

func TestWeekZeroIsNotSettable(t *testing.T) {
	if season, week, ok := New(2026, 0, Regular).Settable(); ok {
		t.Errorf("Settable() = %d, %d, true; want not settable", season, week)
	}
}

func TestOutsideTheRegularSeasonNothingIsSettable(t *testing.T) {
	for _, tt := range []struct {
		name  string
		phase Phase
	}{
		{"preseason", Preseason},
		{"postseason", Postseason},
		{"offseason", Offseason},
		{"unrecognised", Other},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if season, week, ok := New(2026, 1, tt.phase).Settable(); ok {
				t.Errorf("Settable() = %d, %d, true; want not settable", season, week)
			}
		})
	}
}
