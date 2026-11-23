package web

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/eshifrin/bojjaes/internal/lineup"
)

// pickInChrome serves the lineup page at path with a script after the page's
// own that sets the slot select at index to id as a reader would, then
// returns the DOM Chrome is left with.
func pickInChrome(t *testing.T, f *lineupFixture, path string, index int, id string) string {
	t.Helper()
	if testing.Short() {
		t.Skip("running the card script needs Chrome; skipped under -short")
	}
	chrome := chromePath()
	if chrome == "" {
		t.Skip("no Chrome: set BOJJAES_CHROME or install Google Chrome")
	}

	pick := `<script>
  var select = document.querySelectorAll(".row select")[` + strconv.Itoa(index) + `];
  select.value = "` + id + `";
  select.dispatchEvent(new Event("change", { bubbles: true }));
</script>`
	body := f.get(path).Body.String() + pick
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/lineup" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(body))
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, chrome, "--headless", "--disable-gpu", "--dump-dom", srv.URL+path).Output()
	if err != nil {
		t.Fatalf("running Chrome: %v", err)
	}
	return string(out)
}

func TestPickingAPlayerRedrawsTheSlotsCard(t *testing.T) {
	f := newLineupFixture()
	f.tree = lineup.New(weekFS(2026, 9, map[string]string{"wood": woodLine(woodWeek9...)}))
	f.games.week = upcomingDET()

	dom := pickInChrome(t, f, "/lineup?team=wood", 1, "103")

	names, metas := cardNames(dom), cardMetas(dom)
	if len(names) != lineup.StarterCount || len(metas) != lineup.StarterCount {
		t.Fatalf("found %d card names and %d meta lines, want %d:\n%s", len(names), len(metas), lineup.StarterCount, dom)
	}
	if names[1] != "Derrick Henry" || metas[1] != "RB · BAL · Bye" {
		t.Errorf("RB card = %q / %q, want %q / %q", names[1], metas[1], "Derrick Henry", "RB · BAL · Bye")
	}
}

type pressRun struct {
	posts, clipboardWrites, loads int32
	// csvQueries holds the query of each GET /lineup.csv, in order.
	csvQueries []url.Values
	// reloads holds the query of each GET /lineup after the first.
	reloads []url.Values
	dom     string
}

// pressInChrome serves the lineup page with a Save, swaps the clipboard for
// one that reports each write to the server, presses the button matching
// selector, and counts what reached the server.
func pressInChrome(t *testing.T, f *lineupFixture, path, selector string) pressRun {
	t.Helper()
	return press(t, f, path, selector, true)
}

// pressWithoutClipboardInChrome is pressInChrome on a browser with no
// clipboard API at all.
func pressWithoutClipboardInChrome(t *testing.T, f *lineupFixture, path, selector string) pressRun {
	t.Helper()
	return press(t, f, path, selector, false)
}

func press(t *testing.T, f *lineupFixture, path, selector string, clipboard bool) pressRun {
	t.Helper()
	if testing.Short() {
		t.Skip("pressing the form's buttons needs Chrome; skipped under -short")
	}
	chrome := chromePath()
	if chrome == "" {
		t.Skip("no Chrome: set BOJJAES_CHROME or install Google Chrome")
	}

	press := `<script>
  var report = function () { return fetch("/clipboard", { method: "POST" }); };
  Object.defineProperty(navigator, "clipboard", { configurable: true, value: {
    write: report,
    writeText: report
  } });
  document.querySelector(` + strconv.Quote(selector) + `).click();
</script>`
	page := f.get(path).Body.String()
	if !clipboard {
		// Before the page's own script, which checks for the clipboard as it runs.
		page = strings.Replace(page, `<meta charset="utf-8">`, `<meta charset="utf-8"><script>
  Object.defineProperty(navigator, "clipboard", { configurable: true, value: undefined });
</script>`, 1)
		press = `<script>document.querySelector(` + strconv.Quote(selector) + `).click();</script>`
	}
	var run pressRun
	var loads int32
	var mu sync.Mutex
	csv := LineupCSVHandler(f.rosters, f.tree, f.current)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /lineup.csv", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		run.csvQueries = append(run.csvQueries, r.URL.Query())
		mu.Unlock()
		csv.ServeHTTP(w, r)
	})
	mux.HandleFunc("GET /lineup", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if atomic.AddInt32(&loads, 1) == 1 {
			w.Write([]byte(page + press))
			return
		}
		mu.Lock()
		run.reloads = append(run.reloads, r.URL.Query())
		mu.Unlock()
		w.Write(f.get(r.URL.RequestURI()).Body.Bytes())
	})
	mux.HandleFunc("GET /lineup.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("wood lineup"))
	})
	mux.HandleFunc("POST /lineup", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&run.posts, 1)
		w.Write([]byte("saved"))
	})
	mux.HandleFunc("POST /clipboard", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&run.clipboardWrites, 1)
	})
	mux.HandleFunc("GET /favicon.ico", http.NotFound)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	dom, err := exec.CommandContext(ctx, chrome, "--headless", "--disable-gpu", "--dump-dom",
		"--virtual-time-budget=2000", srv.URL+path).Output()
	if err != nil {
		t.Fatalf("running Chrome: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	run.loads = atomic.LoadInt32(&loads)
	run.dom = string(dom)
	return run
}

func TestSavePostsTheFormAndCopyCopies(t *testing.T) {
	f := newLineupFixture()
	f.tree = lineup.New(weekFS(2026, 9, map[string]string{"wood": woodLine(woodWeek9...)}))
	f.saver = &fakeSaver{}

	tests := []struct {
		name, selector                 string
		wantPosts, wantClipboardWrites int32
	}{
		{"Save", `button[name="save"]`, 1, 0},
		{"Copy", `.actions button:not([name])`, 0, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := pressInChrome(t, f, "/lineup?team=wood", tt.selector)

			if run.posts != tt.wantPosts || run.clipboardWrites != tt.wantClipboardWrites {
				t.Errorf("posts = %d, clipboard writes = %d; want %d and %d", run.posts, run.clipboardWrites, tt.wantPosts, tt.wantClipboardWrites)
			}
		})
	}
}

