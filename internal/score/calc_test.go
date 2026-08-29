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
			// Not two 3-point awards: one, from the best-paying threshold —
			// here the 160 combined yards.
			name: "yardage bonus awarded once",
			in:   StatLine{RushYd: 80, RecYd: 80},
			want: 6.0,
		},
		{
			// Receiving pays 3 + two increments for the 25 yards past 80,
			// beating the combined threshold's 3 for 5 yards past 100.
			name: "best qualifying threshold wins",
			in:   StatLine{RecYd: 105},
			want: 4.0,
		},
		{
			name: "below the passing yardage threshold",
			in:   StatLine{PassYd: 199},
			want: 0,
		},
		{
			name: "passing increments are floored",
			in:   StatLine{PassYd: 249},
			want: 3,
		},
		{
			name: "passing increment earned exactly at the boundary",
			in:   StatLine{PassYd: 250},
			want: 4,
		},
		{
			// 299 and 300 together pin the floor at the second increment, not
			// only at the first.
			name: "passing increment is floored past the first increment",
			in:   StatLine{PassYd: 299},
			want: 4,
		},
		{
			name: "second passing increment",
			in:   StatLine{PassYd: 300},
			want: 5,
		},
		{
			// The two awards are earned on different plays, so they stack.
			// Folding passing into yardageBonus's max() would pay 4 here.
			name: "passing and rushing yardage awards both apply",
			in:   StatLine{PassYd: 250, RushYd: 80},
			want: 7,
		},
		{
			// The same line as "yardage bonus awarded once" above, restated to
			// pin that a line with no passing production is untouched by the
			// passing terms.
			name: "no passing production is unaffected",
			in:   StatLine{RushYd: 80, RecYd: 80},
			want: 6.0,
		},
		{
			name: "touchdowns and the long-touchdown bonus",
			in:   StatLine{RecTD: 2, TD40Plus: 1},
			want: 13,
		},
		{
			name: "passing touchdowns",
			in:   StatLine{PassTD: 2},
			want: 12,
		},
		{
			name: "long passing touchdown bonus",
			in:   StatLine{PassTD: 2, TD40Plus: 1},
			want: 13,
		},
		{
			name: "interceptions thrown",
			in:   StatLine{PassInt: 3},
			want: -9,
		},
		{
			// -9 above could also be produced by a missing penalty landing on
			// some other zero; this pins the penalty to a distinguishable
			// number against a scoring line.
			name: "interception against a scoring line",
			in:   StatLine{PassYd: 250, PassInt: 1},
			want: 1,
		},
		{
			name: "two-point conversion",
			in:   StatLine{TwoPt: 1},
			want: 2,
		},
		{
			name: "two conversions in a week",
			in:   StatLine{TwoPt: 2},
			want: 4,
		},
		{
			// Half-sack credit is the first fractional stat; 1.5 pins that it
			// is scaled rather than rounded to a whole sack either way.
			name: "half a sack",
			in:   StatLine{Sack: 0.5},
			want: 1.5,
		},
		{
			// Not the half-sack case doubled: 1.5 pays its proportional share
			// rather than being tabulated up or down to a whole sack.
			name: "one and a half sacks",
			in:   StatLine{Sack: 1.5},
			want: 4.5,
		},
		{
			name: "two sacks",
			in:   StatLine{Sack: 2},
			want: 6,
		},
		{
			name: "fumble lost",
			in:   StatLine{RecYd: 85, FumLost: 1},
			want: 0,
		},
		{
			// The case above lands on 0, which a missing penalty would also
			// produce; this one pins -3 to a distinguishable number.
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
