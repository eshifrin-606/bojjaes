package lineup

import (
	"fmt"
	"io"
	"slices"
	"strings"
)

// Lineup is one player per slot, in slot order.
type Lineup struct {
	starters [StarterCount]Player
}

func newLineup(players []Player) (Lineup, error) {
	if len(players) != StarterCount {
		return Lineup{}, fmt.Errorf("%d players, want %d", len(players), StarterCount)
	}
	var l Lineup
	var filled [StarterCount]bool
	claimed := make([]bool, len(players))
	fill := func(slot int) {
		for i, p := range players {
			if !claimed[i] && slotOrder[slot].Accepts(p.Position) {
				p.ShortName = shortName(p.Name)
				l.starters[slot], filled[slot], claimed[i] = p, true, true
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
	// With exactly nine players, any player no slot accepted leaves a slot
	// empty, so naming the empty slot also accounts for the misfit.
	if empty := slices.Index(filled[:], false); empty >= 0 {
		return Lineup{}, fmt.Errorf("no player fills slot %s", slotOrder[empty])
	}
	return l, nil
}

// WriteCSV renders the lineup as a lineup file in slot order. ShortName is
// derived on read, so it is not written.
func (l Lineup) WriteCSV(w io.Writer) error {
	var b strings.Builder
	b.WriteString("id,name,position,team\n")
	for _, p := range l.starters {
		fmt.Fprintf(&b, "%s,%s,%s,%s\n", p.ID, p.Name, p.Position, p.Team)
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func (l Lineup) Starters() []Player {
	return l.starters[:]
}

// FromSlots builds a lineup that keeps each player in the slot given, unlike
// a read, which assigns slots by position.
func FromSlots(players [StarterCount]Player) (Lineup, error) {
	seen := make(map[string]bool, len(players))
	for i, p := range players {
		slot := slotOrder[i]
		if p.ID == "" {
			return Lineup{}, fmt.Errorf("slot %s is empty", slot)
		}
		if !slot.Accepts(p.Position) {
			return Lineup{}, fmt.Errorf("slot %s: %s cannot play it", slot, p.Position)
		}
		if seen[p.ID] {
			return Lineup{}, fmt.Errorf("slot %s: %s is already in another slot", slot, p.ID)
		}
		seen[p.ID] = true
	}
	return Lineup{starters: players}, nil
}
