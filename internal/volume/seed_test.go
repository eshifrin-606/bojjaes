package volume

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/eshifrin/bojjaes/internal/lineup"
)

// addSourceWeek puts a week of valid rosters into fsys: bojjaes plus the named
// opponent, each roster's record id tagged with version so two versions of a
// week differ in contents.
func addSourceWeek(fsys fstest.MapFS, version, season, week, opponent string) fstest.MapFS {
	dir := season + "/" + week
	fsys[dir] = &fstest.MapFile{Mode: fs.ModeDir}
	for _, team := range []string{"bojjaes", opponent} {
		roster := fmt.Sprintf("id,name,position,team\n%s-%s,Alpha One,QB,BUF\n", version, team)
		fsys[dir+"/"+team+".csv"] = &fstest.MapFile{Data: []byte(roster)}
	}
	return fsys
}

func readMountWeek(t *testing.T, mount, season, week string) map[string]string {
	t.Helper()
	dir := filepath.Join(mount, "lineups", season, week)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading store week %s/%s: %v", season, week, err)
	}
	files := map[string]string{}
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("reading %s: %v", e.Name(), err)
		}
		files[e.Name()] = string(data)
	}
	return files
}

func mustWeekHash(t *testing.T, fsys fs.FS, season, week string) string {
	t.Helper()
	h, err := weekHash(fsys, season, week)
	if err != nil {
		t.Fatalf("weekHash(%s/%s): %v", season, week, err)
	}
	return h
}

func TestWeekHashIgnoresModTimes(t *testing.T) {
	a := addSourceWeek(fstest.MapFS{}, "v1", "2026", "2", "aroma")
	b := addSourceWeek(fstest.MapFS{}, "v1", "2026", "2", "aroma")
	b["2026/2/aroma.csv"].ModTime = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	if ha, hb := mustWeekHash(t, a, "2026", "2"), mustWeekHash(t, b, "2026", "2"); ha != hb {
		t.Errorf("hashes differ across mtimes: %q vs %q", ha, hb)
	}
}

func TestWeekHashDiffersWhenContentsDiffer(t *testing.T) {
	v1 := addSourceWeek(fstest.MapFS{}, "v1", "2026", "2", "aroma")
	v2 := addSourceWeek(fstest.MapFS{}, "v2", "2026", "2", "aroma")

	if h := mustWeekHash(t, v1, "2026", "2"); h == mustWeekHash(t, v2, "2026", "2") {
		t.Errorf("v1 and v2 both hash to %q", h)
	}
}

func TestWeekHashDiffersWhenAFileIsRenamed(t *testing.T) {
	wood := addSourceWeek(fstest.MapFS{}, "v1", "2026", "2", "wood")
	aroma := addSourceWeek(fstest.MapFS{}, "v1", "2026", "2", "aroma")
	// Every file holds the same bytes, so only the names can tell the weeks
	// apart: aroma and wood sort on opposite sides of bojjaes.
	same := wood["2026/2/bojjaes.csv"].Data
	wood["2026/2/wood.csv"].Data = same
	aroma["2026/2/aroma.csv"].Data = same

	if h := mustWeekHash(t, wood, "2026", "2"); h == mustWeekHash(t, aroma, "2026", "2") {
		t.Errorf("wood.csv and aroma.csv weeks both hash to %q", h)
	}
}

func TestWeekHashKeepsBytesWithinTheirFile(t *testing.T) {
	// Without length prefixes, 1.csv's contents could spell out 2.csv's
	// name-and-separator and collide with the two-file week.
	twoFiles := fstest.MapFS{
		"2026/2":       &fstest.MapFile{Mode: fs.ModeDir},
		"2026/2/1.csv": &fstest.MapFile{Data: []byte("a")},
		"2026/2/2.csv": &fstest.MapFile{Data: []byte("bc")},
	}
	oneFile := fstest.MapFS{
		"2026/2":       &fstest.MapFile{Mode: fs.ModeDir},
		"2026/2/1.csv": &fstest.MapFile{Data: []byte("a2.csv\x00bc")},
	}

	if h := mustWeekHash(t, twoFiles, "2026", "2"); h == mustWeekHash(t, oneFile, "2026", "2") {
		t.Errorf("two-file and one-file weeks both hash to %q", h)
	}
}

