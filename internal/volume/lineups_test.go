package volume

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

func mustLineups(t *testing.T, volume string) fs.FS {
	t.Helper()
	tree, err := Lineups(data.Lineups, volume, t.Logf)
	if err != nil {
		t.Fatalf("Lineups(%q): %v", volume, err)
	}
	return tree
}

func TestLineupsOverAnEmptyVolumeIsSeededFromEmbedded(t *testing.T) {
	volume := t.TempDir()

	tree := lineup.New(mustLineups(t, volume))

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

func TestLineupsServesAStoreEditMadeAfterSeeding(t *testing.T) {
	volume := t.TempDir()
	mustLineups(t, volume)
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

	_, got, err := lineup.New(mustLineups(t, volume)).Matchup(season, week)

	if err != nil || got != "phoneedit" {
		t.Errorf("Matchup(%d, %d) opponent = %q, %v; want %q", season, week, got, err, "phoneedit")
	}
}

func TestLineupsWithoutVolumeIsEmbeddedAndWritesNothing(t *testing.T) {
	cwd := t.TempDir()
	t.Chdir(cwd)

	tree, err := Lineups(data.Lineups, "", t.Logf)

	if err != nil {
		t.Fatalf("Lineups(\"\"): %v", err)
	}
	if tree != data.Lineups {
		t.Errorf("Lineups(\"\") = %v, want data.Lineups", tree)
	}
	if entries, err := os.ReadDir(cwd); err != nil || len(entries) != 0 {
		t.Errorf("working directory holds %v (%v), want nothing", entries, err)
	}
}

func TestLineupsFailsOnAnUnwritableVolume(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes through directory permissions")
	}
	volume := t.TempDir()
	if err := os.Chmod(volume, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(volume, 0o755) })

	if _, err := Lineups(data.Lineups, volume, t.Logf); err == nil {
		t.Error("Lineups over a read-only volume returned nil error")
	}
}
