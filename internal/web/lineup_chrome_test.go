package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strconv"
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
