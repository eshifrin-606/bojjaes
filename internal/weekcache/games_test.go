package weekcache

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/eshifrin/bojjaes/internal/games"
)

// These tests pin the games cache by its value type and label. The rules
// themselves are covered once, over stats, in the rest of this package.

// fakeGames counts its calls and stamps each week's one game with that call's
// ordinal, so a test can tell which fetch a caller was served from.
type fakeGames struct {
	mu    sync.Mutex
	calls int
	err   error
}

func (f *fakeGames) WeekGames(ctx context.Context, season, week int) (games.Week, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.err != nil {
		return games.Week{}, f.err
	}
	game := games.Game{Home: "KC", Away: "DEN", Status: strconv.Itoa(f.calls)}
	return games.NewWeek(season, week, []games.Game{game}), nil
}

func (f *fakeGames) fail(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = err
}

func (f *fakeGames) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func gamesFetchOrdinal(t *testing.T, week games.Week) int {
	t.Helper()
	game, ok := week.ForTeam("KC")
	if !ok {
		t.Fatal("served week holds no KC game")
	}
	n, err := strconv.Atoi(game.Status)
	if err != nil {
		t.Fatalf("game status %q is not a fetch ordinal", game.Status)
	}
	return n
}

func newTestGamesCache(source *fakeGames) (*Cache[games.Week], *fakeClock) {
	clock := newFakeClock()
	return New("gamescache", source.WeekGames, testTTL, WithClock(clock.now)), clock
}

func TestGamesAreServedFromCacheWithinTheTTLAndFetchedAgainAfter(t *testing.T) {
	source := &fakeGames{}
	cache, clock := newTestGamesCache(source)
	ctx := context.Background()

	if _, _, err := cache.AsOf(ctx, 2026, 5); err != nil {
		t.Fatalf("first AsOf: %v", err)
	}
	clock.advance(time.Minute)
	second, _, err := cache.AsOf(ctx, 2026, 5)
	if err != nil {
		t.Fatalf("second AsOf: %v", err)
	}
	if got := source.callCount(); got != 1 {
		t.Errorf("a minute apart: upstream called %d times, want 1", got)
	}
	if got := gamesFetchOrdinal(t, second); got != 1 {
		t.Errorf("a minute apart: served fetch %d, want fetch 1", got)
	}

	clock.advance(5 * time.Minute)
	third, _, err := cache.AsOf(ctx, 2026, 5)
	if err != nil {
		t.Fatalf("third AsOf: %v", err)
	}
	if got := source.callCount(); got != 2 {
		t.Errorf("six minutes on: upstream called %d times, want 2", got)
	}
	if got := gamesFetchOrdinal(t, third); got != 2 {
		t.Errorf("six minutes on: served fetch %d, want fetch 2", got)
	}
}

func TestAFailedGamesFetchIsNotStored(t *testing.T) {
	source := &fakeGames{}
	source.fail(errUpstream)
	cache, _ := newTestGamesCache(source)
	ctx := context.Background()

	if _, _, err := cache.AsOf(ctx, 2026, 5); !errors.Is(err, errUpstream) {
		t.Fatalf("failed AsOf returned %v, want %v", err, errUpstream)
	}
	source.fail(nil)

	week, _, err := cache.AsOf(ctx, 2026, 5)
	if err != nil {
		t.Fatalf("AsOf after the failure: %v", err)
	}
	if got := source.callCount(); got != 2 {
		t.Fatalf("upstream called %d times, want 2", got)
	}
	if got := gamesFetchOrdinal(t, week); got != 2 {
		t.Errorf("served fetch %d, want fetch 2", got)
	}
}

func TestInvalidatingAStatsWeekLeavesItsGamesCached(t *testing.T) {
	statsSource, gamesSource := &fakeSource{}, &fakeGames{}
	stats, clock := newTestCache(statsSource, testTTL)
	gameWeeks := New("gamescache", gamesSource.WeekGames, testTTL, WithClock(clock.now))
	ctx := context.Background()

	if _, _, err := stats.AsOf(ctx, 2026, 5); err != nil {
		t.Fatalf("stats AsOf: %v", err)
	}
	if _, _, err := gameWeeks.AsOf(ctx, 2026, 5); err != nil {
		t.Fatalf("first games AsOf: %v", err)
	}

	stats.Invalidate(2026, 5)
	clock.advance(time.Minute)

	if _, _, err := gameWeeks.AsOf(ctx, 2026, 5); err != nil {
		t.Fatalf("games AsOf after invalidation: %v", err)
	}
	if got := gamesSource.callCount(); got != 1 {
		t.Errorf("games upstream called %d times, want 1", got)
	}
}

// The label is what tells the two caches' lines apart in one process log.
func TestAGamesHitIsLoggedUnderItsOwnLabel(t *testing.T) {
	statsSource, gamesSource := &fakeSource{}, &fakeGames{}
	stats, clock := newTestCache(statsSource, testTTL)
	gameWeeks := New("gamescache", gamesSource.WeekGames, testTTL, WithClock(clock.now))
	statsLogs, gamesLogs := recordLogs(stats), recordLogs(gameWeeks)
	ctx := context.Background()

	for range 2 {
		if _, _, err := gameWeeks.AsOf(ctx, 2026, 5); err != nil {
			t.Fatalf("games AsOf: %v", err)
		}
		if _, _, err := stats.AsOf(ctx, 2026, 5); err != nil {
			t.Fatalf("stats AsOf: %v", err)
		}
		clock.advance(time.Minute)
	}

	if len(*gamesLogs) != 2 {
		t.Fatalf("games miss then hit logged %d lines, want 2: %q", len(*gamesLogs), *gamesLogs)
	}
	if got, want := (*gamesLogs)[1], "gamescache hit: 2026 week 5, 1m0s old"; got != want {
		t.Errorf("games hit logged %q, want %q", got, want)
	}
	if len(*statsLogs) != 2 || !strings.HasPrefix((*statsLogs)[1], "statscache hit:") {
		t.Errorf("stats miss then hit logged %q, want a statscache hit second", *statsLogs)
	}
	for _, line := range *statsLogs {
		if strings.Contains(line, "gamescache") {
			t.Errorf("stats cache logged a gamescache line: %q", line)
		}
	}
}
