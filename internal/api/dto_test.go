package api

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/eshifrin/bojjaes/internal/score"
)

// A player who dressed and recorded nothing scores a real 0.0. An `omitempty`
// on Points would erase it, leaving a scored player indistinguishable from an
// absent one in the JSON.
func TestBatchResponseKeepsGenuineZero(t *testing.T) {
	resp := newBatchResponse(2025, 14)
	resp.Scores = append(resp.Scores, ScoreResponse{
		Stats:  score.StatLine{PlayerID: "7591", Season: 2025, Week: 14},
		Points: 0,
	})

	body, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshalling: %v", err)
	}

	if !strings.Contains(string(body), `"points":0`) {
		t.Errorf("a zero score did not serialize an explicit points field: %s", body)
	}
}

// Absent buckets must read as "nothing here", not as "field missing". A `null`
// forces every caller to nil-check before ranging.
func TestBatchResponseEmptyBucketsAreArrays(t *testing.T) {
	body, err := json.Marshal(newBatchResponse(2026, 1))
	if err != nil {
		t.Fatalf("marshalling: %v", err)
	}

	for _, want := range []string{`"scores":[]`, `"no_stats":[]`} {
		if !strings.Contains(string(body), want) {
			t.Errorf("empty response does not contain %s: %s", want, body)
		}
	}
}
