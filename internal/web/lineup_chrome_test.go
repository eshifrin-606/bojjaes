package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strconv"
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
	posts, clipboardWrites int32
}

// pressInChrome serves the lineup page with a Save, swaps the clipboard for
// one that reports each write to the server, presses the button matching
// selector, and counts what reached the server.
func pressInChrome(t *testing.T, f *lineupFixture, path, selector string) pressRun {
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
	var run pressRun
	var loads int32
	mux := http.NewServeMux()
	mux.HandleFunc("GET /lineup", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if atomic.AddInt32(&loads, 1) == 1 {
			w.Write([]byte(page + press))
			return
		}
		w.Write([]byte(page))
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
	if err := exec.CommandContext(ctx, chrome, "--headless", "--disable-gpu", "--dump-dom",
		"--virtual-time-budget=2000", srv.URL+path).Run(); err != nil {
		t.Fatalf("running Chrome: %v", err)
	}
	return pressRun{atomic.LoadInt32(&run.posts), atomic.LoadInt32(&run.clipboardWrites)}
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
