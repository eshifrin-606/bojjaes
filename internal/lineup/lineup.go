package lineup

// starterCount is the league's starting lineup size: the first starterCount
// records in a lineup are starters, and everything after is bench.
const starterCount = 9

// Lineup is a parsed, ordered set of records with the starters/bench split
// layered on top as a view — it never re-parses or reorders the records.
//
// The split is purely positional: a lineup carries no position column, so
// moving a record above the ninth line promotes it and demotes whatever it
// displaced. That is intended behaviour of a hand-maintained lineup card,
// not an accident of the split.
type Lineup struct {
	records []Record
}

func newLineup(records []Record) Lineup {
	split := benchStart(records)
	// Starters and bench are resolved separately so a bench player sharing a
	// short name never pushes a starter back to their full name.
	fillShortNames(records[:split])
	fillShortNames(records[split:])
	return Lineup{records: records}
}

func benchStart(records []Record) int {
	return min(len(records), starterCount)
}

// Starters returns the lineup's starters, in file order.
func (l Lineup) Starters() []Record {
	return l.records[:benchStart(l.records)]
}

// Bench returns the lineup's bench, in file order and non-nil even when
// empty.
func (l Lineup) Bench() []Record {
	bench := l.records[benchStart(l.records):]
	if bench == nil {
		return []Record{}
	}
	return bench
}
