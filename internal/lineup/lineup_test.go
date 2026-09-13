package lineup

import (
	"strings"
	"testing"
)

func recordsN(n int) []Record {
	records := make([]Record, n)
	for i := range records {
		records[i] = Record{ID: string(rune('a' + i))}
	}
	return records
}

func TestLineupSplit(t *testing.T) {
	tests := []struct {
		name         string
		records      []Record
		wantStarters int
		wantBench    int
	}{
		{
			name:         "twelve records split at nine",
			records:      recordsN(12),
			wantStarters: 9,
			wantBench:    3,
		},
		{
			name:         "lineup shorter than the starting nine is all starters",
			records:      recordsN(5),
			wantStarters: 5,
			wantBench:    0,
		},
		{
			// Boundary case: kept even though it passed on the first run,
			// since the split's edge is exactly where an off-by-one hides.
			name:         "lineup of exactly nine has an empty bench",
			records:      recordsN(9),
			wantStarters: 9,
			wantBench:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Lineup{records: tt.records}
			starters, bench := r.Starters(), r.Bench()

			if len(starters) != tt.wantStarters {
				t.Errorf("len(Starters()) = %d, want %d", len(starters), tt.wantStarters)
			}
			if len(bench) != tt.wantBench {
				t.Errorf("len(Bench()) = %d, want %d", len(bench), tt.wantBench)
			}
			if bench == nil {
				t.Errorf("Bench() = nil, want non-nil")
			}
			for i, rec := range starters {
				if rec != tt.records[i] {
					t.Errorf("Starters()[%d] = %v, want %v", i, rec, tt.records[i])
				}
			}
			for i, rec := range bench {
				if rec != tt.records[tt.wantStarters+i] {
					t.Errorf("Bench()[%d] = %v, want %v", i, rec, tt.records[tt.wantStarters+i])
				}
			}
		})
	}
}

// TestLineupSplitAfterParse pins parsing and splitting together: comment and
// blank lines must not consume a starter slot, which only shows up once the
// split runs against parser output rather than a hand-built []Record.
func TestLineupSplitAfterParse(t *testing.T) {
	input := `# starting nine
id,name,position,team

1,Alpha,QB,BUF
2,Bravo,RB,BUF
# midway comment
3,Charlie,RB,BUF
4,Delta,WR,BUF

5,Echo,WR,BUF
6,Foxtrot,WR,BUF
7,Golf,TE,BUF
8,Hotel,K,BUF
9,India,LB,BUF
`
	records, err := parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	r := Lineup{records: records}
	if got := len(r.Starters()); got != 9 {
		t.Errorf("len(Starters()) = %d, want 9", got)
	}
	if got := len(r.Bench()); got != 0 {
		t.Errorf("len(Bench()) = %d, want 0", got)
	}
}

// A position is the player's listed position, not the slot he fills, so an
// illegal lineup still splits by file order alone.
func TestLineupSplitIgnoresPosition(t *testing.T) {
	input := header +
		"1,Q1,QB,BUF\n2,Q2,QB,BUF\n3,Q3,QB,BUF\n4,Q4,QB,BUF\n5,Q5,QB,BUF\n" +
		"6,Q6,QB,BUF\n7,Q7,QB,BUF\n8,Q8,QB,BUF\n9,Q9,QB,BUF\n10,Kicker,K,BUF\n"
	records, err := parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	r := Lineup{records: records}
	starters, bench := r.Starters(), r.Bench()
	if len(starters) != 9 {
		t.Errorf("len(Starters()) = %d, want 9", len(starters))
	}
	for i, rec := range starters {
		if rec.Position != "QB" {
			t.Errorf("Starters()[%d].Position = %q, want QB", i, rec.Position)
		}
	}
	if len(bench) != 1 || bench[0].Position != "K" {
		t.Errorf("Bench() = %+v, want only the kicker", bench)
	}
}

// TestLineupSplitReordered documents that moving the tenth record above the
// ninth promotes it and demotes the record it displaced: intended behaviour
// of a hand-maintained lineup card, not an accident of the split.
func TestLineupSplitReordered(t *testing.T) {
	base := recordsN(10)
	reordered := make([]Record, len(base))
	copy(reordered, base)
	// Swap the tenth record (index 9) above the ninth (index 8).
	reordered[8], reordered[9] = reordered[9], reordered[8]

	r := Lineup{records: reordered}
	starters, bench := r.Starters(), r.Bench()

	if got, want := starters[8], base[9]; got != want {
		t.Errorf("Starters()[8] = %v, want %v (the promoted record)", got, want)
	}
	if got, want := bench[0], base[8]; got != want {
		t.Errorf("Bench()[0] = %v, want %v (the demoted record)", got, want)
	}
}
