package sleeper

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestForcedByPairs(t *testing.T) {
	tests := []struct {
		name string
		desc string
		want []fumblePair
	}{
		{
			name: "single fumble",
			desc: "C.Keenum scrambles left end for 7 yards. C.Keenum FUMBLES, forced by J.Greenard. Fumble RECOVERED by PHI-C.Keenum at PHI 42.",
			want: []fumblePair{{"C.Keenum", "J.Greenard"}},
		},
		{
			name: "two fumbles in order",
			desc: "D.Jones steps back to pass. Sacked at IND 44 for -3 yards (W.Anderson). D.Jones FUMBLES, forced by W.Anderson. Fumble RECOVERED by HOU-W.Anderson at IND 42. W.Anderson FUMBLES, forced by M.Alie-Cox. Fumble RECOVERED by HOU-K.Lassiter at IND 40. ",
			want: []fumblePair{{"D.Jones", "W.Anderson"}, {"W.Anderson", "M.Alie-Cox"}},
		},
		{
			name: "pair only before overturned is dropped",
			desc: "J.Herbert steps back to pass. Sacked at BUF 18 for 0 yards (E.Oliver). J.Herbert FUMBLES, forced by E.Oliver. Fumble RECOVERED by BUF-D.Alford at BUF 18. TOUCHDOWN. The Replay Official reviewed the fumble and the play was overturned. J.Herbert steps back to pass. Pass incomplete short left intended for [E.Oliver].",
			want: nil,
		},
		{
			name: "pair only after overturned is kept",
			desc: "J.Allen steps back to pass. Sacked at BUF 34 for 0 yards (K.Mack). Los Angeles challenged the runner was down by contact and the play was overturned. J.Allen steps back to pass. Sacked at BUF 28 for -6 yards (K.Mack). J.Allen FUMBLES, forced by D.Henley. Fumble RECOVERED by LAC-D.Henley at BUF 28.",
			want: []fumblePair{{"J.Allen", "D.Henley"}},
		},
		{
			name: "only the last overturned counts",
			desc: "A.One FUMBLES, forced by B.Two. overturned. C.Three FUMBLES, forced by D.Four. Overturned again. E.Five FUMBLES, forced by F.Six.",
			want: []fumblePair{{"E.Five", "F.Six"}},
		},
		{
			name: "suffixed forcer keeps suffix without swallowing next sentence",
			desc: "C.Keenum FUMBLES, forced by K.Moore II. Fumble RECOVERED by PHI-C.Keenum at PHI 42.",
			want: []fumblePair{{"C.Keenum", "K.Moore II"}},
		},
		{
			name: "suffixed fumbler with period suffix",
			desc: "M.Harrison Jr. FUMBLES, forced by A.Bar. Fumble RECOVERED by X-Y.Z at PHI 42.",
			want: []fumblePair{{"M.Harrison Jr.", "A.Bar"}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := forcedByPairs(tc.desc); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("forcedByPairs() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAbbrevName(t *testing.T) {
	tests := []struct{ name, first, last, parsed, norm string }{
		{"plain", "Case", "Keenum", "C.Keenum", "c.keenum"},
		{"suffix on last name", "Kyle", "Moore II", "K.Moore", "k.moore"},
		{"suffix on parsed name", "Kyle", "Moore", "K.Moore II", "k.moore"},
		{"Jr both sides", "Marvin", "Harrison Jr.", "M.Harrison Jr.", "m.harrison"},
		{"case", "Mo", "Alie-Cox", "M.ALIE-COX", "m.alie-cox"},
		{"V suffix", "Pat", "Smith V", "P.Smith", "p.smith"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got, want := abbrevName(tc.first, tc.last), normalizeName(tc.parsed); got != want || got != tc.norm {
				t.Errorf("abbrevName(%q, %q) = %q, normalizeName(%q) = %q, want both %q", tc.first, tc.last, got, tc.parsed, want, tc.norm)
			}
		})
	}
}

func row(id, first, last, team string, stats map[string]float64) playRow {
	return playRow{PlayerID: id, Stats: stats, Player: playPlayer{FirstName: first, LastName: last, Team: team}}
}

func fumblePlay(desc string, rows ...playRow) play {
	return play{ID: "p1", Metadata: playMeta{Description: desc}, PlayStats: rows}
}

func noLog(string, ...any) {}

func TestForcedFumbleTurnovers(t *testing.T) {
	lostFumble := map[string]float64{"fum": 1, "fum_lost": 1, "idp_ff": 1}

	t.Run("lost fumble credits the forcer, not the fumbler", func(t *testing.T) {
		p := fumblePlay("C.Keenum FUMBLES, forced by J.Greenard. Fumble RECOVERED by PHI-X.Y at PHI 42.",
			row("1737", "Case", "Keenum", "CHI", lostFumble),
			row("6900", "Jonathan", "Greenard", "PHI", map[string]float64{"idp_tkl": 1}))
		got := forcedFumbleTurnovers([]play{p}, noLog)
		want := map[string]int{"6900": 1}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("fumble that was not lost credits no one", func(t *testing.T) {
		p := fumblePlay("C.Keenum FUMBLES, forced by J.Greenard. Fumble RECOVERED by CHI-C.Keenum at CHI 42.",
			row("1737", "Case", "Keenum", "CHI", map[string]float64{"fum": 1, "idp_ff": 1}),
			row("6900", "Jonathan", "Greenard", "PHI", nil))
		assertCredits(t, forcedFumbleTurnovers([]play{p}, noLog), map[string]int{})
	})

	t.Run("forced-by text without idp_ff credits no one", func(t *testing.T) {
		p := fumblePlay("C.Keenum FUMBLES, forced by J.Greenard. Fumble RECOVERED by PHI-X.Y at PHI 42.",
			row("1737", "Case", "Keenum", "CHI", map[string]float64{"fum": 1, "fum_lost": 1}),
			row("6900", "Jonathan", "Greenard", "PHI", nil))
		assertCredits(t, forcedFumbleTurnovers([]play{p}, noLog), map[string]int{})
	})

	t.Run("forcer row with empty stats is credited", func(t *testing.T) {
		p := fixturePlay(t, "testdata/plays_2026_w3.json", "ff5e0d30")
		assertCredits(t, forcedFumbleTurnovers([]play{p}, noLog), map[string]int{"5991": 1})
	})

	t.Run("same-team namesake is not the forcer", func(t *testing.T) {
		p := fumblePlay("C.Keenum FUMBLES, forced by J.Greenard. Fumble RECOVERED by PHI-X.Y at PHI 42.",
			row("1737", "Case", "Keenum", "CHI", lostFumble),
			row("111", "Jim", "Greenard", "CHI", nil),
			row("6900", "Jonathan", "Greenard", "PHI", nil))
		assertCredits(t, forcedFumbleTurnovers([]play{p}, noLog), map[string]int{"6900": 1})
	})

	t.Run("team rows are neither fumbler nor forcer", func(t *testing.T) {
		p := fumblePlay("C.Keenum FUMBLES, forced by J.Greenard. Fumble RECOVERED by PHI-X.Y at PHI 42.",
			row("1737", "Case", "Keenum", "CHI", lostFumble),
			playRow{PlayerID: "PHI", Stats: map[string]float64{"idp_ff": 1, "fum_lost": 1}, Player: playPlayer{FirstName: "J", LastName: "Greenard", Team: "PHI"}},
			row("6900", "Jonathan", "Greenard", "PHI", nil))
		assertCredits(t, forcedFumbleTurnovers([]play{p}, noLog), map[string]int{"6900": 1})
	})

	t.Run("team row flagged as fumbler credits no one", func(t *testing.T) {
		p := fumblePlay("C.Keenum FUMBLES, forced by J.Greenard. Fumble RECOVERED by PHI-X.Y at PHI 42.",
			playRow{PlayerID: "CHI", Stats: lostFumble, Player: playPlayer{FirstName: "Case", LastName: "Keenum", Team: "CHI"}},
			row("6900", "Jonathan", "Greenard", "PHI", nil))
		assertCredits(t, forcedFumbleTurnovers([]play{p}, noLog), map[string]int{})
	})

	t.Run("double fumble is judged per fumble", func(t *testing.T) {
		p := fixturePlay(t, "testdata/plays_2026_w3.json", "4ca72840")
		assertCredits(t, forcedFumbleTurnovers([]play{p}, noLog), map[string]int{"10892": 1})
	})

	t.Run("sacker is not the forcer", func(t *testing.T) {
		p := fixturePlay(t, "testdata/plays_2026_w3.json", "4d35c720")
		assertCredits(t, forcedFumbleTurnovers([]play{p}, noLog), map[string]int{"2393": 1})
	})

	// No fixture play has a forcer before "overturned" that conflicts with the
	// final version, so this one is built by hand.
	t.Run("forcer named only before overturned is not credited", func(t *testing.T) {
		p := fumblePlay("J.Allen FUMBLES, forced by K.Mack. Los Angeles challenged and the play was overturned. J.Allen FUMBLES, forced by D.Henley. Fumble RECOVERED by LAC-D.Henley at BUF 28.",
			row("4984", "Josh", "Allen", "BUF", lostFumble),
			row("4", "Khalil", "Mack", "LAC", map[string]float64{"idp_sack": 1}),
			row("5", "Daiyan", "Henley", "LAC", nil))
		assertCredits(t, forcedFumbleTurnovers([]play{p}, noLog), map[string]int{"5": 1})
	})

	t.Run("ambiguous forcer credits no one and is logged", func(t *testing.T) {
		p := fumblePlay("C.Keenum FUMBLES, forced by J.Greenard. Fumble RECOVERED by PHI-X.Y at PHI 42.",
			row("1737", "Case", "Keenum", "CHI", lostFumble),
			row("6900", "Jonathan", "Greenard", "PHI", nil),
			row("6901", "Jake", "Greenard", "PHI", nil))
		var logged []string
		logf := func(format string, args ...any) { logged = append(logged, fmt.Sprintf(format, args...)) }
		assertCredits(t, forcedFumbleTurnovers([]play{p}, logf), map[string]int{})
		assertLoggedPlay(t, logged, "p1")
	})

	t.Run("no pair for the fumbler credits no one and is logged", func(t *testing.T) {
		p := fumblePlay("C.Keenum FUMBLES, forced by J.Greenard. Fumble RECOVERED by PHI-X.Y at PHI 42.",
			row("2", "Sam", "Darnold", "CHI", lostFumble),
			row("6900", "Jonathan", "Greenard", "PHI", nil))
		var logged []string
		logf := func(format string, args ...any) { logged = append(logged, fmt.Sprintf(format, args...)) }
		assertCredits(t, forcedFumbleTurnovers([]play{p}, logf), map[string]int{})
		assertLoggedPlay(t, logged, "p1")
	})
}

func assertLoggedPlay(t *testing.T, logged []string, playID string) {
	t.Helper()
	if len(logged) != 1 || !strings.Contains(logged[0], playID) {
		t.Errorf("logged %q, want one line mentioning play %q", logged, playID)
	}
}

func assertCredits(t *testing.T, got, want map[string]int) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func loadPlays(t *testing.T, path string) []play {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var plays []play
	if err := json.Unmarshal(b, &plays); err != nil {
		t.Fatal(err)
	}
	return plays
}

func fixturePlay(t *testing.T, path, idPrefix string) play {
	t.Helper()
	for _, p := range loadPlays(t, path) {
		if strings.HasPrefix(p.ID, idPrefix) {
			return p
		}
	}
	t.Fatalf("no play %s in %s", idPrefix, path)
	return play{}
}

func TestForcedFumbleTurnoversOverturnedFixtures(t *testing.T) {
	all := append(loadPlays(t, "testdata/plays_2026_w2.json"), loadPlays(t, "testdata/plays_2026_w3.json")...)
	overturned := map[string]bool{"71ba3f10": true, "235e2300": true, "ee9351e0": true, "bb8b5f10": true}
	var plays []play
	for _, p := range all {
		if overturned[p.ID[:8]] {
			plays = append(plays, p)
		}
	}
	if len(plays) != len(overturned) {
		t.Fatalf("found %d of %d overturned fixture plays", len(plays), len(overturned))
	}
	// Treadwell's OT fumble stood only before the overturn, so it has no idp_ff row.
	want := map[string]int{"10914": 1, "7117": 1, "6896": 1}
	assertCredits(t, forcedFumbleTurnovers(plays, noLog), want)
}

func TestForcedFumbleTurnoversMatchOfficialRecord(t *testing.T) {
	f, err := os.Open("testdata/ff-test-players.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	header := records[0]
	col := func(rec []string, name string) string {
		for i, h := range header {
			if h == name {
				return rec[i]
			}
		}
		t.Fatalf("no column %q", name)
		return ""
	}
	plays := map[string][]play{
		"2": loadPlays(t, "testdata/plays_2026_w2.json"),
		"3": loadPlays(t, "testdata/plays_2026_w3.json"),
	}
	for _, rec := range records[1:] {
		id, team, qtr, clock := col(rec, "sleeper_id"), col(rec, "team"), col(rec, "qtr"), col(rec, "time")
		t.Run(col(rec, "name"), func(t *testing.T) {
			var joined []play
			for _, p := range plays[col(rec, "week")] {
				m := p.Metadata
				if (m.Team == team || m.Opponent == team) && m.QuarterName == qtr && fmt.Sprintf("%d:%02d", m.Minutes, m.Seconds) == clock {
					joined = append(joined, p)
				}
			}
			if len(joined) == 0 {
				t.Fatalf("no fixture play for %s Q%s %s", team, qtr, clock)
			}
			credited := forcedFumbleTurnovers(joined, noLog)[id] > 0
			if want := col(rec, "ff_turnover") == "yes"; credited != want {
				t.Errorf("player %s credited = %v, want %v", id, credited, want)
			}
		})
	}
}
