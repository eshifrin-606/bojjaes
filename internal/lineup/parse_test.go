package lineup

import (
	"strings"
	"testing"
)

const header = "id,name,position,team\n"

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
