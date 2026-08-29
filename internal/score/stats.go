package score

// StatLine is one player's NFL production for one week, in provider-neutral
// terms. Sleeper's stat keys stay behind statLineFrom.
//
// Fantasy points are deliberately absent: a stat line cannot carry a stale
// total if it never carries a total. See Points.
type StatLine struct {
	PlayerID string `json:"player_id"`
	Season   int    `json:"season"`
	Week     int    `json:"week"`

	PassYd int `json:"pass_yd"`
	RushYd int `json:"rush_yd"`
	RecYd  int `json:"rec_yd"`

	PassTD int `json:"pass_td"`
	RushTD int `json:"rush_td"`
	RecTD  int `json:"rec_td"`

	// TD40Plus counts passing, rushing, and receiving touchdowns of 40+ yards.
	// The bonus is a flat point regardless of how the touchdown was scored, so
	// the flavors are summed rather than tracked apart. One long touchdown pass
	// pays both the passer and the receiver, on their own stat lines.
	TD40Plus int `json:"td_40_plus"`

	// TwoPt counts two-point conversions in every flavor — thrown, run in, or
	// caught. The rules pay 2 for each regardless, and one conversion play
	// pays both the passer and the scorer on their separate stat lines.
	TwoPt int `json:"two_pt"`

	// PassInt is interceptions thrown, never interceptions caught — the
	// penalty belongs to the passer.
	PassInt int `json:"pass_int"`

	FumLost int `json:"fum_lost"`

	// Sack is credited in half-sack granularity, so it is fractional where
	// every other count is whole.
	Sack float64 `json:"sack"`

	// FGMade carries no distance: the league pays a flat rate per field goal,
	// and the 50+ bonus is counted separately.
	FGMade int `json:"fg_made"`

	XPMade int `json:"xp_made"`

	// FG50Plus counts the FGMade kicks that were from 50+ yards, paying a
	// bonus point on top of them under the same Misc clause as TD40Plus.
	FG50Plus int `json:"fg_50_plus"`
}
