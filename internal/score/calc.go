package score

// Points computes HMFFL fantasy points for a stat line. Only the
// rushing/receiving rules of docs/scoring.md are implemented so far.
func Points(s StatLine) float64 {
	pts := yardageBonus(s)
	pts += 6 * float64(s.RushTD+s.RecTD)
	pts += float64(s.TD40Plus)
	pts -= 3 * float64(s.FumLost)
	return pts
}

// yardageBonus awards at most once, even when several thresholds are cleared,
// so the best-paying one wins. Increments count from that threshold — the
// reading the spreadsheet takes for the 80/80 case.
func yardageBonus(s StatLine) float64 {
	return max(
		thresholdBonus(s.RushYd, 80),
		thresholdBonus(s.RecYd, 80),
		thresholdBonus(s.RushYd+s.RecYd, 100),
	)
}

func thresholdBonus(yards, threshold int) float64 {
	if yards < threshold {
		return 0
	}
	return 3 + 0.5*float64((yards-threshold)/10)
}
