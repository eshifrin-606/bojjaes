package weekcache

import (
	"context"
	"testing"
	"time"
)

func TestAFetchedWeekReportsItsFetchInstant(t *testing.T) {
	cache, _ := newTestCache(&fakeSource{}, testTTL)

	_, asOf, err := cache.WeekStatsAsOf(context.Background(), 2026, 4)
	if err != nil {
		t.Fatalf("WeekStatsAsOf: %v", err)
	}

	got, ok := cache.FetchedAt(2026, 4)
	if !ok {
		t.Fatal("FetchedAt reported no instant for a fetched week")
	}
	if !got.Equal(asOf) {
		t.Errorf("FetchedAt = %v, want the instant WeekStatsAsOf returned, %v", got, asOf)
	}
}

func TestAWeekInFlightHasNoInstantYet(t *testing.T) {
	source := &fakeSource{}
	source.block(4)
	cache, _ := newTestCache(source, testTTL)

	fetched := make(chan error, 1)
	go func() {
		_, _, err := cache.WeekStatsAsOf(context.Background(), 2026, 4)
		fetched <- err
	}()
	waitForCalls(t, source, 1)

	type answer struct {
		at time.Time
		ok bool
	}
	asked := make(chan answer, 1)
	go func() {
		at, ok := cache.FetchedAt(2026, 4)
		asked <- answer{at, ok}
	}()

	select {
	case got := <-asked:
		if got.ok {
			t.Errorf("FetchedAt reported %v for a week still in flight, want none", got.at)
		}
	case <-time.After(2 * time.Second):
		t.Error("FetchedAt waited on the flight instead of answering")
	}

	source.release(4)
	if err := <-fetched; err != nil {
		t.Fatalf("WeekStatsAsOf: %v", err)
	}
}

func TestANeverFetchedWeekHasNoInstant(t *testing.T) {
	source := &fakeSource{}
	cache, _ := newTestCache(source, testTTL)

	if at, ok := cache.FetchedAt(2026, 4); ok {
		t.Errorf("FetchedAt reported %v for a week never fetched, want none", at)
	}
	if got := source.callCount(); got != 0 {
		t.Errorf("upstream called %d times, want 0", got)
	}
}

func TestAnInvalidatedWeekHasNoInstant(t *testing.T) {
	cache, _ := newTestCache(&fakeSource{}, testTTL)

	if _, _, err := cache.WeekStatsAsOf(context.Background(), 2026, 4); err != nil {
		t.Fatalf("WeekStatsAsOf: %v", err)
	}
	cache.Invalidate(2026, 4)

	if at, ok := cache.FetchedAt(2026, 4); ok {
		t.Errorf("FetchedAt reported %v for an invalidated week, want none", at)
	}
}

// Expiry decides when the next read fetches, not whether the last fetch
// happened: an entry merely ageing must not look superseded.
func TestAnExpiredEntryStillReportsItsInstant(t *testing.T) {
	source := &fakeSource{}
	cache, clock := newTestCache(source, testTTL)

	_, asOf, err := cache.WeekStatsAsOf(context.Background(), 2026, 4)
	if err != nil {
		t.Fatalf("WeekStatsAsOf: %v", err)
	}
	clock.advance(6 * time.Minute)

	got, ok := cache.FetchedAt(2026, 4)
	if !ok {
		t.Fatal("FetchedAt reported no instant for an entry past its TTL")
	}
	if !got.Equal(asOf) {
		t.Errorf("FetchedAt = %v, want the fetch instant %v", got, asOf)
	}
	if got := source.callCount(); got != 1 {
		t.Errorf("upstream called %d times, want 1", got)
	}
}

func TestAskingForTheFetchInstantCostsNothing(t *testing.T) {
	source := &fakeSource{}
	cache, _ := newTestCache(source, testTTL)

	if _, _, err := cache.WeekStatsAsOf(context.Background(), 2026, 4); err != nil {
		t.Fatalf("WeekStatsAsOf: %v", err)
	}
	logs := recordLogs(cache)

	for range 100 {
		cache.FetchedAt(2026, 4)
		cache.FetchedAt(2026, 5)
	}

	if got := source.callCount(); got != 1 {
		t.Errorf("upstream called %d times, want 1", got)
	}
	if len(*logs) != 0 {
		t.Errorf("asking logged %d lines, want none: %q", len(*logs), *logs)
	}
}
