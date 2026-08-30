package score

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// fixtureServer serves testdata/week14.json, failing the test if the caller
// asks for any path but the expected one.
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

// jsonServer serves one canned body regardless of path.
func jsonServer(t *testing.T, body string) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestFetchWeekly(t *testing.T) {
	srv := fixtureServer(t)

	weekly, err := fetchWeekly(context.Background(), srv.URL, 2025, 14)
	if err != nil {
		t.Fatalf("fetchWeekly: %v", err)
	}

	if got := len(weekly); got != 6 {
		t.Errorf("decoded %d entries, want 6", got)
	}
	if got := weekly[NacuaPlayerID]["rec_yd"]; got != 167 {
		t.Errorf("weekly[%s][rec_yd] = %v, want 167", NacuaPlayerID, got)
	}
}

// An unplayed week returns 200 with an empty object, which is an answer rather
// than a failure.
func TestFetchWeeklyEmptyPayload(t *testing.T) {
	srv := jsonServer(t, `{}`)

	weekly, err := fetchWeekly(context.Background(), srv.URL, 2026, 1)
	if err != nil {
		t.Fatalf("fetchWeekly: %v", err)
	}
	if len(weekly) != 0 {
		t.Errorf("decoded %d entries, want 0", len(weekly))
	}
}

// A stalled read must be interruptible, or a hung upstream pins the handler
// goroutine for the life of the process.
func TestFetchWeeklyContextCancelled(t *testing.T) {
	blocked := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-blocked
	}))
	t.Cleanup(func() {
		close(blocked)
		srv.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := fetchWeekly(ctx, srv.URL, 2025, 14)
		done <- err
	}()

	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("fetchWeekly returned no error after its context was cancelled")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("fetchWeekly ignored context cancellation and blocked on the stalled server")
	}
}

func TestFetchWeeklyUpstreamFailures(t *testing.T) {
	unreachable := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	unreachableURL := unreachable.URL
	unreachable.Close()

	tests := []struct {
		name    string
		baseURL func(t *testing.T) string
	}{
		{
			name:    "transport failure",
			baseURL: func(*testing.T) string { return unreachableURL },
		},
		{
			name: "non-200 status",
			baseURL: func(t *testing.T) string {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					http.Error(w, "boom", http.StatusInternalServerError)
				}))
				t.Cleanup(srv.Close)
				return srv.URL
			},
		},
		{
			name:    "undecodable body",
			baseURL: func(t *testing.T) string { return jsonServer(t, `not json`).URL },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := fetchWeekly(context.Background(), tt.baseURL(t), 2025, 14)
			if err == nil {
				t.Fatal("fetchWeekly returned no error")
			}

			// The message has to identify what was looked up, or a rotted
			// week is indistinguishable from a broken upstream.
			for _, want := range []string{"2025", "14"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not mention %q", err, want)
				}
			}
		})
	}
}

