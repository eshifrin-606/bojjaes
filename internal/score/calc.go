package score

// Points computes HMFFL fantasy points for a stat line, implementing the
// rushing/receiving subset of docs/scoring.md.
func Points(s StatLine) float64 {
	pts := yardageBonus(s)
	pts += 6 * float64(s.RushTD+s.RecTD)
	pts += float64(s.TD40Plus)
	pts -= 3 * float64(s.FumLost)
	return pts
}

// yardageBonus pays 3 points for clearing a yardage threshold, plus 0.5 for
// each full 10 yards past it.
//
// Three paths qualify — 80 rushing, 80 receiving, 100 combined — but the award
// is made at most once, so we take the best of them. Increments count from the
// threshold of whichever clause is paying, which is the reading the 80/80 case
// takes in the spreadsheet.
func yardageBonus(s StatLine) float64 {
	return max(
		clause(s.RushYd, 80),
		clause(s.RecYd, 80),
		clause(s.RushYd+s.RecYd, 100),
	)
}

// clause returns the award for one qualifying path, or 0 if yards falls short
// of the threshold.
func clause(yards, threshold int) float64 {
	if yards < threshold {
		return 0
	}
	return 3 + 0.5*float64((yards-threshold)/10)
}
