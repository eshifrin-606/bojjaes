package lineup

import (
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

// rosterCSV builds a lineup file from alternating id and position arguments,
// giving every player a dummy name and team.
func rosterCSV(t *testing.T, idPositions ...string) string {
	t.Helper()
	if len(idPositions)%2 != 0 {
		t.Fatalf("rosterCSV: odd argument count %d, want id/position pairs", len(idPositions))
	}
	var b strings.Builder
	b.WriteString(header)
	for i := 0; i < len(idPositions); i += 2 {
		id, position := idPositions[i], idPositions[i+1]
		b.WriteString(id + ",Player " + id + "," + position + ",BUF\n")
	}
	return b.String()
}

// legalRoster is a nine-player lineup file in slot order. Each row given takes
// the place of the first filler player with the same position, so a test names
// only the players it is about.
func legalRoster(t *testing.T, rows ...string) string {
	t.Helper()
	lines := []string{
		"901,Filler Quarterback,QB,BUF",
		"902,Filler Back,RB,BUF",
		"903,Filler Receiver,WR,BUF",
		"904,Filler End,TE,BUF",
		"905,Filler Flanker,WR,BUF",
		"906,Filler Runner,RB,BUF",
		"907,Filler Kicker,K,BUF",
		"908,Filler Backer,LB,BUF",
		"909,Filler Corner,CB,BUF",
	}
	replaced := make([]bool, len(lines))
	for _, row := range rows {
		position := strings.TrimSpace(strings.Split(row, ",")[2])
		i := -1
		for j, line := range lines {
			if !replaced[j] && strings.Split(line, ",")[2] == position {
				i = j
				break
			}
		}
		if i < 0 {
			t.Fatalf("legalRoster: no filler %s left for row %q", position, row)
		}
		lines[i], replaced[i] = row, true
	}
	return header + strings.Join(lines, "\n") + "\n"
}

// Comment and blank lines are not players, so nine players among them are a
// whole lineup.
func TestLineupCommentsAndBlankLinesDoNotCountAsPlayers(t *testing.T) {
	input := `# starting nine
id,name,position,team

1,Alpha,QB,BUF
2,Bravo,RB,BUF
# midway comment
3,Charlie,RB,BUF
4,Delta,WR,BUF

5,Echo,WR,BUF
6,Foxtrot,CB,BUF
7,Golf,TE,BUF
8,Hotel,K,BUF
9,India,LB,BUF
`
	players, err := parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if _, err := newLineup(players); err != nil {
		t.Errorf("newLineup: %v, want nine players to fill the lineup", err)
	}
}

// readLineup parses a roster file and slots its players, failing the test on
// any error.
func readLineup(t *testing.T, csv string) Lineup {
	t.Helper()
	l, err := readLineupErr(t, csv)
	if err != nil {
		t.Fatalf("newLineup: %v", err)
	}
	return l
}

func readLineupErr(t *testing.T, csv string) (Lineup, error) {
	t.Helper()
	players, err := parse(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return newLineup(players)
}

func starterIDs(l Lineup) []string {
	var ids []string
	for _, p := range l.Starters() {
		ids = append(ids, p.ID)
	}
	return ids
}

func TestLineupInSlotOrderKeepsFileOrder(t *testing.T) {
	l := readLineup(t, rosterCSV(t,
		"1", "QB", "2", "RB", "3", "WR", "4", "TE", "5", "WR", "6", "RB", "7", "K", "8", "LB", "9", "CB"))

	want := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"}
	if got := starterIDs(l); !slices.Equal(got, want) {
		t.Errorf("starter ids = %v, want %v", got, want)
	}
}

func TestLineupFillsSlotsByPosition(t *testing.T) {
	l := readLineup(t, rosterCSV(t,
		"k", "K", "lb", "LB", "wr1", "WR", "wr2", "WR", "te", "TE", "rb1", "RB", "rb2", "RB", "qb", "QB", "de", "DE"))

	want := []string{"qb", "rb1", "wr1", "te", "wr2", "rb2", "k", "lb", "de"}
	if got := starterIDs(l); !slices.Equal(got, want) {
		t.Errorf("starter ids = %v, want %v", got, want)
	}
}

func TestLineupFullbackFillsRBSlotAsFB(t *testing.T) {
	l := readLineup(t, rosterCSV(t,
		"qb", "QB", "fb", "FB", "wr1", "WR", "wr2", "WR", "te", "TE", "wr3", "WR", "k", "K", "lb", "LB", "cb", "CB"))

	want := Player{ID: "fb", Name: "Player fb", ShortName: "P fb", Position: "FB", Team: "BUF"}
	if got := l.Starters()[1]; got != want {
		t.Errorf("RB slot = %+v, want %+v", got, want)
	}
}

func TestLineupExtraSkillPlayersFillFlexInFileOrder(t *testing.T) {
	tests := []struct {
		name   string
		roster []string
		want   []string
	}{
		{
			name:   "three RBs before the WR and TE",
			roster: []string{"qb", "QB", "rb1", "RB", "rb2", "RB", "rb3", "RB", "wr", "WR", "te", "TE", "k", "K", "lb", "LB", "cb", "CB"},
			want:   []string{"qb", "rb1", "wr", "te", "rb2", "rb3", "k", "lb", "cb"},
		},
		{
			name:   "three TEs",
			roster: []string{"qb", "QB", "te1", "TE", "rb", "RB", "te2", "TE", "wr", "WR", "te3", "TE", "k", "K", "lb", "LB", "cb", "CB"},
			want:   []string{"qb", "rb", "wr", "te1", "te2", "te3", "k", "lb", "cb"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := readLineup(t, rosterCSV(t, tt.roster...))

			if got := starterIDs(l); !slices.Equal(got, tt.want) {
				t.Errorf("starter ids = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLineupDefendersKeepFileOrder(t *testing.T) {
	l := readLineup(t, rosterCSV(t,
		"cb", "CB", "qb", "QB", "rb1", "RB", "wr1", "WR", "te", "TE", "de", "DE", "wr2", "WR", "rb2", "RB", "k", "K"))

	if got, want := starterIDs(l)[7:], []string{"cb", "de"}; !slices.Equal(got, want) {
		t.Errorf("D slots = %v, want %v", got, want)
	}
}

func TestLineupSwappingRBLinesSwapsRBAndFlex(t *testing.T) {
	before := readLineup(t, rosterCSV(t,
		"qb", "QB", "rb1", "RB", "wr1", "WR", "te", "TE", "rb2", "RB", "wr2", "WR", "k", "K", "lb", "LB", "cb", "CB"))
	after := readLineup(t, rosterCSV(t,
		"qb", "QB", "rb2", "RB", "wr1", "WR", "te", "TE", "rb1", "RB", "wr2", "WR", "k", "K", "lb", "LB", "cb", "CB"))

	want := starterIDs(before)
	rb, flex := slices.Index(slotOrder[:], RB), slices.Index(slotOrder[:], FLEX)
	want[rb], want[flex] = want[flex], want[rb]
	if got := starterIDs(after); !slices.Equal(got, want) {
		t.Errorf("after swap, starter ids = %v, want %v", got, want)
	}
}

func TestLineupRefusesRosterThatLeavesASlotEmpty(t *testing.T) {
	tests := []struct {
		name     string
		roster   []string
		wantSlot string
	}{
		{
			name:     "no tight end",
			roster:   []string{"qb", "QB", "rb1", "RB", "rb2", "RB", "wr1", "WR", "wr2", "WR", "wr3", "WR", "k", "K", "lb", "LB", "cb", "CB"},
			wantSlot: "TE",
		},
		{
			name:     "a second quarterback",
			roster:   []string{"qb1", "QB", "qb2", "QB", "rb", "RB", "wr1", "WR", "te", "TE", "wr2", "WR", "k", "K", "lb", "LB", "cb", "CB"},
			wantSlot: "FLEX",
		},
		{
			name:     "a third defender",
			roster:   []string{"qb", "QB", "k", "K", "rb", "RB", "wr1", "WR", "te", "TE", "wr2", "WR", "lb", "LB", "cb", "CB", "de", "DE"},
			wantSlot: "FLEX",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := readLineupErr(t, rosterCSV(t, tt.roster...))
			if err == nil {
				t.Fatal("newLineup() = nil error, want an error naming an empty slot")
			}
			if !strings.Contains(err.Error(), tt.wantSlot) {
				t.Errorf("newLineup() error = %q, want it to name slot %s", err, tt.wantSlot)
			}
		})
	}
}

func TestLineupWriteCSVRendersInSlotOrder(t *testing.T) {
	l := readLineup(t, rosterCSV(t,
		"k", "K", "lb", "LB", "wr1", "WR", "wr2", "WR", "te", "TE", "rb1", "RB", "rb2", "RB", "qb", "QB", "de", "DE"))

	var b strings.Builder
	if err := l.WriteCSV(&b); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}

	want := header +
		"qb,Player qb,QB,BUF\n" +
		"rb1,Player rb1,RB,BUF\n" +
		"wr1,Player wr1,WR,BUF\n" +
		"te,Player te,TE,BUF\n" +
		"wr2,Player wr2,WR,BUF\n" +
		"rb2,Player rb2,RB,BUF\n" +
		"k,Player k,K,BUF\n" +
		"lb,Player lb,LB,BUF\n" +
		"de,Player de,DE,BUF\n"
	if got := b.String(); got != want {
		t.Errorf("WriteCSV =\n%s\nwant\n%s", got, want)
	}
}

func TestLineupWriteCSVKeepsFullbackPosition(t *testing.T) {
	l := readLineup(t, rosterCSV(t,
		"qb", "QB", "fb", "FB", "wr1", "WR", "wr2", "WR", "te", "TE", "wr3", "WR", "k", "K", "lb", "LB", "cb", "CB"))

	var b strings.Builder
	if err := l.WriteCSV(&b); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}

	if got, want := strings.Split(b.String(), "\n")[2], "fb,Player fb,FB,BUF"; got != want {
		t.Errorf("RB slot line = %q, want %q", got, want)
	}
}

func TestLineupWriteCSVReadsBackUnchanged(t *testing.T) {
	l := readLineup(t, legalRoster(t,
		"k,Jason Sanders,K,MIA",
		"qb,Patrick Mahomes II,QB,KC",
		"rb,Christian McCaffrey,RB,SF"))

	var b strings.Builder
	if err := l.WriteCSV(&b); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}

	if got := readLineup(t, b.String()); !slices.Equal(got.Starters(), l.Starters()) {
		t.Errorf("read back = %+v, want %+v", got.Starters(), l.Starters())
	}
}

// slotPlayers is nine eligible players in slot order, an RB in RB and a
// second RB in the first FLEX. Callers swap in the player a test is about.
func slotPlayers() [StarterCount]Player {
	var players [StarterCount]Player
	for i, position := range []string{"QB", "RB", "WR", "TE", "RB", "WR", "K", "LB", "CB"} {
		id := string(rune('a' + i))
		players[i] = Player{ID: id, Name: "Player " + id, Position: position, Team: "BUF"}
	}
	return players
}

func TestFromSlotsKeepsTheGivenSlots(t *testing.T) {
	players := slotPlayers()

	l, err := FromSlots(players)
	if err != nil {
		t.Fatalf("FromSlots: %v", err)
	}

	want := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i"}
	if got := starterIDs(l); !slices.Equal(got, want) {
		t.Errorf("starters = %q, want %q", got, want)
	}
}

func TestFromSlotsRefusesAPlayerTheSlotDoesNotAccept(t *testing.T) {
	players := slotPlayers()
	players[0].Position = "WR"

	_, err := FromSlots(players)

	if err == nil || !strings.Contains(err.Error(), "QB") {
		t.Errorf("FromSlots error = %v, want one naming QB", err)
	}
}

func TestFromSlotsRefusesOnePlayerInTwoSlots(t *testing.T) {
	players := slotPlayers()
	players[4].ID = players[1].ID

	_, err := FromSlots(players)

	if err == nil {
		t.Errorf("FromSlots error = nil, want a duplicate")
	}
}

func TestFromSlotsRefusesAPlayerWithNoID(t *testing.T) {
	players := slotPlayers()
	players[6].ID = ""

	_, err := FromSlots(players)

	if err == nil || !strings.Contains(err.Error(), "K") {
		t.Errorf("FromSlots error = %v, want one naming K", err)
	}
}

// ShortName is derived on read, so it is the one field a round trip adds.
func TestFromSlotsReadsBackFromTheTreeInTheSameSlots(t *testing.T) {
	built, err := FromSlots(slotPlayers())
	if err != nil {
		t.Fatalf("FromSlots: %v", err)
	}
	var b strings.Builder
	if err := built.WriteCSV(&b); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}
	tree := New(fstest.MapFS{"2026/9/wood.csv": &fstest.MapFile{Data: []byte(b.String())}})

	read, err := tree.Read(2026, 9, "wood")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	got := read.Starters()
	for i := range got {
		got[i].ShortName = ""
	}
	if want := built.Starters(); !slices.Equal(got, want) {
		t.Errorf("read back %v, want %v", got, want)
	}
}
