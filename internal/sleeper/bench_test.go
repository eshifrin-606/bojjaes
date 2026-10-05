package sleeper

import (
	"bytes"
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

	var fixture []weeklyRow
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decoding fixture: %v", err)
	}

	grown := make([]weeklyRow, 0, realWeekPlayers)
	for len(grown) < realWeekPlayers {
		for _, row := range fixture {
			if len(grown) >= realWeekPlayers {
				break
			}
			grown = append(grown, weeklyRow{PlayerID: row.PlayerID + "-" + strconv.Itoa(len(grown)), Stats: row.Stats})
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
	weekly, err := decodeWeekly(bytes.NewReader(grownPayload(b)))
	if err != nil {
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
		if _, err := decodeWeekly(bytes.NewReader(body)); err != nil {
			b.Fatalf("decoding: %v", err)
		}
	}
}
