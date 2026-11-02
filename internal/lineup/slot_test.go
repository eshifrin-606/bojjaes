package lineup

import "testing"

func TestSlotAccepts(t *testing.T) {
	tests := []struct {
		slot     Slot
		position string
		want     bool
	}{
		{RB, "FB", true},
		{FLEX, "TE", true},
		{D, "CB", true},
		{QB, "WR", false},
		{K, "K", true},
	}
	for _, tt := range tests {
		if got := tt.slot.Accepts(tt.position); got != tt.want {
			t.Errorf("%s.Accepts(%q) = %v, want %v", tt.slot, tt.position, got, tt.want)
		}
	}
}
