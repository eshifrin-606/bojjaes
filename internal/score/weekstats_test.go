package score

import (
	"sync"
	"testing"
)

func TestWeekStatsPlayerReturnsHeldStatLine(t *testing.T) {
	w := NewWeekStats(2025, 14, map[string]StatLine{
		"9493": {PlayerID: "9493", Season: 2025, Week: 14, RecYd: 105},
	})

	got, ok := w.Player("9493")
	if !ok {
		t.Fatalf("Player(9493) reported absent; want found")
	}
	if got.RecYd != 105 {
		t.Errorf("RecYd = %d, want 105", got.RecYd)
	}
}

func TestWeekStatsPlayerAbsentIsAValue(t *testing.T) {
	w := NewWeekStats(2025, 14, map[string]StatLine{
		"9493": {PlayerID: "9493", Season: 2025, Week: 14, RecYd: 105},
	})

	got, ok := w.Player("0000")
	if ok {
		t.Fatalf("Player(0000) reported found; want absent")
	}
	if got != (StatLine{}) {
		t.Errorf("absent player yielded %+v, want the zero StatLine", got)
	}
}

// A stat line cannot be attributed to a week other than the one it was fetched
// for, however it was labelled on the way in.
func TestWeekStatsPlayerCarriesTheWeeksSeasonAndWeek(t *testing.T) {
	w := NewWeekStats(2025, 14, map[string]StatLine{
		"9493": {PlayerID: "9493", Season: 1999, Week: 3, RecYd: 105},
	})

	got, _ := w.Player("9493")
	if got.Season != 2025 || got.Week != 14 {
		t.Errorf("season/week = %d/%d, want 2025/14", got.Season, got.Week)
	}
}

// A WeekStats is one value shared by every request in flight. Reading it is safe
// only because nothing writes to it after construction — under -race, a
// memoising read would show up here.
func TestWeekStatsConcurrentReadsAgree(t *testing.T) {
	w := NewWeekStats(2025, 14, map[string]StatLine{
		"9493": {PlayerID: "9493", RecYd: 167, RecTD: 2},
	})

	want, _ := w.Player("9493")

	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if got, ok := w.Player("9493"); !ok || got != want {
				t.Errorf("concurrent read = %+v (found %v), want %+v", got, ok, want)
			}
			if _, ok := w.Player("absent"); ok {
				t.Error("concurrent read found an absent player")
			}
		}()
	}
	wg.Wait()
}