func mustSeed(t *testing.T, src fs.FS, mount string) {
	t.Helper()
	if err := Seed(src, mount, t.Logf); err != nil {
		t.Fatalf("Seed: %v", err)
	}
}

func sourceWeekFiles(src fstest.MapFS, season, week string) map[string]string {
	prefix := season + "/" + week + "/"
	files := map[string]string{}
	for name, f := range src {
		if rest, ok := strings.CutPrefix(name, prefix); ok && !strings.Contains(rest, "/") {
			files[rest] = string(f.Data)
		}
	}
	return files
}

func TestSeedCopiesEveryWeekIntoAnEmptyMount(t *testing.T) {
	src := addSourceWeek(fstest.MapFS{}, "v1", "2026", "1", "wood")
	addSourceWeek(src, "v1", "2026", "2", "aroma")
	mount := t.TempDir()

	mustSeed(t, src, mount)

	for _, week := range []string{"1", "2"} {
		if got, want := readMountWeek(t, mount, "2026", week), sourceWeekFiles(src, "2026", week); !maps.Equal(got, want) {
			t.Errorf("store week 2026/%s = %v, want %v", week, got, want)
		}
	}
}

func readManifest(t *testing.T, mount string) map[string]string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(mount, "seed-manifest.json"))
	if err != nil {
		t.Fatalf("reading manifest: %v", err)
	}
	var m struct {
		Weeks map[string]string `json:"weeks"`
	}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("parsing manifest %q: %v", data, err)
	}
	return m.Weeks
}

func TestSeedRecordsEachCopiedWeekInTheManifest(t *testing.T) {
	src := addSourceWeek(fstest.MapFS{}, "v1", "2026", "1", "wood")
	addSourceWeek(src, "v1", "2026", "2", "aroma")
	mount := t.TempDir()

	mustSeed(t, src, mount)

	want := map[string]string{
		"2026/1": mustWeekHash(t, src, "2026", "1"),
		"2026/2": mustWeekHash(t, src, "2026", "2"),
	}
	if got := readManifest(t, mount); !maps.Equal(got, want) {
		t.Errorf("manifest weeks = %v, want %v", got, want)
	}
}

func TestSeedSkipsEntriesThatAreNotNumberedWeeks(t *testing.T) {
	src := addSourceWeek(fstest.MapFS{}, "v1", "2026", "1", "wood")
	src["lost+found/1/bojjaes.csv"] = &fstest.MapFile{Data: []byte("recovered")}
	src["notes.txt"] = &fstest.MapFile{Data: []byte("not a season")}
	addSourceWeek(src, "v1", "2026", "draft", "aroma")
	mount := t.TempDir()

	mustSeed(t, src, mount)

	for _, name := range []string{"lost+found", "notes.txt", "2026/draft"} {
		if _, err := os.Stat(filepath.Join(mount, "lineups", name)); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("store has %s (stat err %v), want it skipped", name, err)
		}
	}
	if got := readManifest(t, mount); len(got) != 1 || got["2026/1"] == "" {
		t.Errorf("manifest weeks = %v, want only 2026/1", got)
	}
}

func TestSeedCreatesLineupsBesideAFreshVolumesLostAndFound(t *testing.T) {
	src := addSourceWeek(fstest.MapFS{}, "v1", "2026", "1", "wood")
	addSourceWeek(src, "v1", "2026", "2", "aroma")
	mount := t.TempDir()
	lostFound := filepath.Join(mount, "lost+found")
	if err := os.Mkdir(lostFound, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lostFound, "#123"), []byte("orphan"), 0o600); err != nil {
		t.Fatal(err)
	}

	mustSeed(t, src, mount)

	for _, week := range []string{"1", "2"} {
		if got, want := readMountWeek(t, mount, "2026", week), sourceWeekFiles(src, "2026", week); !maps.Equal(got, want) {
			t.Errorf("store week 2026/%s = %v, want %v", week, got, want)
		}
	}
	entries, err := os.ReadDir(lostFound)
	if err != nil || len(entries) != 1 || entries[0].Name() != "#123" {
		t.Errorf("lost+found = %v (err %v), want only #123", entries, err)
	}
	if data, _ := os.ReadFile(filepath.Join(lostFound, "#123")); string(data) != "orphan" {
		t.Errorf("lost+found/#123 = %q, want %q", data, "orphan")
	}
}

