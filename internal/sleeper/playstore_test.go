package sleeper

import (
	"sort"
	"testing"
	"time"
)

func snapshotIDs(s *PlayStore, season, week int) []string {
	ids := []string{}
	for _, p := range s.snapshot(season, week) {
		ids = append(ids, p.ID)
	}
	sort.Strings(ids)
	return ids
}

func assertIDs(t *testing.T, got []string, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestPlayStoreSnapshotHoldsMergedPlaysPerWeek(t *testing.T) {
	s := &PlayStore{}
	s.merge(2026, 2, []play{{ID: "a"}, {ID: "b"}})
	s.merge(2026, 3, []play{{ID: "c"}})

	assertIDs(t, snapshotIDs(s, 2026, 2), "a", "b")
	assertIDs(t, snapshotIDs(s, 2026, 3), "c")
	assertIDs(t, snapshotIDs(s, 2025, 2))
}

func TestPlayStoreNewerCopyReplacesHeld(t *testing.T) {
	s := &PlayStore{}
	s.merge(2026, 3, []play{{ID: "a", UpdatedAt: 100, Metadata: playMeta{Description: "old"}}})
	s.merge(2026, 3, []play{{ID: "a", UpdatedAt: 200, Metadata: playMeta{Description: "new"}}})

	got := s.snapshot(2026, 3)
	if len(got) != 1 || got[0].Metadata.Description != "new" {
		t.Fatalf("got %+v, want the newer copy", got)
	}
}

func TestPlayStoreIgnoresOlderCopy(t *testing.T) {
	s := &PlayStore{}
	s.merge(2026, 3, []play{{ID: "a", UpdatedAt: 200, Metadata: playMeta{Description: "new"}}})
	s.merge(2026, 3, []play{{ID: "a", UpdatedAt: 100, Metadata: playMeta{Description: "old"}}})

	got := s.snapshot(2026, 3)
	if len(got) != 1 || got[0].Metadata.Description != "new" {
		t.Fatalf("got %+v, want the held copy", got)
	}
}

func TestNeedsWholeFetchWhenWeekEmpty(t *testing.T) {
	s := &PlayStore{}
	if !s.needsWholeFetch(2026, 3, []play{{ID: "a", UpdatedAt: 100}}) {
		t.Fatal("empty week should need a whole fetch")
	}
}

func TestNeedsWholeFetchOnGap(t *testing.T) {
	s := &PlayStore{}
	s.merge(2026, 3, []play{{ID: "a", UpdatedAt: 100}})
	if !s.needsWholeFetch(2026, 3, []play{{ID: "b", UpdatedAt: 150}, {ID: "c", UpdatedAt: 200}}) {
		t.Fatal("poll starting after the newest held play leaves a gap")
	}
}

func TestNeedsWholeFetchFalseOnOverlap(t *testing.T) {
	s := &PlayStore{}
	s.merge(2026, 3, []play{{ID: "a", UpdatedAt: 100}})
	if s.needsWholeFetch(2026, 3, []play{{ID: "a", UpdatedAt: 100}, {ID: "c", UpdatedAt: 200}}) {
		t.Fatal("overlapping poll should not need a whole fetch")
	}
}

func TestPostGameRefresh(t *testing.T) {
	newest := time.Date(2026, 9, 28, 20, 0, 0, 0, time.UTC)
	held := func() *PlayStore {
		s := &PlayStore{}
		s.merge(2026, 3, []play{{ID: "a", UpdatedAt: newest.UnixMilli()}})
		return s
	}

	t.Run("quiet week never fetched whole", func(t *testing.T) {
		if !held().needsPostGameRefresh(2026, 3, newest.Add(61*time.Minute)) {
			t.Fatal("want refresh")
		}
	})
	t.Run("still within the hour", func(t *testing.T) {
		if held().needsPostGameRefresh(2026, 3, newest.Add(59*time.Minute)) {
			t.Fatal("week may still be live")
		}
	})
	t.Run("whole fetch long after last change", func(t *testing.T) {
		s := held()
		s.markWholeFetched(2026, 3, newest.Add(90*time.Minute))
		if s.needsPostGameRefresh(2026, 3, newest.Add(3*time.Hour)) {
			t.Fatal("settled week should not refetch")
		}
	})
	t.Run("whole fetch within the hour of last change", func(t *testing.T) {
		s := held()
		s.markWholeFetched(2026, 3, newest.Add(10*time.Minute))
		if !s.needsPostGameRefresh(2026, 3, newest.Add(3*time.Hour)) {
			t.Fatal("corrections may have landed after that fetch")
		}
	})
}
