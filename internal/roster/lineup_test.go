package roster

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

func TestRosterSplit(t *testing.T) {
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
			name:         "roster shorter than a lineup is all starters",
			records:      recordsN(5),
			wantStarters: 5,
			wantBench:    0,
		},
		{
			// Boundary case: kept even though it passed on the first run,
			// since the split's edge is exactly where an off-by-one hides.
			name:         "roster of exactly nine has an empty bench",
			records:      recordsN(9),
			wantStarters: 9,
			wantBench:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Roster{records: tt.records}
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

// TestRosterSplitAfterParse pins parsing and splitting together: comment and
// blank lines must not consume a starter slot, which only shows up once the
// split runs against parser output rather than a hand-built []Record.
func TestRosterSplitAfterParse(t *testing.T) {
	input := `# starting nine

1,Alpha
2,Bravo
# midway comment
3,Charlie
4,Delta

5,Echo
6,Foxtrot
7,Golf
8,Hotel
9,India
`
	records, err := parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	r := Roster{records: records}
	if got := len(r.Starters()); got != 9 {
		t.Errorf("len(Starters()) = %d, want 9", got)
	}
	if got := len(r.Bench()); got != 0 {
		t.Errorf("len(Bench()) = %d, want 0", got)
	}
}

// TestRosterSplitReordered documents that moving the tenth record above the
// ninth promotes it and demotes the record it displaced: intended behaviour
// of a hand-maintained lineup card, not an accident of the split.
func TestRosterSplitReordered(t *testing.T) {
	base := recordsN(10)
	reordered := make([]Record, len(base))
	copy(reordered, base)
	// Swap the tenth record (index 9) above the ninth (index 8).
	reordered[8], reordered[9] = reordered[9], reordered[8]

	r := Roster{records: reordered}
	starters, bench := r.Starters(), r.Bench()

	if got, want := starters[8], base[9]; got != want {
		t.Errorf("Starters()[8] = %v, want %v (the promoted record)", got, want)
	}
	if got, want := bench[0], base[8]; got != want {
		t.Errorf("Bench()[0] = %v, want %v (the demoted record)", got, want)
	}
}