// fixtureWeekly decodes testdata/week14.json into the shape fetchWeekly
// returns, so the per-player mapping can be tested without a server.
func fixtureWeekly(t *testing.T) map[string]map[string]float64 {
	t.Helper()

	body, err := os.ReadFile("testdata/week14.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	var weekly map[string]map[string]float64
	if err := json.Unmarshal(body, &weekly); err != nil {
		t.Fatalf("decoding fixture: %v", err)
	}
	return weekly
}

func TestStatLineFrom(t *testing.T) {
	weekly := fixtureWeekly(t)

	tests := []struct {
		name     string
		playerID string
		want     StatLine
	}{
		{
			// Nacua's real week 14 line: 7 catches for 167 and 2 TDs, no
			// carries. rush_yd, fum_lost and the 40+ TD keys are absent from
			// his fixture entry, so this also covers missing keys reading as
			// zero.
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
			// James Cook, the fixture's second entry, carries rush_yd and
			// fum_lost where Nacua's does not — so indexing the wrong player
			// cannot pass both cases.
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
		{
			// Dak Prescott's real week 14 line, the fixture's quarterback:
			// 376 yards, a 42-yard TD pass, 2 picks and a conversion pass.
			// Every new passing key is asserted here at a non-zero value, so a
			// mistyped key cannot pass by reading zero.
			name:     "quarterback passing stats are mapped",
			playerID: "3294",
			want: StatLine{
				PlayerID: "3294",
				Season:   2025,
				Week:     14,
				PassYd:   376,
				PassTD:   1,
				TD40Plus: 1,
				PassInt:  2,
				TwoPt:    1,
				RushYd:   14,
			},
		},
		{
			// Kyle Monangai ran a conversion in; his entry carries rush_2pt
			// and none of the passing keys.
			name:     "a rushed two-point conversion is mapped",
			playerID: "12534",
			want: StatLine{
				PlayerID: "12534",
				Season:   2025,
				Week:     14,
				RushYd:   57,
				TwoPt:    1,
			},
		},
		{
			// Jake Ferguson caught the conversion Prescott threw — the same
			// Dallas play, which the rules pay to both players. Both stat
			// lines carry a TwoPt of 1.
			name:     "a caught two-point conversion is mapped",
			playerID: "8110",
			want: StatLine{
				PlayerID: "8110",
				Season:   2025,
				Week:     14,
				RecYd:    58,
				TwoPt:    1,
				FumLost:  1,
			},
		},
		{
			// A player who dressed and recorded nothing carries an entry with
			// no mapped stat keys. That is a real 0.0, not an absence.
			name:     "player present but scoreless",
			playerID: "7591",
			want: StatLine{
				PlayerID: "7591",
				Season:   2025,
				Week:     14,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := statLineFrom(weekly, tt.playerID, 2025, 14)
			if !ok {
				t.Fatalf("statLineFrom reported player %q absent", tt.playerID)
			}
			if got != tt.want {
				t.Errorf("statLineFrom = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// pass_rush_yd is passing plus rushing yards combined, not passing yards.
// It appears on non-quarterbacks — the fixture's running backs both carry it
// — so mapping it in place of pass_yd would hand a rusher a passing bonus
// they never earned. It produces a plausible number rather than a failure,
// which is why it gets its own test.
func TestStatLineFromIgnoresCombinedPassRushYards(t *testing.T) {
	weekly := fixtureWeekly(t)

	for _, playerID := range []string{"8138", "12534"} {
		got, ok := statLineFrom(weekly, playerID, 2025, 14)
		if !ok {
			t.Fatalf("statLineFrom reported player %q absent", playerID)
		}
		if raw := weekly[playerID]["pass_rush_yd"]; raw == 0 {
			t.Fatalf("fixture player %q no longer carries pass_rush_yd; this test guards nothing", playerID)
		}
		if got.PassYd != 0 {
			t.Errorf("player %q: PassYd = %d, want 0; pass_rush_yd is not passing yardage", playerID, got.PassYd)
		}
	}
}

// The kicking keys are mapped from a hand-built payload rather than the week
// 14 fixture, which carries no kicker. Each of the three is given a distinct
// value so a key mapped to the wrong field cannot pass by reading zero.
func TestStatLineFromKicking(t *testing.T) {
	weekly := map[string]map[string]float64{
		"kicker": {"fgm": 2, "fgm_50p": 1, "xpm": 4},
	}

	got, ok := statLineFrom(weekly, "kicker", 2025, 14)
	if !ok {
		t.Fatal("statLineFrom reported the kicker absent")
	}

	want := StatLine{
		PlayerID: "kicker",
		Season:   2025,
		Week:     14,
		FGMade:   2,
		FG50Plus: 1,
		XPMade:   4,
	}
	if got != want {
		t.Errorf("statLineFrom = %+v, want %+v", got, want)
	}
}

// A half sack pays 1.5, so the value has to survive the transform as 0.5.
// Reading idp_sack through the integer reader every other stat uses would
// truncate it to 0 and drop the points without failing anything.
func TestStatLineFromHalfSack(t *testing.T) {
	weekly := map[string]map[string]float64{
		"defender": {"idp_sack": 0.5},
	}

	got, ok := statLineFrom(weekly, "defender", 2025, 14)
	if !ok {
		t.Fatal("statLineFrom reported the defender absent")
	}
	if got.Sack != 0.5 {
		t.Errorf("Sack = %v, want 0.5", got.Sack)
	}
}

// The two interception stats sit on different players and pay opposite signs,
// so the mapping has to keep them apart: idp_int is the defender's 6, pass_int
// the passer's -3. Both payloads are asserted in one test because crossing the
// keys is the failure worth catching, and that only shows up as a pair.
func TestStatLineFromInterceptionsAreNotCrossed(t *testing.T) {
	weekly := map[string]map[string]float64{
		"defender": {"idp_int": 1, "idp_def_td": 1},
		"passer":   {"pass_int": 2},
	}

	defender, ok := statLineFrom(weekly, "defender", 2025, 14)
	if !ok {
		t.Fatal("statLineFrom reported the defender absent")
	}
	if defender.IntCaught != 1 {
		t.Errorf("defender IntCaught = %d, want 1", defender.IntCaught)
	}
	if defender.DefTD != 1 {
		t.Errorf("defender DefTD = %d, want 1", defender.DefTD)
	}
	if defender.PassInt != 0 {
		t.Errorf("defender PassInt = %d, want 0; an interception caught is not one thrown", defender.PassInt)
	}

	passer, ok := statLineFrom(weekly, "passer", 2025, 14)
	if !ok {
		t.Fatal("statLineFrom reported the passer absent")
	}
	if passer.PassInt != 2 {
		t.Errorf("passer PassInt = %d, want 2", passer.PassInt)
	}
	if passer.IntCaught != 0 {
		t.Errorf("passer IntCaught = %d, want 0; an interception thrown is not one caught", passer.IntCaught)
	}
}

// pass_int_td sits on the quarterback who *threw* the pick-six, not on the
// defender who scored it. Mapping it as the missing touchdown stat would pay
// the passer 6 for a play the rules dock him 3 — a 9-point swing in the wrong
// direction, and the most expensive mistake available in this transform.
// Nothing maps the key, so this test passes today; it exists to fail loudly if
// that changes.
func TestStatLineFromIgnoresPickSixThrown(t *testing.T) {
	weekly := map[string]map[string]float64{
		"passer": {"pass_int": 1, "pass_int_td": 1},
	}

	got, ok := statLineFrom(weekly, "passer", 2025, 14)
	if !ok {
		t.Fatal("statLineFrom reported the passer absent")
	}
	if pts := Points(got); pts != -3 {
		t.Errorf("Points = %v, want -3; a pick-six thrown is a penalty, never a touchdown scored", pts)
	}
}

// pass_sack sits on the quarterback who was sacked. The 3 points belong to the
// defender who recorded it, whose key is idp_sack. Same shape as the pick-six
// guard above: correct code never maps this key.
func TestStatLineFromIgnoresSacksTaken(t *testing.T) {
	weekly := map[string]map[string]float64{
		"passer": {"pass_sack": 3},
	}

	got, ok := statLineFrom(weekly, "passer", 2025, 14)
	if !ok {
		t.Fatal("statLineFrom reported the passer absent")
	}
	if got.Sack != 0 {
		t.Errorf("Sack = %v, want 0; being sacked is not recording one", got.Sack)
	}
	if pts := Points(got); pts != 0 {
		t.Errorf("Points = %v, want 0", pts)
	}
}

// st_td is the special-teams touchdown, which the rules pay like any other.
// kr_td and pr_td name the same play but sit on team rows, so st_td is the
// only one a player lookup ever sees.
func TestStatLineFromReturnTouchdown(t *testing.T) {
	weekly := map[string]map[string]float64{
		"returner": {"st_td": 1},
	}

	got, ok := statLineFrom(weekly, "returner", 2025, 14)
	if !ok {
		t.Fatal("statLineFrom reported the returner absent")
	}
	if got.ReturnTD != 1 {
		t.Errorf("ReturnTD = %d, want 1", got.ReturnTD)
	}
}

// A turnover-qualified recovery is spread across three keys, and the IDP one
// alone undercounts: it misses special-teams recoveries. The three counts are
// distinct powers of two so a dropped term names itself in the total.
func TestStatLineFromFumbleRecoveriesSumThreeKeys(t *testing.T) {
	weekly := map[string]map[string]float64{
		"defender": {"idp_fum_rec": 1, "st_fum_rec": 2, "def_st_fum_rec": 4},
	}

	got, ok := statLineFrom(weekly, "defender", 2025, 14)
	if !ok {
		t.Fatal("statLineFrom reported the defender absent")
	}
	if got.FumRec != 7 {
		t.Errorf("FumRec = %d, want 7", got.FumRec)
	}
}

// Scoring the fixture end to end covers the mapping and the rules together:
// a key mapped to the wrong field can still satisfy statLineFrom's own tests
// if its expectation was written to match.
func TestFixtureScores(t *testing.T) {
	weekly := fixtureWeekly(t)

	tests := []struct {
		name     string
		playerID string
		want     float64
	}{
		{
			// 376 passing yards pays 3 + three 50-yard increments = 6; the TD
			// pass 6; its 42 yards 1; the conversion 2; the 2 picks -6. The
			// 14 rushing yards clear no threshold.
			name:     "quarterback",
			playerID: "3294",
			want:     9,
		},
		{
			// 58 receiving yards clear no threshold, so the caught conversion
			// and the lost fumble are the whole line.
			name:     "tight end with a conversion and a fumble",
			playerID: "8110",
			want:     -1,
		},
		{
			name:     "running back with a rushed conversion",
			playerID: "12534",
			want:     2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line, ok := statLineFrom(weekly, tt.playerID, 2025, 14)
			if !ok {
				t.Fatalf("statLineFrom reported player %q absent", tt.playerID)
			}
			if got := Points(line); got != tt.want {
				t.Errorf("Points(%+v) = %v, want %v", line, got, tt.want)
			}
		})
	}
}

func TestStatLineFromAbsent(t *testing.T) {
	weekly := fixtureWeekly(t)

	if _, ok := statLineFrom(weekly, "does-not-exist", 2025, 14); ok {
		t.Error("statLineFrom reported an absent player as present; a silent zero score is exactly the failure this guards")
	}
}

// Sleeper returns a null entry for some rostered-but-idle players. A nil map
// reads every stat as zero, so it has to report absent rather than scoring a
// plausible 0.0.
func TestStatLineFromNullEntry(t *testing.T) {
	weekly := map[string]map[string]float64{NacuaPlayerID: nil}

	if _, ok := statLineFrom(weekly, NacuaPlayerID, 2025, 14); ok {
		t.Error("statLineFrom reported a null entry as present; a silent zero score is exactly the failure this guards")
	}
}
