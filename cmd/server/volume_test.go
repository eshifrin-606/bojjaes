package main

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/eshifrin/bojjaes/internal/lineup"
)

func TestResolveLineupVolumeUsesLineupVolume(t *testing.T) {
	got := resolveLineupVolume(fakeEnv(map[string]string{"LINEUP_VOLUME": "/data"}))

	if got != "/data" {
		t.Errorf("resolveLineupVolume = %q, want %q", got, "/data")
	}
}

func TestResolveLineupVolumeIsEmptyWhenUnset(t *testing.T) {
	got := resolveLineupVolume(fakeEnv(nil))

	if got != "" {
		t.Errorf("resolveLineupVolume = %q, want empty", got)
	}
}

func TestResolveLineupVolumeIsEmptyWhenEmpty(t *testing.T) {
	got := resolveLineupVolume(fakeEnv(map[string]string{"LINEUP_VOLUME": ""}))

	if got != "" {
		t.Errorf("resolveLineupVolume = %q, want empty", got)
	}
}

func embeddedLatest(t *testing.T) (season, week int) {
	t.Helper()
	seasons, err := fs.ReadDir(lineup.Embedded, ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range seasons {
		if n, err := strconv.Atoi(s.Name()); err == nil && n > season {
			season = n
		}
	}
	week, ok := lineup.New(lineup.Embedded).LatestWeek(season)
	if !ok {
		t.Fatalf("embedded season %d has no week", season)
	}
	return season, week
}

func TestLineupTreeWithoutVolumeIsEmbedded(t *testing.T) {
	season, week := embeddedLatest(t)

	got, ok := lineup.New(lineupTree("")).LatestWeek(season)

	if !ok || got != week {
		t.Errorf("LatestWeek(%d) = %d, %v; want %d, true", season, got, ok, week)
	}
}

func TestLineupTreeServesVolumeWeekAfterEmbedded(t *testing.T) {
	season, week := embeddedLatest(t)
	volume := t.TempDir()
	src := path.Join(strconv.Itoa(season), strconv.Itoa(week))
	dst := filepath.Join(volume, "lineups", strconv.Itoa(season), strconv.Itoa(week+1))
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	rosters, err := fs.ReadDir(lineup.Embedded, src)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rosters {
		data, err := fs.ReadFile(lineup.Embedded, path.Join(src, r.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dst, r.Name()), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got, ok := lineup.New(lineupTree(volume)).LatestWeek(season)

	if !ok || got != week+1 {
		t.Errorf("LatestWeek(%d) = %d, %v; want %d, true", season, got, ok, week+1)
	}
}
