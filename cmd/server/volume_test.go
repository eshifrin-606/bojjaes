package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	"github.com/eshifrin/bojjaes/internal/data"
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

func embeddedSeasons(t *testing.T) []int {
	t.Helper()
	entries, err := fs.ReadDir(data.Lineups, ".")
	if err != nil {
		t.Fatal(err)
	}
	var seasons []int
	for _, e := range entries {
		if n, err := strconv.Atoi(e.Name()); err == nil && e.IsDir() {
			seasons = append(seasons, n)
		}
	}
	if len(seasons) == 0 {
		t.Fatal("no embedded seasons")
	}
	return seasons
}

func mustLineupTree(t *testing.T, volume string) fs.FS {
	t.Helper()
	tree, err := lineupTree(volume, t.Logf)
	if err != nil {
		t.Fatalf("lineupTree(%q): %v", volume, err)
	}
	return tree
}

func TestLineupTreeOverAnEmptyVolumeIsSeededFromEmbedded(t *testing.T) {
	volume := t.TempDir()

	tree := lineup.New(mustLineupTree(t, volume))

	embedded := lineup.New(data.Lineups)
	for _, season := range embeddedSeasons(t) {
		want, wantOK := embedded.LatestWeek(season)
		got, ok := tree.LatestWeek(season)
		if got != want || ok != wantOK {
			t.Errorf("LatestWeek(%d) = %d, %v; want %d, %v", season, got, ok, want, wantOK)
		}
	}
	if _, err := os.Stat(filepath.Join(volume, "seed-manifest.json")); err != nil {
		t.Errorf("seed manifest: %v", err)
	}
}

func embeddedLatest(t *testing.T) (season, week int) {
	t.Helper()
	seasons := embeddedSeasons(t)
	season = slices.Max(seasons)
	week, ok := lineup.New(data.Lineups).LatestWeek(season)
	if !ok {
		t.Fatalf("embedded season %d has no week", season)
	}
	return season, week
}

func TestLineupTreeServesAStoreEditMadeAfterSeeding(t *testing.T) {
	volume := t.TempDir()
	mustLineupTree(t, volume)
	season, week := embeddedLatest(t)
	_, opponent, err := lineup.New(data.Lineups).Matchup(season, week)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(volume, "lineups", strconv.Itoa(season), strconv.Itoa(week))
	roster, err := os.ReadFile(filepath.Join(dir, opponent+".csv"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, opponent+".csv")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "phoneedit.csv"), roster, 0o644); err != nil {
		t.Fatal(err)
	}

	_, got, err := lineup.New(mustLineupTree(t, volume)).Matchup(season, week)

	if err != nil || got != "phoneedit" {
		t.Errorf("Matchup(%d, %d) opponent = %q, %v; want %q", season, week, got, err, "phoneedit")
	}
}

func TestLineupTreeWithoutVolumeIsEmbeddedAndWritesNothing(t *testing.T) {
	cwd := t.TempDir()
	t.Chdir(cwd)

	tree, err := lineupTree("", t.Logf)

	if err != nil {
		t.Fatalf("lineupTree(\"\"): %v", err)
	}
	if tree != data.Lineups {
		t.Errorf("lineupTree(\"\") = %v, want data.Lineups", tree)
	}
	if entries, err := os.ReadDir(cwd); err != nil || len(entries) != 0 {
		t.Errorf("working directory holds %v (%v), want nothing", entries, err)
	}
}

func TestLineupTreeFailsOnAnUnwritableVolume(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes through directory permissions")
	}
	volume := t.TempDir()
	if err := os.Chmod(volume, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(volume, 0o755) })

	if _, err := lineupTree(volume, t.Logf); err == nil {
		t.Error("lineupTree over a read-only volume returned nil error")
	}
}
