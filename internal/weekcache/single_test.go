package weekcache

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// countingSingleSource returns its call ordinal, so a test can tell which fetch
// a caller was served from.
type countingSingleSource struct {
	mu    sync.Mutex
	calls int
	err   error
}

func (f *countingSingleSource) fetch(context.Context) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.err != nil {
		return 0, f.err
	}
	return f.calls, nil
}

func (f *countingSingleSource) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func newTestSingle(source *countingSingleSource) (*Single[int], *fakeClock) {
	clock := newFakeClock()
	return NewSingle("nflstate", source.fetch, CurrentWeekTTL, WithClock(clock.now)), clock
}

func TestSingleServesASecondRequestWithinTheHourFromCache(t *testing.T) {
	source := &countingSingleSource{}
	single, clock := newTestSingle(source)

	first, firstAt, err := single.AsOf(context.Background())
	if err != nil {
		t.Fatalf("first AsOf: %v", err)
	}
	clock.advance(10 * time.Minute)
	second, secondAt, err := single.AsOf(context.Background())
	if err != nil {
		t.Fatalf("second AsOf: %v", err)
	}

	if got := source.callCount(); got != 1 {
		t.Errorf("source called %d times, want 1", got)
	}
	if second != first || !secondAt.Equal(firstAt) {
		t.Errorf("second caller got fetch %d at %v, want fetch %d at %v", second, secondAt, first, firstAt)
	}
}

func TestSingleFetchesAgainAfterTheHour(t *testing.T) {
	source := &countingSingleSource{}
	single, clock := newTestSingle(source)

	if _, _, err := single.AsOf(context.Background()); err != nil {
		t.Fatalf("first AsOf: %v", err)
	}
	clock.advance(61 * time.Minute)
	if _, _, err := single.AsOf(context.Background()); err != nil {
		t.Fatalf("second AsOf: %v", err)
	}

	if got := source.callCount(); got != 2 {
		t.Errorf("source called %d times, want 2", got)
	}
}

func TestSingleDoesNotRememberAFailure(t *testing.T) {
	source := &countingSingleSource{err: errors.New("sleeper down")}
	single, clock := newTestSingle(source)

	if _, _, err := single.AsOf(context.Background()); err == nil {
		t.Fatal("first AsOf succeeded; want the source's error")
	}
	clock.advance(time.Minute)
	single.AsOf(context.Background())

	if got := source.callCount(); got != 2 {
		t.Errorf("source called %d times, want 2", got)
	}
}

func TestASingleHitNamesNoWeek(t *testing.T) {
	source := &countingSingleSource{}
	single, clock := newTestSingle(source)
	logs := recordLogs(single.cache)

	single.AsOf(context.Background())
	clock.advance(10 * time.Minute)
	single.AsOf(context.Background())

	if len(*logs) != 2 || (*logs)[1] != "nflstate hit: current, 10m0s old" {
		t.Errorf("miss then hit logged %q, want the hit line second", *logs)
	}
}

func TestASingleMissNamesNoWeek(t *testing.T) {
	source := &countingSingleSource{}
	single, _ := newTestSingle(source)
	logs := recordLogs(single.cache)

	single.AsOf(context.Background())

	if len(*logs) != 1 || !strings.HasPrefix((*logs)[0], "nflstate miss: current fetched in ") {
		t.Errorf("miss logged %q, want the current-week miss line", *logs)
	}
}
