package statscache

import (
	"context"
	"testing"
	"time"
)

func TestARequestAfterInvalidationFetchesAgain(t *testing.T) {
	source := &fakeSource{}
	cache, clock := newTestCache(source, testTTL)

	if _, _, err := cache.WeekStatsAsOf(context.Background(), 2026, 4); err != nil {
		t.Fatalf("first WeekStats: %v", err)
	}
	clock.advance(time.Minute)
	cache.Invalidate(2026, 4)

	second, _, err := cache.WeekStatsAsOf(context.Background(), 2026, 4)
	if err != nil {
		t.Fatalf("second WeekStats: %v", err)
	}
	if got := source.callCount(); got != 2 {
		t.Errorf("upstream called %d times, want 2", got)
	}
	if got := fetchOrdinal(t, second); got != 2 {
		t.Errorf("second caller was served fetch %d, want fetch 2", got)
	}
}

func TestInvalidationMakesNoCallByItself(t *testing.T) {
	source := &fakeSource{}
	cache := New(source, testTTL)

	if _, _, err := cache.WeekStatsAsOf(context.Background(), 2026, 4); err != nil {
		t.Fatalf("WeekStats: %v", err)
	}
	cache.Invalidate(2026, 4)

	if got := source.callCount(); got != 1 {
		t.Errorf("upstream called %d times, want 1", got)
	}
}

func TestInvalidationLeavesOtherWeeksCached(t *testing.T) {
	source := &fakeSource{}
	cache, _ := newTestCache(source, testTTL)

	for _, week := range []int{4, 5} {
		if _, _, err := cache.WeekStatsAsOf(context.Background(), 2026, week); err != nil {
			t.Fatalf("week %d WeekStats: %v", week, err)
		}
	}
	cache.Invalidate(2026, 4)

	if _, _, err := cache.WeekStatsAsOf(context.Background(), 2026, 5); err != nil {
		t.Fatalf("week 5 again: %v", err)
	}
	if got := source.callCount(); got != 2 {
		t.Errorf("upstream called %d times, want 2", got)
	}
}

func TestInvalidatingANeverFetchedWeekIsHarmless(t *testing.T) {
	source := &fakeSource{}
	cache := New(source, testTTL)

	cache.Invalidate(2026, 4)

	if _, _, err := cache.WeekStatsAsOf(context.Background(), 2026, 4); err != nil {
		t.Fatalf("WeekStats: %v", err)
	}
	if got := source.callCount(); got != 1 {
		t.Errorf("upstream called %d times, want 1", got)
	}
}

func TestAnInvalidatedFlightAnswersItsWaiterButIsNotCached(t *testing.T) {
	source := &fakeSource{}
	source.block(4)
	cache, _ := newTestCache(source, testTTL)

	served := make(chan int, 1)
	go func() {
		stats, _, err := cache.WeekStatsAsOf(context.Background(), 2026, 4)
		if err != nil {
			t.Errorf("waiting WeekStats: %v", err)
		}
		served <- fetchOrdinal(t, stats)
	}()
	waitForCalls(t, source, 1)
	cache.Invalidate(2026, 4)
	source.release(4)

	if got := <-served; got != 1 {
		t.Errorf("the waiter was served fetch %d, want fetch 1", got)
	}
	if _, _, err := cache.WeekStatsAsOf(context.Background(), 2026, 4); err != nil {
		t.Fatalf("later WeekStats: %v", err)
	}
	if got := source.callCount(); got != 2 {
		t.Errorf("upstream called %d times, want 2", got)
	}
}

func TestAStaleFlightsFailureDoesNotClobberANewerFlight(t *testing.T) {
	source := &fakeSource{}
	source.fail(errUpstream)
	source.block(4)
	staleBlock := source.blocks[4]
	cache, _ := newTestCache(source, testTTL)

	staleDone := make(chan error, 1)
	go func() {
		_, _, err := cache.WeekStatsAsOf(context.Background(), 2026, 4)
		staleDone <- err
	}()
	waitForCalls(t, source, 1)
	cache.Invalidate(2026, 4)

	// The newer flight gets its own gate, so the stale one can fail first.
	source.block(4)
	source.fail(nil)
	freshDone := make(chan error, 1)
	go func() {
		_, _, err := cache.WeekStatsAsOf(context.Background(), 2026, 4)
		freshDone <- err
	}()
	waitForCalls(t, source, 2)

	close(staleBlock)
	if err := <-staleDone; err == nil {
		t.Fatal("stale flight succeeded, want its failure")
	}
	source.release(4)
	if err := <-freshDone; err != nil {
		t.Fatalf("fresh flight: %v", err)
	}

	if _, _, err := cache.WeekStatsAsOf(context.Background(), 2026, 4); err != nil {
		t.Fatalf("later WeekStats: %v", err)
	}
	if got := source.callCount(); got != 2 {
		t.Errorf("upstream called %d times, want 2: the fresh flight's result was not kept", got)
	}
}
