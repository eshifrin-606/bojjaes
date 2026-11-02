package lineup

import (
	"slices"
	"strings"
	"testing"
)

const header = "id,name,position,team\n"

// rosterCSV builds a lineup file from alternating id and position arguments,
// giving every record a dummy name and team.
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

// legalRoster is a nine-record lineup file in slot order. Each row given takes
// the place of the first filler record with the same position, so a test names
// only the records it is about.
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

func TestParseSingleRecord(t *testing.T) {
	got, err := parse(strings.NewReader(header + "4984,Josh Allen,QB,BUF"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := []Record{{ID: "4984", Name: "Josh Allen", Position: "QB", Team: "BUF"}}
	if len(got) != len(want) || got[0] != want[0] {
		t.Errorf("parse() = %v, want %v", got, want)
	}
}

func TestParsePreservesFileOrder(t *testing.T) {
	got, err := parse(strings.NewReader(header + "1,A,QB,BUF\n2,B,RB,KC\n3,C,WR,SF"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := []Record{
		{ID: "1", Name: "A", Position: "QB", Team: "BUF"},
		{ID: "2", Name: "B", Position: "RB", Team: "KC"},
		{ID: "3", Name: "C", Position: "WR", Team: "SF"},
	}
	if len(got) != len(want) {
		t.Fatalf("parse() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("record %d = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestParseTrimsSurroundingSpaces(t *testing.T) {
	got, err := parse(strings.NewReader("id, name, position, team\n4984 , Josh Allen , QB , BUF"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := Record{ID: "4984", Name: "Josh Allen", Position: "QB", Team: "BUF"}
	if len(got) != 1 || got[0] != want {
		t.Errorf("parse() = %+v, want [%+v]", got, want)
	}
}

func TestParseWithoutTrailingNewline(t *testing.T) {
	got, err := parse(strings.NewReader(header + "1,A,QB,BUF\n2,B,RB,KC"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := []Record{
		{ID: "1", Name: "A", Position: "QB", Team: "BUF"},
		{ID: "2", Name: "B", Position: "RB", Team: "KC"},
	}
	if len(got) != len(want) {
		t.Fatalf("parse() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("record %d = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestParseMissingNameIsEmpty(t *testing.T) {
	got, err := parse(strings.NewReader(header + "4984,,QB,BUF"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := Record{ID: "4984", Name: "", Position: "QB", Team: "BUF"}
	if len(got) != 1 || got[0] != want {
		t.Errorf("parse() = %+v, want [%+v]", got, want)
	}
}

func TestParseRefusesLineWithNoID(t *testing.T) {
	_, err := parse(strings.NewReader(header + "1,A,QB,BUF\n,Josh Allen,QB,BUF\n2,B,RB,KC"))
	if err == nil {
		t.Fatal("parse() = nil error, want an error naming line 3")
	}
	if !strings.Contains(err.Error(), "line 3") {
		t.Errorf("parse() error = %q, want it to name line 3", err)
	}
}

func TestParseRefusesAllBlankFields(t *testing.T) {
	_, err := parse(strings.NewReader(header + "1,A,QB,BUF\n  , , ,  \n2,B,RB,KC"))
	if err == nil {
		t.Fatal("parse() = nil error, want an error naming line 3")
	}
	if !strings.Contains(err.Error(), "line 3") {
		t.Errorf("parse() error = %q, want it to name line 3", err)
	}
}

func TestParseRefusesEmptyLineup(t *testing.T) {
	_, err := parse(strings.NewReader("# lineup\n\n  \n"))
	if err == nil {
		t.Fatal("parse() = nil error, want an error for a lineup with no records")
	}
}

func TestParseRefusesDuplicateID(t *testing.T) {
	_, err := parse(strings.NewReader(header + "4984,Josh Allen,QB,BUF\n4984,Someone Else,RB,KC"))
	if err == nil {
		t.Fatal("parse() = nil error, want an error naming the repeated id 4984")
	}
	if !strings.Contains(err.Error(), "4984") {
		t.Errorf("parse() error = %q, want it to name the id 4984", err)
	}
}

func TestParseAllowsRepeatedName(t *testing.T) {
	// Regression: the duplicate check must key on id, not name.
	got, err := parse(strings.NewReader(header + "1,Same Name,QB,BUF\n2,Same Name,RB,KC"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("parse() = %v, want 2 records", got)
	}
}

func TestParseSkipsCommentsAndBlankLines(t *testing.T) {
	input := "# lineup\n" + header + "1,A,QB,BUF\n\n  # indented comment\n2,B,RB,KC\n3,C,WR,SF\n"
	got, err := parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := []Record{
		{ID: "1", Name: "A", Position: "QB", Team: "BUF"},
		{ID: "2", Name: "B", Position: "RB", Team: "KC"},
		{ID: "3", Name: "C", Position: "WR", Team: "SF"},
	}
	if len(got) != len(want) {
		t.Fatalf("parse() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("record %d = %v, want %v", i, got[i], want[i])
		}
	}
}

// Name, position, and team are labels: nothing checks them against the
// player the id names (4984 is Josh Allen, QB, BUF).
func TestParseDoesNotValidateLabels(t *testing.T) {
	got, err := parse(strings.NewReader(header + "4984,Travis Kelce,TE,KC\n"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := Record{ID: "4984", Name: "Travis Kelce", Position: "TE", Team: "KC"}
	if len(got) != 1 || got[0] != want {
		t.Errorf("parse() = %v, want [%v]", got, want)
	}
}

func TestParseReadsColumnsByHeaderName(t *testing.T) {
	got, err := parse(strings.NewReader("team,position,name,id\nBUF,QB,Josh Allen,4984"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := Record{ID: "4984", Name: "Josh Allen", Position: "QB", Team: "BUF"}
	if len(got) != 1 || got[0] != want {
		t.Errorf("parse() = %+v, want [%+v]", got, want)
	}
}

func TestParseCommentsBeforeHeaderAreNotTheHeader(t *testing.T) {
	input := "# lineup\n\n   # indented\n" + header + "4984,Josh Allen,QB,BUF\n"
	got, err := parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := Record{ID: "4984", Name: "Josh Allen", Position: "QB", Team: "BUF"}
	if len(got) != 1 || got[0] != want {
		t.Errorf("parse() = %+v, want [%+v]", got, want)
	}
}

func TestParseRefusesHeaderMissingAColumn(t *testing.T) {
	_, err := parse(strings.NewReader("id,name,position\n4984,Josh Allen,QB\n"))
	if err == nil {
		t.Fatal("parse() = nil error, want an error naming the missing column team")
	}
	if !strings.Contains(err.Error(), "team") {
		t.Errorf("parse() error = %q, want it to name the column team", err)
	}
}

func TestParseRefusesUnknownColumn(t *testing.T) {
	_, err := parse(strings.NewReader("id,name,postion,team\n4984,Josh Allen,QB,BUF\n"))
	if err == nil {
		t.Fatal("parse() = nil error, want an error naming the column postion")
	}
	if !strings.Contains(err.Error(), "postion") {
		t.Errorf("parse() error = %q, want it to name the column postion", err)
	}
}

func TestParseRefusesRepeatedColumn(t *testing.T) {
	_, err := parse(strings.NewReader("id,name,position,team,team\n4984,Josh Allen,QB,BUF,BUF\n"))
	if err == nil {
		t.Fatal("parse() = nil error, want an error naming the repeated column team")
	}
	if !strings.Contains(err.Error(), "team") {
		t.Errorf("parse() error = %q, want it to name the column team", err)
	}
}

func TestParseRefusesHeaderlessOldFormat(t *testing.T) {
	_, err := parse(strings.NewReader("# old format\n4984,Josh Allen\n4985,Someone Else\n"))
	if err == nil {
		t.Fatal("parse() = nil error, want the id,name line refused as a header of unknown columns")
	}
}

func TestParseRefusesTooFewFields(t *testing.T) {
	_, err := parse(strings.NewReader(header + "1,A,QB,BUF\n4984,Josh Allen\n"))
	if err == nil {
		t.Fatal("parse() = nil error, want an error naming line 3")
	}
	if !strings.Contains(err.Error(), "line 3") {
		t.Errorf("parse() error = %q, want it to name line 3", err)
	}
}

func TestParseRefusesNameContainingComma(t *testing.T) {
	_, err := parse(strings.NewReader(header + "1,A,QB,BUF\n1234,Smith, Jr.,WR,BUF\n"))
	if err == nil {
		t.Fatal("parse() = nil error, want an error naming line 3")
	}
	if !strings.Contains(err.Error(), "line 3") {
		t.Errorf("parse() error = %q, want it to name line 3", err)
	}
}

func TestParseRefusesUnknownPosition(t *testing.T) {
	_, err := parse(strings.NewReader(header + "1,A,QB,BUF\n2,B,W R,BUF\n"))
	if err == nil {
		t.Fatal("parse() = nil error, want an error naming line 3 and position W R")
	}
	for _, want := range []string{"line 3", `"W R"`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("parse() error = %q, want it to contain %s", err, want)
		}
	}
}

func TestParseRefusesPositionOutsideExactSet(t *testing.T) {
	for _, position := range []string{"wr", "", "OL", "P"} {
		_, err := parse(strings.NewReader(header + "1,A,QB,BUF\n2,B," + position + ",BUF\n"))
		if err == nil {
			t.Errorf("position %q: parse() = nil error, want an error naming line 3", position)
			continue
		}
		for _, want := range []string{"line 3", `"` + position + `"`} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("position %q: parse() error = %q, want it to contain %s", position, err, want)
			}
		}
	}
}

func TestParseCarriesSleeperDefensivePositionsAsWritten(t *testing.T) {
	got, err := parse(strings.NewReader(header + "1,A,DE,CLE\n2,B,DL,NYJ\n"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(got) != 2 || got[0].Position != "DE" || got[1].Position != "DL" {
		t.Errorf("parse() = %+v, want positions DE then DL", got)
	}
}

func TestParseRefusesUnknownTeam(t *testing.T) {
	_, err := parse(strings.NewReader(header + "1,A,QB,BUF\n2,B,QB,BUFF\n"))
	if err == nil {
		t.Fatal("parse() = nil error, want an error naming line 3 and team BUFF")
	}
	for _, want := range []string{"line 3", `"BUFF"`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("parse() error = %q, want it to contain %s", err, want)
		}
	}
}

func TestParseRefusesTeamOutsideExactSet(t *testing.T) {
	for _, team := range []string{"buf", "OAK", ""} {
		_, err := parse(strings.NewReader(header + "1,A,QB,BUF\n2,B,QB," + team + "\n"))
		if err == nil {
			t.Errorf("team %q: parse() = nil error, want an error naming line 3", team)
			continue
		}
		for _, want := range []string{"line 3", `"` + team + `"`} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("team %q: parse() error = %q, want it to contain %s", team, err, want)
			}
		}
	}
}

func TestParseAcceptsFreeAgent(t *testing.T) {
	got, err := parse(strings.NewReader(header + "4984,Josh Allen,QB,FA\n"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(got) != 1 || got[0].Team != "FA" {
		t.Errorf("parse() = %+v, want team FA", got)
	}
}

// Comment and blank lines are not records, so nine records among them are a
// whole lineup.
func TestLineupCommentsAndBlankLinesDoNotCountAsRecords(t *testing.T) {
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
	records, err := parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if _, err := newLineup(records); err != nil {
		t.Errorf("newLineup: %v, want nine records to fill the lineup", err)
	}
}

func TestShortName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{name: "Josh Allen", want: "J Allen"},
		{name: "Mahomes", want: "Mahomes"},
		{name: "", want: ""},
		{name: "Émile Smith", want: "É Smith"},
		{name: "Will McDonald IV", want: "W McDonald"},
		{name: "Marvin Harrison Jr.", want: "M Harrison"},
		{name: "Marvin Harrison Jr", want: "M Harrison"},
		{name: "Luther Burden III", want: "L Burden"},
		{name: "Ken Griffey Sr.", want: "K Griffey"},
		{name: "Ken Griffey Sr", want: "K Griffey"},
		{name: "Robert Griffin II", want: "R Griffin"},
		{name: "Frank Smith V", want: "F Smith"},
		{name: "Tony V", want: "T V"},
		{name: "Amon-Ra St. Brown", want: "A St. Brown"},
		{name: "Ja'Marr Chase", want: "J Chase"},
		{name: "Josh  Allen", want: "J Allen"},
		{name: "A.J. Brown", want: "A.J. Brown"},
		{name: "KC Concepcion", want: "KC Concepcion"},
		{name: "DJ Moore", want: "DJ Moore"},
		{name: "DK Metcalf", want: "DK Metcalf"},
		{name: "TJ Watt", want: "TJ Watt"},
		{name: "JK Dobbins", want: "JK Dobbins"},
		{name: "Al Smith", want: "A Smith"},
		{name: "JOE Smith", want: "J Smith"},
		{name: "CeeDee Lamb", want: "C Lamb"},
		{name: "A.J. Brown Jr.", want: "A.J. Brown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shortName(tt.name); got != tt.want {
				t.Errorf("shortName(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

// readLineup parses a roster file and slots its records, failing the test on
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
	records, err := parse(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return newLineup(records)
}

func starterIDs(l Lineup) []string {
	var ids []string
	for _, rec := range l.Starters() {
		ids = append(ids, rec.ID)
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

	want := Record{ID: "fb", Name: "Player fb", ShortName: "P fb", Position: "FB", Team: "BUF"}
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
