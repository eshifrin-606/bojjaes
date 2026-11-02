package lineup

// StarterCount is the league's starting lineup size.
const StarterCount = len(slotOrder)

type Slot string

const (
	QB   Slot = "QB"
	RB   Slot = "RB"
	WR   Slot = "WR"
	TE   Slot = "TE"
	FLEX Slot = "FLEX"
	K    Slot = "K"
	D    Slot = "D"
)

var slotOrder = [9]Slot{QB, RB, WR, TE, FLEX, FLEX, K, D, D}

var slotAccepts = map[Slot][]string{
	QB:   {"QB"},
	RB:   {"RB", "FB"},
	WR:   {"WR"},
	TE:   {"TE"},
	FLEX: {"RB", "FB", "WR", "TE"},
	K:    {"K"},
	D:    {"DL", "DE", "DT", "NT", "LB", "OLB", "ILB", "DB", "CB", "S", "SS", "FS"},
}
