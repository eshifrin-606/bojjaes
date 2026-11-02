package lineup

import (
	"bufio"
	"fmt"
	"io"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Record struct {
	ID string
	// Name is display text only. Nothing outside the lineup looks a player up
	// by name, so a wrong name pairs silently with the right stats.
	Name string
	// ShortName is Name narrowed for tight layouts, and is display text on
	// the same terms: nothing identifies a player by it.
	ShortName string
	// Position is display text only.
	Position string
	// Team is display text and the key to the player's game in the week's
	// games; it never identifies the player.
	Team string
}

// positions copies Sleeper's spelling of every position that can score in
// this league. Sleeper lists some linemen as DL and others as DE; both are
// kept as written rather than grouped.
var positions = []string{
	"QB", "RB", "FB", "WR", "TE", "K",
	"DL", "DE", "DT", "NT", "LB", "OLB", "ILB", "DB", "CB", "S", "SS", "FS",
}

// teams leaves out OAK, which Sleeper's index still carries. FA means the
// player had no NFL team when the file was written.
var teams = []string{
	"ARI", "ATL", "BAL", "BUF", "CAR", "CHI", "CIN", "CLE", "DAL", "DEN", "DET",
	"GB", "HOU", "IND", "JAX", "KC", "LAC", "LAR", "LV", "MIA", "MIN", "NE",
	"NO", "NYG", "NYJ", "PHI", "PIT", "SEA", "SF", "TB", "TEN", "WAS", "FA",
}

var requiredColumns = []string{"id", "name", "position", "team"}

func parseHeader(fields []string) (map[string]int, error) {
	columns := make(map[string]int, len(fields))
	for i, name := range fields {
		name = strings.TrimSpace(name)
		if !slices.Contains(requiredColumns, name) {
			return nil, fmt.Errorf("header: unknown column %q", name)
		}
		if _, seen := columns[name]; seen {
			return nil, fmt.Errorf("header: repeated column %q", name)
		}
		columns[name] = i
	}
	for _, name := range requiredColumns {
		if _, ok := columns[name]; !ok {
			return nil, fmt.Errorf("header: missing column %q", name)
		}
	}
	return columns, nil
}

func parse(r io.Reader) ([]Record, error) {
	var records []Record

	scanner := bufio.NewScanner(r)
	var columns map[string]int
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		fields := strings.Split(line, ",")
		if columns == nil {
			var err error
			if columns, err = parseHeader(fields); err != nil {
				return nil, err
			}
			continue
		}

		if len(fields) != len(columns) {
			return nil, fmt.Errorf("line %d: %d fields, header has %d", lineNum, len(fields), len(columns))
		}
		field := func(column string) string {
			return strings.TrimSpace(fields[columns[column]])
		}

		id := field("id")
		if id == "" {
			// A skipped line would silently drop a player from the lineup.
			// Refusing the file is loud; skipping it would not be.
			return nil, fmt.Errorf("line %d: no id", lineNum)
		}
		position := field("position")
		if !slices.Contains(positions, position) {
			return nil, fmt.Errorf("line %d: unknown position %q", lineNum, position)
		}
		team := field("team")
		if !slices.Contains(teams, team) {
			return nil, fmt.Errorf("line %d: unknown team %q", lineNum, team)
		}
		records = append(records, Record{
			ID:       id,
			Name:     field("name"),
			Position: position,
			Team:     team,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("no records")
	}

	seen := make(map[string]bool, len(records))
	for _, rec := range records {
		if seen[rec.ID] {
			// Two records for one id would double-print and double-count that
			// player's points in the starters' total, with nothing in the
			// report disclosing it.
			return nil, fmt.Errorf("duplicate id %s", rec.ID)
		}
		seen[rec.ID] = true
	}

	return records, nil
}

// StarterCount is the league's starting lineup size.
const StarterCount = len(slotOrder)

type Slot string

const (
	QB   Slot = "QB"
	RB   Slot = "RB"
	WR   Slot = "WR"
	TE   Slot = "TE"
	FLEX Slot = "FLEX"
	K    Slot = "K"
	D    Slot = "D"
)

var slotOrder = [9]Slot{QB, RB, WR, TE, FLEX, FLEX, K, D, D}

var slotAccepts = map[Slot][]string{
	QB:   {"QB"},
	RB:   {"RB", "FB"},
	WR:   {"WR"},
	TE:   {"TE"},
	FLEX: {"RB", "FB", "WR", "TE"},
	K:    {"K"},
	D:    {"DL", "DE", "DT", "NT", "LB", "OLB", "ILB", "DB", "CB", "S", "SS", "FS"},
}

// Lineup is a roster's records, one per slot, in slot order.
type Lineup struct {
	starters [StarterCount]Record
}

func newLineup(records []Record) (Lineup, error) {
	if len(records) != StarterCount {
		return Lineup{}, fmt.Errorf("%d records, want %d", len(records), StarterCount)
	}
	var l Lineup
	var filled [StarterCount]bool
	claimed := make([]bool, len(records))
	fill := func(slot int) {
		for i, rec := range records {
			if !claimed[i] && slices.Contains(slotAccepts[slotOrder[slot]], rec.Position) {
				rec.ShortName = shortName(rec.Name)
				l.starters[slot], filled[slot], claimed[i] = rec, true, true
				return
			}
		}
	}
	// The dedicated slots claim first, so a FLEX gets only the skill players
	// they leave over. Within each pass, file order breaks ties.
	for slot, s := range slotOrder {
		if s != FLEX && s != D {
			fill(slot)
		}
	}
	for slot, s := range slotOrder {
		if s == FLEX || s == D {
			fill(slot)
		}
	}
	// With exactly nine records, any record no slot accepted leaves a slot
	// empty, so naming the empty slot also accounts for the misfit.
	if empty := slices.Index(filled[:], false); empty >= 0 {
		return Lineup{}, fmt.Errorf("no record fills slot %s", slotOrder[empty])
	}
	return l, nil
}

// WriteCSV renders the lineup as a roster file in slot order. ShortName is
// derived on read, so it is not written.
func (l Lineup) WriteCSV(w io.Writer) error {
	var b strings.Builder
	b.WriteString("id,name,position,team\n")
	for _, rec := range l.starters {
		fmt.Fprintf(&b, "%s,%s,%s,%s\n", rec.ID, rec.Name, rec.Position, rec.Team)
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// Starters returns the lineup's records in slot order.
func (l Lineup) Starters() []Record {
	return l.starters[:]
}

var generationalSuffixes = []string{"Jr.", "Jr", "Sr.", "Sr", "II", "III", "IV", "V"}

func shortName(name string) string {
	words := strings.Fields(name)
	if len(words) < 2 {
		return name
	}
	if len(words) > 2 && slices.Contains(generationalSuffixes, words[len(words)-1]) {
		words = words[:len(words)-1]
	}
	lead := string([]rune(words[0])[0])
	if writtenAsInitials(words[0]) {
		lead = words[0]
	}
	return lead + " " + strings.Join(words[1:], " ")
}

// Only two capitals count as initials without a period: a longer all-capitals
// word ("JOE") is as likely to be a name typed in capitals.
func writtenAsInitials(word string) bool {
	if strings.Contains(word, ".") {
		return true
	}
	return utf8.RuneCountInString(word) == 2 &&
		!strings.ContainsFunc(word, func(r rune) bool { return !unicode.IsUpper(r) })
}
