package score

// Points computes HMFFL fantasy points for a stat line. The passing,
// rushing/receiving, and two-point rules of docs/scoring.md are implemented;
// kicking and defensive scoring are not.
//
// It reads stats, never positions, so a category is added as another term
// rather than as a branch.
func Points(s StatLine) float64 {
	pts := yardageBonus(s)
	pts += thresholdBonus(s.PassYd, 200, 50, 1)
	pts += 6 * float64(s.PassTD+s.RushTD+s.RecTD)
	pts += float64(s.TD40Plus)
	pts += 2 * float64(s.TwoPt)
	pts -= 3 * float64(s.PassInt+s.FumLost)
	return pts
}

// yardageBonus awards at most once, even when several thresholds are cleared,
// so the best-paying one wins. Increments count from that threshold — the
// reading the spreadsheet takes for the 80/80 case.
func yardageBonus(s StatLine) float64 {
	return max(
		thresholdBonus(s.RushYd, 80, 10, 0.5),
		thresholdBonus(s.RecYd, 80, 10, 0.5),
		thresholdBonus(s.RushYd+s.RecYd, 100, 10, 0.5),
	)
}

// thresholdBonus is shared by the passing and rushing/receiving tables, which
// pay 3 at their threshold and differ only in the size and value of the
// increments past it. The integer division is the league's floor rule, and it
// lives here once so the two tables cannot drift apart in how they round.
func thresholdBonus(yards, threshold, increment int, perIncrement float64) float64 {
	if yards < threshold {
		return 0
	}
	return 3 + perIncrement*float64((yards-threshold)/increment)
}