var statusPattern = regexp.MustCompile(`<p class="status">(.*?)</p>`)

func TestAnEditMarksASavedWeekUnsaved(t *testing.T) {
	f := newLineupFixture()
	f.tree = lineup.New(weekFS(2026, 9, map[string]string{"wood": woodLine(woodWeek9...)}))

	dom := pickInChrome(t, f, "/lineup?team=wood", 1, "103")

	m := statusPattern.FindStringSubmatch(dom)
	if m == nil || m[1] != "Unsaved changes" {
		t.Errorf("status = %v, want %q", m, "Unsaved changes")
	}
}

func TestAnEditHighlightsASavedWeeksSave(t *testing.T) {
	f := newLineupFixture()
	f.saver = &fakeSaver{}
	f.tree = lineup.New(weekFS(2026, 9, map[string]string{"wood": woodLine(woodWeek9...)}))

	dom := pickInChrome(t, f, "/lineup?team=wood", 1, "103")

	names, attrs := actionNames(t, dom)
	if len(names) == 0 || names[0] != "Save" {
		t.Fatalf("action row after an edit = %q, want Save first", names)
	}
	if !strings.Contains(attrs[0], `class="primary"`) {
		t.Errorf("Save after an edit = %q, want class=\"primary\"", attrs[0])
	}
}

var actionsMeasuredPattern = regexp.MustCompile(`<pre id="measured">(.*?)</pre>`)

