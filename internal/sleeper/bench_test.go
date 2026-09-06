package sleeper

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"

	"github.com/eshifrin/bojjaes/internal/score"
)

// The fixture is a trimmed 11-player slice; a real week is a few thousand
// entries. grownPayload repeats the fixture under synthetic IDs so the eager
// transform is measured against the size it will actually see.
const realWeekPlayers = 3000

func grownPayload(t testing.TB) []byte {
	t.Helper()

	raw, err := os.ReadFile("testdata/week14.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	var fixture map[string]map[string]float64
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decoding fixture: %v", err)
	}

	grown := make(map[string]map[string]float64, realWeekPlayers)
	for len(grown) < realWeekPlayers {
		for id, stats := range fixture {
			if len(grown) >= realWeekPlayers {
				break
			}
			grown[id+"-"+strconv.Itoa(len(grown))] = stats
		}
	}

	body, err := json.Marshal(grown)
	if err != nil {
		t.Fatalf("re-encoding: %v", err)
	}
	return body
}

// BenchmarkTransform isolates the mapping from the fetch and the decode: this
// is the cost the eager WeekStats pays that a lazy one would not.
func BenchmarkTransform(b *testing.B) {
	var weekly map[string]map[string]float64
	if err := json.Unmarshal(grownPayload(b), &weekly); err != nil {
		b.Fatalf("decoding: %v", err)
	}

	for b.Loop() {
		players := make(map[string]score.StatLine, len(weekly))
		for playerID := range weekly {
			line, ok := statLineFrom(weekly, playerID, 2025, 14)
			if !ok {
				continue
			}
			players[playerID] = line
		}
	}
}

// BenchmarkDecode is the comparison: the JSON decode that produced the
// transform's input, on the same payload.
func BenchmarkDecode(b *testing.B) {
	body := grownPayload(b)

	for b.Loop() {
		var weekly map[string]map[string]float64
		if err := json.Unmarshal(body, &weekly); err != nil {
			b.Fatalf("decoding: %v", err)
		}
	}
}
