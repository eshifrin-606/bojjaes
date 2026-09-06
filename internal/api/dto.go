package api

import "github.com/eshifrin/bojjaes/internal/score"

// ScoreResponse echoes the stats alongside the total, so a wrong player or a
// bad week shows in the output instead of hiding behind a plausible number.
type ScoreResponse struct {
	Stats  score.StatLine `json:"stats"`
	Points float64        `json:"points"`
}

// BatchRequest asks for one season and week. The season type is implicitly
// regular — the league scores regular-season play.
type BatchRequest struct {
	Season    int      `json:"season"`
	Week      int      `json:"week"`
	PlayerIDs []string `json:"player_ids"`
}

// BatchResponse splits scored players from players the payload had nothing for.
//
// The split is what keeps the two apart: NoStats is a list of IDs with nowhere
// to put a number, so an absent player cannot acquire a point total. Points
// carries no omitempty for the mirror-image reason — a real 0.0 must survive.
type BatchResponse struct {
	Season  int             `json:"season"`
	Week    int             `json:"week"`
	Scores  []ScoreResponse `json:"scores"`
	NoStats []string        `json:"no_stats"`
}

// newBatchResponse starts both buckets non-nil so they serialize as `[]`
// rather than `null`.
func newBatchResponse(season, week int) BatchResponse {
	return BatchResponse{
		Season:  season,
		Week:    week,
		Scores:  []ScoreResponse{},
		NoStats: []string{},
	}
}