// measureActionsInChrome lays the lineup page out in Chrome, script stripped,
// and decodes into out the result of the JS function measure applied to each
// action row button, left to right.
func measureActionsInChrome(t *testing.T, f *lineupFixture, path, measure string, out any) {
	t.Helper()
	if testing.Short() {
		t.Skip("measuring the action row needs Chrome; skipped under -short")
	}
	chrome := chromePath()
	if chrome == "" {
		t.Skip("no Chrome: set BOJJAES_CHROME or install Google Chrome")
	}

	script := `<pre id="measured"></pre>
<script>
  document.getElementById("measured").textContent = JSON.stringify(Array.prototype.map.call(
    document.querySelectorAll(".actions button"), ` + measure + `));
</script>`
	page := scriptPattern.ReplaceAllString(f.get(path).Body.String(), "") + script
	file := filepath.Join(t.TempDir(), "page.html")
	if err := os.WriteFile(file, []byte(page), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	dom, err := exec.CommandContext(ctx, chrome, "--headless", "--disable-gpu", "--dump-dom", "file://"+file).Output()
	if err != nil {
		t.Fatalf("running Chrome: %v", err)
	}
	m := actionsMeasuredPattern.FindSubmatch(dom)
	if m == nil {
		t.Fatalf("Chrome output has no measured element:\n%s", dom)
	}
	if err := json.Unmarshal(m[1], out); err != nil {
		t.Fatalf("decoding %q: %v", m[1], err)
	}
}

func actionWidthsInChrome(t *testing.T, f *lineupFixture, path string) []float64 {
	t.Helper()
	var widths []float64
	measureActionsInChrome(t, f, path, `function (b) { return b.getBoundingClientRect().width; }`, &widths)
	return widths
}

func TestTheActionRowSplitsItsWidth(t *testing.T) {
	near := func(a, b float64) bool { return math.Abs(a-b) <= 2 }

	t.Run("with a saver", func(t *testing.T) {
		f := newLineupFixture()
		f.saver = &fakeSaver{}

		w := actionWidthsInChrome(t, f, "/lineup?team=wood")

		if len(w) != 3 || !near(w[0], 2*w[1]) || !near(w[1], w[2]) {
			t.Errorf("Save, Copy, Download widths = %v, want Save ≈ 2×Copy and Copy ≈ Download", w)
		}
	})
	t.Run("with this week saved", func(t *testing.T) {
		f := newLineupFixture()
		f.saver = &fakeSaver{}
		f.tree = lineup.New(weekFS(2026, 9, map[string]string{"wood": woodLine(woodWeek9...)}))

		w := actionWidthsInChrome(t, f, "/lineup?team=wood")

		if len(w) != 3 || !near(w[0], 2*w[1]) || !near(w[1], w[2]) {
			t.Errorf("Save, Copy, Download widths = %v, want Save ≈ 2×Copy and Copy ≈ Download", w)
		}
	})
	t.Run("without a saver", func(t *testing.T) {
		w := actionWidthsInChrome(t, newLineupFixture(), "/lineup?team=wood")

		if len(w) != 2 || !near(w[0], 3*w[1]) {
			t.Errorf("Copy, Download widths = %v, want Copy ≈ 3×Download", w)
		}
	})
}

const downloadSelector = `button[aria-label="Download"]`

func TestDownloadingAValidLineupFetchesItsFileWithoutCopyingOrReloading(t *testing.T) {
	f := newLineupFixture()
	path := proposalURL("/lineup", woodWeek9...)
	page, _ := url.Parse(path)

	run := pressInChrome(t, f, path, downloadSelector)

	if len(run.csvQueries) == 0 {
		t.Fatalf("no GET /lineup.csv; clipboard writes = %d, loads = %d", run.clipboardWrites, run.loads)
	}
	for key, want := range page.Query() {
		if got := run.csvQueries[0][key]; !slices.Equal(got, want) {
			t.Errorf("/lineup.csv %s = %q, want %q", key, got, want)
		}
	}
	if run.clipboardWrites != 0 || run.loads != 1 {
		t.Errorf("clipboard writes = %d, page loads = %d; want 0 and 1", run.clipboardWrites, run.loads)
	}
}

func TestDownloadingADuplicateReloadsTheFormListingIt(t *testing.T) {
	path := proposalURL("/lineup", woodWeek9With(4, "102")...)
	page, _ := url.Parse(path)

	run := pressInChrome(t, newLineupFixture(), path, downloadSelector)

	if len(run.reloads) != 1 {
		t.Fatalf("reloads = %v, want one", run.reloads)
	}
	for key, want := range page.Query() {
		if got := run.reloads[0][key]; !slices.Equal(got, want) {
			t.Errorf("reloaded %s = %q, want %q", key, got, want)
		}
	}
	if !strings.Contains(run.dom, "is in two slots") {
		t.Errorf("the reloaded form does not list the duplicate:\n%s", run.dom)
	}
}

func TestWithoutAClipboardDownloadingADuplicateStillReloadsTheForm(t *testing.T) {
	path := proposalURL("/lineup", woodWeek9With(4, "102")...)
	page, _ := url.Parse(path)

	run := pressWithoutClipboardInChrome(t, newLineupFixture(), path, downloadSelector)

	if len(run.reloads) != 1 {
		t.Fatalf("reloads = %v, want one; page:\n%s", run.reloads, run.dom)
	}
	for key, want := range page.Query() {
		if got := run.reloads[0][key]; !slices.Equal(got, want) {
			t.Errorf("reloaded %s = %q, want %q", key, got, want)
		}
	}
	if !strings.Contains(run.dom, `<ul class="problems">`) {
		t.Errorf("the reloaded form does not list the duplicate:\n%s", run.dom)
	}
}

func TestAnEditResetsCopiedAndLeavesDownloadEnabled(t *testing.T) {
	f := newLineupFixture()
	f.saver = &fakeSaver{}

	dom := pickInChrome(t, f, proposalURL("/lineup", woodWeek9...)+"#copied", 1, "103")

	names, attrs := actionNames(t, dom)
	if want := []string{"Save", "Copy", "Download"}; !slices.Equal(names, want) {
		t.Errorf("action row after an edit = %q, want %q", names, want)
	}
	if len(attrs) == 3 && strings.Contains(attrs[2], "disabled") {
		t.Errorf("Download is disabled after an edit: %q", attrs[2])
	}
}

func TestAnUnsavedWeeksSaveIsMutedGreen(t *testing.T) {
	f := newLineupFixture()
	f.saver = &fakeSaver{}

	var colors [][2]string
	measureActionsInChrome(t, f, "/lineup?team=wood",
		`function (b) { var s = getComputedStyle(b); return [s.backgroundColor, s.color]; }`, &colors)

	if want := [2]string{"rgb(79, 138, 91)", "rgb(255, 255, 255)"}; len(colors) == 0 || colors[0] != want {
		t.Errorf("action row colours = %v, want Save %v", colors, want)
	}
}
