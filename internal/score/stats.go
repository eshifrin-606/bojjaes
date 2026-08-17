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

	RushYd int `json:"rush_yd"`
	RecYd  int `json:"rec_yd"`

	RushTD int `json:"rush_td"`
	RecTD  int `json:"rec_td"`

	// TD40Plus counts rushing and receiving touchdowns of 40+ yards.
	TD40Plus int `json:"td_40_plus"`

	FumLost int `json:"fum_lost"`
}
