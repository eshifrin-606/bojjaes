package score

import "testing"

func TestPoints(t *testing.T) {
	tests := []struct {
		name string
		in   StatLine
		want float64
	}{
		{
			name: "below every yardage threshold",
			in:   StatLine{RecYd: 60, RushYd: 20},
			want: 0,
		},
		{
			name: "receiving increments are floored",
			in:   StatLine{RecYd: 99},
			want: 3.5,
		},
		{
			// Not two separate 3-point awards: one award, from the clause
			// that pays best — here the 160 combined yards.
			name: "yardage bonus awarded once",
			in:   StatLine{RushYd: 80, RecYd: 80},
			want: 6.0,
		},
		{
			// Receiving pays 3 + two increments for the 25 yards past 80,
			// beating the combined clause's 3 for 5 yards past 100.
			name: "best qualifying clause wins",
			in:   StatLine{RecYd: 105},
			want: 4.0,
		},
		{
			name: "touchdowns and the long-touchdown bonus",
			in:   StatLine{RecTD: 2, TD40Plus: 1},
			want: 13,
		},
		{
			name: "fumble lost",
			in:   StatLine{RecYd: 85, FumLost: 1},
			want: 0,
		},
		{
			// The case above lands on 0, which a missing penalty would also
			// produce. This one pins the -3 to a distinguishable number.
			name: "fumble lost against an increment",
			in:   StatLine{RecYd: 90, FumLost: 1},
			want: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Points(tt.in); got != tt.want {
				t.Errorf("Points(%+v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
