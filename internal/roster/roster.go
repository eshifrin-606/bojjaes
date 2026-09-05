package roster

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type Record struct {
	ID string
	// Name is display text only. Nothing outside the roster looks a player up
	// by name, so a wrong name pairs silently with the right stats.
	Name string
}

func parse(r io.Reader) ([]Record, error) {
	var records []Record

	scanner := bufio.NewScanner(r)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		id, name, _ := strings.Cut(line, ",")
		id = strings.TrimSpace(id)
		if id == "" {
			// A skipped line would shift every later record up one slot,
			// silently promoting a bench player into the starting nine.
			// Refusing the file is loud; skipping it would not be.
			return nil, fmt.Errorf("line %d: no id", lineNum)
		}
		records = append(records, Record{ID: id, Name: strings.TrimSpace(name)})
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
