package roster

import (
	"strings"
	"testing"
)

func TestParseSingleRecord(t *testing.T) {
	got, err := parse(strings.NewReader("4984,Josh Allen"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := []Record{{ID: "4984", Name: "Josh Allen"}}
	if len(got) != len(want) || got[0] != want[0] {
		t.Errorf("parse() = %v, want %v", got, want)
	}
}

func TestParsePreservesFileOrder(t *testing.T) {
	got, err := parse(strings.NewReader("1,A\n2,B\n3,C"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := []Record{{ID: "1", Name: "A"}, {ID: "2", Name: "B"}, {ID: "3", Name: "C"}}
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
	got, err := parse(strings.NewReader("4984 , Josh Allen"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := Record{ID: "4984", Name: "Josh Allen"}
	if len(got) != 1 || got[0] != want {
		t.Errorf("parse() = %v, want [%v]", got, want)
	}
}

func TestParseSplitsOnFirstCommaOnly(t *testing.T) {
	got, err := parse(strings.NewReader("1234,Smith, Jr."))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := Record{ID: "1234", Name: "Smith, Jr."}
	if len(got) != 1 || got[0] != want {
		t.Errorf("parse() = %v, want [%v]", got, want)
	}
}

func TestParseWithoutTrailingNewline(t *testing.T) {
	got, err := parse(strings.NewReader("1,A\n2,B"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := []Record{{ID: "1", Name: "A"}, {ID: "2", Name: "B"}}
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
	got, err := parse(strings.NewReader("4984,\n4985"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := []Record{{ID: "4984", Name: ""}, {ID: "4985", Name: ""}}
	if len(got) != len(want) {
		t.Fatalf("parse() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("record %d = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestParseRefusesLineWithNoID(t *testing.T) {
	_, err := parse(strings.NewReader("1,A\n,Josh Allen\n2,B"))
	if err == nil {
		t.Fatal("parse() = nil error, want an error naming line 2")
	}
	if !strings.Contains(err.Error(), "2") {
		t.Errorf("parse() error = %q, want it to name line 2", err)
	}
}

func TestParseRefusesAllBlankFields(t *testing.T) {
	_, err := parse(strings.NewReader("1,A\n  ,  \n2,B"))
	if err == nil {
		t.Fatal("parse() = nil error, want an error naming line 2")
	}
	if !strings.Contains(err.Error(), "2") {
		t.Errorf("parse() error = %q, want it to name line 2", err)
	}
}

func TestParseRefusesEmptyRoster(t *testing.T) {
	_, err := parse(strings.NewReader("# roster\n\n  \n"))
	if err == nil {
		t.Fatal("parse() = nil error, want an error for a roster with no records")
	}
}

func TestParseRefusesDuplicateID(t *testing.T) {
	_, err := parse(strings.NewReader("4984,Josh Allen\n4984,Someone Else"))
	if err == nil {
		t.Fatal("parse() = nil error, want an error naming the repeated id 4984")
	}
	if !strings.Contains(err.Error(), "4984") {
		t.Errorf("parse() error = %q, want it to name the id 4984", err)
	}
}

func TestParseAllowsRepeatedName(t *testing.T) {
	// Regression: the duplicate check must key on id, not name.
	got, err := parse(strings.NewReader("1,Same Name\n2,Same Name"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("parse() = %v, want 2 records", got)
	}
}

func TestParseSkipsCommentsAndBlankLines(t *testing.T) {
	input := "# roster\n1,A\n\n  # indented comment\n2,B\n3,C\n"
	got, err := parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := []Record{{ID: "1", Name: "A"}, {ID: "2", Name: "B"}, {ID: "3", Name: "C"}}
	if len(got) != len(want) {
		t.Fatalf("parse() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("record %d = %v, want %v", i, got[i], want[i])
		}
	}
}

// The roster is the only thing that ever reads a name, so nothing can notice
// that this one belongs to a different player than the id does.
func TestParseDoesNotValidateName(t *testing.T) {
	got, err := parse(strings.NewReader("4984,Patrick Mahomes\n"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := Record{ID: "4984", Name: "Patrick Mahomes"}
	if len(got) != 1 || got[0] != want {
		t.Errorf("parse() = %v, want [%v]", got, want)
	}
}