func writeStoreFile(t *testing.T, mount, season, week, name, contents string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(mount, "lineups", season, week, name), []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

const phoneEdit = "id,name,position,team\nphone-aroma,Beta Two,RB,MIA\n"

func TestSeedLeavesAWeekAloneWhenGitsCopyIsUnchanged(t *testing.T) {
	src := addSourceWeek(fstest.MapFS{}, "v1", "2026", "1", "wood")
	addSourceWeek(src, "v1", "2026", "2", "aroma")
	mount := t.TempDir()
	mustSeed(t, src, mount)
	writeStoreFile(t, mount, "2026", "2", "aroma.csv", phoneEdit)

	mustSeed(t, src, mount)

	if got := readMountWeek(t, mount, "2026", "2")["aroma.csv"]; got != phoneEdit {
		t.Errorf("store 2026/2 aroma.csv = %q, want the phone edit %q", got, phoneEdit)
	}
}

func TestSeedReplacesAWeekWhenGitsCopyChanged(t *testing.T) {
	v1 := addSourceWeek(fstest.MapFS{}, "v1", "2026", "2", "aroma")
	v2 := addSourceWeek(fstest.MapFS{}, "v2", "2026", "2", "aroma")
	mount := t.TempDir()
	mustSeed(t, v1, mount)

	mustSeed(t, v2, mount)

	if got, want := readMountWeek(t, mount, "2026", "2"), sourceWeekFiles(v2, "2026", "2"); !maps.Equal(got, want) {
		t.Errorf("store week 2026/2 = %v, want v2 %v", got, want)
	}
	if got, want := readManifest(t, mount)["2026/2"], mustWeekHash(t, v2, "2026", "2"); got != want {
		t.Errorf("manifest 2026/2 = %q, want v2's hash %q", got, want)
	}
}

func TestSeedReplacesAStoreWeekTheManifestDoesNotRecord(t *testing.T) {
	src := addSourceWeek(fstest.MapFS{}, "v1", "2026", "2", "aroma")
	mount := t.TempDir()
	if err := os.MkdirAll(filepath.Join(mount, "lineups", "2026", "2"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeStoreFile(t, mount, "2026", "2", "aroma.csv", phoneEdit)
	writeStoreFile(t, mount, "2026", "2", "bojjaes.csv", phoneEdit)

	mustSeed(t, src, mount)

	if got, want := readMountWeek(t, mount, "2026", "2"), sourceWeekFiles(src, "2026", "2"); !maps.Equal(got, want) {
		t.Errorf("store week 2026/2 = %v, want source %v", got, want)
	}
	if got, want := readManifest(t, mount)["2026/2"], mustWeekHash(t, src, "2026", "2"); got != want {
		t.Errorf("manifest 2026/2 = %q, want %q", got, want)
	}
}

func TestSeedWritesNothingWhenNoWeekChanged(t *testing.T) {
	src := addSourceWeek(fstest.MapFS{}, "v1", "2026", "1", "wood")
	addSourceWeek(src, "v1", "2026", "2", "aroma")
	mount := t.TempDir()
	mustSeed(t, src, mount)
	manifestPath := filepath.Join(mount, "seed-manifest.json")
	old := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(manifestPath, old, old); err != nil {
		t.Fatal(err)
	}
	before := readManifest(t, mount)
	weeksBefore := []map[string]string{readMountWeek(t, mount, "2026", "1"), readMountWeek(t, mount, "2026", "2")}

	mustSeed(t, src, mount)

	info, err := os.Stat(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(old) {
		t.Errorf("manifest mtime = %v, want untouched %v", info.ModTime(), old)
	}
	if got := readManifest(t, mount); !maps.Equal(got, before) {
		t.Errorf("manifest weeks = %v, want %v", got, before)
	}
	for i, week := range []string{"1", "2"} {
		if got := readMountWeek(t, mount, "2026", week); !maps.Equal(got, weeksBefore[i]) {
			t.Errorf("store week 2026/%s = %v, want %v", week, got, weeksBefore[i])
		}
	}
}

func TestSeedReplacesAWeekWhole(t *testing.T) {
	old := addSourceWeek(fstest.MapFS{}, "v1", "2026", "2", "wood")
	src := addSourceWeek(fstest.MapFS{}, "v1", "2026", "2", "aroma")
	mount := t.TempDir()
	mustSeed(t, old, mount)

	mustSeed(t, src, mount)

	if got, want := readMountWeek(t, mount, "2026", "2"), sourceWeekFiles(src, "2026", "2"); !maps.Equal(got, want) {
		t.Errorf("store week 2026/2 = %v, want exactly %v", got, want)
	}
	ours, theirs, err := lineup.New(os.DirFS(filepath.Join(mount, "lineups"))).Matchup(2026, 2)
	if err != nil || ours != "bojjaes" || theirs != "aroma" {
		t.Errorf("Matchup(2026, 2) = %q, %q, %v; want bojjaes, aroma", ours, theirs, err)
	}
}

func TestSeedReplacingAWeekLeavesItsNeighbourAlone(t *testing.T) {
	v1 := addSourceWeek(fstest.MapFS{}, "v1", "2026", "1", "wood")
	addSourceWeek(v1, "v1", "2026", "2", "aroma")
	v2 := addSourceWeek(fstest.MapFS{}, "v1", "2026", "1", "wood")
	addSourceWeek(v2, "v2", "2026", "2", "aroma")
	mount := t.TempDir()
	mustSeed(t, v1, mount)
	writeStoreFile(t, mount, "2026", "1", "wood.csv", phoneEdit)
	before := readMountWeek(t, mount, "2026", "1")

	mustSeed(t, v2, mount)

	if got := readMountWeek(t, mount, "2026", "1"); !maps.Equal(got, before) {
		t.Errorf("store week 2026/1 = %v, want untouched %v", got, before)
	}
}

func TestSeedKeepsWeeksGitNoLongerHas(t *testing.T) {
	src := addSourceWeek(fstest.MapFS{}, "v1", "2026", "1", "wood")
	mount := t.TempDir()
	store := addSourceWeek(fstest.MapFS{}, "phone", "2026", "9", "aroma")
	mustSeed(t, store, mount)
	storeWeek := readMountWeek(t, mount, "2026", "9")
	entries := map[string]string{"2026/9": readManifest(t, mount)["2026/9"], "2026/3": "deadbeef"}
	data, err := json.Marshal(map[string]any{"weeks": entries})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mount, "seed-manifest.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	mustSeed(t, src, mount)

	if got := readMountWeek(t, mount, "2026", "9"); !maps.Equal(got, storeWeek) {
		t.Errorf("store week 2026/9 = %v, want kept %v", got, storeWeek)
	}
	got := readManifest(t, mount)
	for key, want := range entries {
		if got[key] != want {
			t.Errorf("manifest %s = %q, want kept %q", key, got[key], want)
		}
	}
}

type logRecorder struct{ lines []string }

func (r *logRecorder) logf(format string, args ...any) {
	r.lines = append(r.lines, fmt.Sprintf(format, args...))
}

func (r *logRecorder) anyLine(subs ...string) bool {
	for _, line := range r.lines {
		all := true
		for _, s := range subs {
			all = all && strings.Contains(line, s)
		}
		if all {
			return true
		}
	}
	return false
}

func TestSeedLogsAPhoneEditItOverwrites(t *testing.T) {
	v1 := addSourceWeek(fstest.MapFS{}, "v1", "2026", "2", "aroma")
	v2 := addSourceWeek(fstest.MapFS{}, "v2", "2026", "2", "aroma")
	mount := t.TempDir()
	mustSeed(t, v1, mount)
	writeStoreFile(t, mount, "2026", "2", "aroma.csv", phoneEdit)
	var log logRecorder

	if err := Seed(v2, mount, log.logf); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	if !log.anyLine("2026/2") {
		t.Errorf("no log line names 2026/2; got %q", log.lines)
	}
	if !log.anyLine("aroma.csv", fmt.Sprintf("%q", phoneEdit)) {
		t.Errorf("no log line carries aroma.csv and its phone-edited contents; got %q", log.lines)
	}
}

func TestSeedDoesNotLogReplacingWhatGitLastSeeded(t *testing.T) {
	v1 := addSourceWeek(fstest.MapFS{}, "v1", "2026", "2", "aroma")
	v2 := addSourceWeek(fstest.MapFS{}, "v2", "2026", "2", "aroma")
	mount := t.TempDir()
	mustSeed(t, v1, mount)
	var log logRecorder

	if err := Seed(v2, mount, log.logf); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	if got, want := readMountWeek(t, mount, "2026", "2"), sourceWeekFiles(v2, "2026", "2"); !maps.Equal(got, want) {
		t.Errorf("store week 2026/2 = %v, want v2 %v", got, want)
	}
	if log.anyLine("overwrit") {
		t.Errorf("logged an overwrite: %q", log.lines)
	}
}

func TestSeedDoesNotLogSeedingAFreshWeek(t *testing.T) {
	src := addSourceWeek(fstest.MapFS{}, "v1", "2026", "2", "aroma")
	mount := t.TempDir()
	var log logRecorder

	if err := Seed(src, mount, log.logf); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	if log.anyLine("overwrit") {
		t.Errorf("logged an overwrite: %q", log.lines)
	}
}

func TestSeedLogsAStoreOnlyWeekGitNowAdds(t *testing.T) {
	src := addSourceWeek(fstest.MapFS{}, "git", "2026", "9", "aroma")
	mount := t.TempDir()
	if err := os.MkdirAll(filepath.Join(mount, "lineups", "2026", "9"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeStoreFile(t, mount, "2026", "9", "aroma.csv", phoneEdit)
	var log logRecorder

	if err := Seed(src, mount, log.logf); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	if !log.anyLine("aroma.csv", fmt.Sprintf("%q", phoneEdit)) {
		t.Errorf("phone-set 2026/9 not logged; got %q", log.lines)
	}
	if got, want := readMountWeek(t, mount, "2026", "9"), sourceWeekFiles(src, "2026", "9"); !maps.Equal(got, want) {
		t.Errorf("store week 2026/9 = %v, want git's %v", got, want)
	}
}

func TestSeedDoesNotLogAWeekAlreadyHoldingGitsNewCopy(t *testing.T) {
	src := addSourceWeek(fstest.MapFS{}, "v1", "2026", "2", "aroma")
	mount := t.TempDir()
	mustSeed(t, src, mount)
	// A crash after the copy but before the manifest write leaves no entry.
	if err := os.Remove(filepath.Join(mount, "seed-manifest.json")); err != nil {
		t.Fatal(err)
	}
	var log logRecorder

	if err := Seed(src, mount, log.logf); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	if log.anyLine("overwrit") {
		t.Errorf("logged an overwrite: %q", log.lines)
	}
	if got, want := readManifest(t, mount)["2026/2"], mustWeekHash(t, src, "2026", "2"); got != want {
		t.Errorf("manifest 2026/2 = %q, want %q", got, want)
	}
}

func TestSeedReseedsEveryWeekWhenTheManifestIsUnreadable(t *testing.T) {
	src := addSourceWeek(fstest.MapFS{}, "v1", "2026", "1", "wood")
	addSourceWeek(src, "v1", "2026", "2", "aroma")
	mount := t.TempDir()
	mustSeed(t, src, mount)
	writeStoreFile(t, mount, "2026", "2", "aroma.csv", phoneEdit)
	if err := os.WriteFile(filepath.Join(mount, "seed-manifest.json"), []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	var log logRecorder

	if err := Seed(src, mount, log.logf); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	if !log.anyLine("manifest") {
		t.Errorf("no log line says the manifest was unreadable; got %q", log.lines)
	}
	if !log.anyLine("aroma.csv", fmt.Sprintf("%q", phoneEdit)) {
		t.Errorf("overwritten phone edit not logged; got %q", log.lines)
	}
	for _, week := range []string{"1", "2"} {
		if got, want := readMountWeek(t, mount, "2026", week), sourceWeekFiles(src, "2026", week); !maps.Equal(got, want) {
			t.Errorf("store week 2026/%s = %v, want %v", week, got, want)
		}
	}
	want := map[string]string{
		"2026/1": mustWeekHash(t, src, "2026", "1"),
		"2026/2": mustWeekHash(t, src, "2026", "2"),
	}
	if got := readManifest(t, mount); !maps.Equal(got, want) {
		t.Errorf("manifest weeks = %v, want %v", got, want)
	}
}

func TestSeedFailsOnAnUnwritableMount(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes through directory permissions")
	}
	src := addSourceWeek(fstest.MapFS{}, "v1", "2026", "1", "wood")
	mount := t.TempDir()
	if err := os.Chmod(mount, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(mount, 0o755) })

	if err := Seed(src, mount, t.Logf); err == nil {
		t.Error("Seed into a read-only mount returned nil, want an error")
	}
}
