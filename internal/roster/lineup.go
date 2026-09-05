package roster

// lineupSize is the league's starting lineup size: the first lineupSize
// records in a roster are starters, and everything after is bench.
const lineupSize = 9

// Roster is a parsed, ordered set of records with the starters/bench split
// layered on top as a view — it never re-parses or reorders the records.
//
// The split is purely positional: a roster carries no position column, so
// moving a record above the ninth line promotes it and demotes whatever it
// displaced. That is intended behaviour of a hand-maintained lineup card,
// not an accident of the split.
type Roster struct {
	records []Record
}

// Starters returns the roster's starting lineup, in file order.
func (r Roster) Starters() []Record {
	if len(r.records) < lineupSize {
		return r.records
	}
	return r.records[:lineupSize]
}

// Bench returns the roster's bench, in file order and non-nil even when
// empty.
func (r Roster) Bench() []Record {
	if len(r.records) < lineupSize {
		return []Record{}
	}
	return r.records[lineupSize:]
}
