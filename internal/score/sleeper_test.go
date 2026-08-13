package score

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fixtureServer serves testdata/week14.json at the path FetchStatLine is
// expected to request, and fails the test on any other path.
func fixtureServer(t *testing.T) *httptest.Server {
	t.Helper()

	const wantPath = "/v1/stats/nfl/regular/2025/14"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != wantPath {
			t.Errorf("requested %q, want %q", r.URL.Path, wantPath)
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "testdata/week14.json")
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestFetchStatLinePlayerAbsent(t *testing.T) {
	srv := fixtureServer(t)

	_, err := FetchStatLine(srv.URL, 2025, 14, "does-not-exist")
	if err == nil {
		t.Fatal("FetchStatLine returned no error for an absent player; a silent zero score is exactly the failure this guards")
	}

	// The message has to identify what was looked up, or a rotted hardcoded
	// ID is indistinguishable from a bad week.
	for _, want := range []string{"does-not-exist", "2025", "14"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

func TestFetchStatLine(t *testing.T) {
	srv := fixtureServer(t)

	tests := []struct {
		name     string
		playerID string
		want     StatLine
	}{
		{
			// Nacua's real week 14 line: 7 catches for 167 and 2 TDs, no
			// carries. rush_yd, fum_lost and the 40+ TD keys are all absent
			// from his fixture entry, so this also covers "missing stat key
			// reads as zero" (task 4.3).
			name:     "player present in the weekly payload",
			playerID: NacuaPlayerID,
			want: StatLine{
				PlayerID: "9493",
				Season:   2025,
				Week:     14,
				RecYd:    167,
				RecTD:    2,
			},
		},
		{
			// James Cook, the fixture's second entry — carries rush_yd and
			// fum_lost, which Nacua's entry lacks. If the transform indexed
			// the wrong player, the case above would not look like this.
			name:     "a different player in the same payload",
			playerID: "8138",
			want: StatLine{
				PlayerID: "8138",
				Season:   2025,
				Week:     14,
				RushYd:   80,
				RecYd:    31,
				FumLost:  1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FetchStatLine(srv.URL, 2025, 14, tt.playerID)
			if err != nil {
				t.Fatalf("FetchStatLine: %v", err)
			}
			if got != tt.want {
				t.Errorf("FetchStatLine = %+v, want %+v", got, tt.want)
			}
		})
	}
}
