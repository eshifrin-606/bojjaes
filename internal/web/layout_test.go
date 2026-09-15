package web

import (
	"context"
	"encoding/json"
	"html"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/eshifrin/bojjaes/internal/lineup"
)

type rect struct {
	Top, Bottom, Left, Right, Width, Height float64
}

type columnLayout struct {
	Header      *rect
	H2          *rect
	Total       *rect
	FirstLi     *rect
	SecondLi    *rect
	FirstPlayer *rect
	Column      rect
}

type layout struct {
	Columns []columnLayout
}

func chromePath() string {
	if p := os.Getenv("BOJJAES_CHROME"); p != "" {
		return p
	}
	const macApp = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
	if _, err := os.Stat(macApp); err == nil {
		return macApp
	}
	for _, name := range []string{"chromium", "google-chrome", "chrome-headless-shell"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return ""
}

const measuringScript = `<pre id="layout"></pre>
<script>
  function box(el) {
    if (!el) return null;
    var r = el.getBoundingClientRect();
    return {Top: r.top, Bottom: r.bottom, Left: r.left, Right: r.right, Width: r.width, Height: r.height};
  }
  var columns = Array.prototype.map.call(document.querySelectorAll(".column"), function (c) {
    return {
      Header: box(c.querySelector("header")),
      H2: box(c.querySelector("h2")),
      Total: box(c.querySelector(".total")),
      FirstLi: box(c.querySelector("li")),
      SecondLi: box(c.querySelectorAll("li")[1]),
      FirstPlayer: box(c.querySelector(".player")),
      Column: box(c)
    };
  });
  document.getElementById("layout").textContent = JSON.stringify({Columns: columns});
</script>`

// fixtureCSS is page-only styling for the harness, kept out of the template.
const fixtureCSS = `<style></style>`

var layoutPattern = regexp.MustCompile(`(?s)<pre id="layout">(.*?)</pre>`)

// renderedLayout measures body as Chrome lays it out. The refresh script is
// stripped so a slow run can never reload the page mid-measurement.
func renderedLayout(t *testing.T, body string) layout {
	t.Helper()
	if testing.Short() {
		t.Skip("rendered layout needs Chrome; skipped under -short")
	}
	chrome := chromePath()
	if chrome == "" {
		t.Skip("no Chrome: set BOJJAES_CHROME or install Google Chrome")
	}

	page := scriptPattern.ReplaceAllString(body, "") + fixtureCSS + measuringScript
	file := filepath.Join(t.TempDir(), "page.html")
	if err := os.WriteFile(file, []byte(page), 0o600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, chrome, "--headless", "--disable-gpu", "--dump-dom", "file://"+file).Output()
	if err != nil {
		t.Fatalf("running Chrome: %v", err)
	}

	m := layoutPattern.FindSubmatch(out)
	if m == nil {
		t.Fatalf("Chrome output has no layout element:\n%s", out)
	}
	var l layout
	if err := json.Unmarshal([]byte(html.UnescapeString(strings.TrimSpace(string(m[1])))), &l); err != nil {
		t.Fatalf("decoding layout %q: %v", m[1], err)
	}
	return l
}

func TestTheHarnessMeasuresTheStarterList(t *testing.T) {
	rec := serve(Handler(lineup.New(fixtureWeek()), &fakeSource{weekStats: fixtureStats()}), http.MethodGet, "/2025/15")

	l := renderedLayout(t, rec.Body.String())

	if len(l.Columns) != 2 {
		t.Fatalf("measured %d columns, want 2", len(l.Columns))
	}
	for i, c := range l.Columns {
		if c.FirstLi == nil || c.FirstLi.Height == 0 {
			t.Errorf("column %d first li = %+v, want a non-zero height", i, c.FirstLi)
		}
	}
	if l.Columns[0].FirstLi != nil && l.Columns[1].FirstLi != nil && l.Columns[0].FirstLi.Top != l.Columns[1].FirstLi.Top {
		t.Errorf("first li tops = %v and %v, want equal", l.Columns[0].FirstLi.Top, l.Columns[1].FirstLi.Top)
	}
}

// narrowCardCSS makes the cards narrow enough to show short names, since
// headless Chrome will not open a phone-width window.
const narrowCardCSS = `<style>.matchup { max-width: 390px }</style>`

func TestStarterRowsLineUpAcrossCards(t *testing.T) {
	wrapping := slices.Clone(ourLine)
	wrapping[0].name = "Puka Nacua Nacua Nacua Nacua"
	weekTree := weekFS(2025, 15, map[string]string{
		"bojjaes": lineupCSV(wrapping),
		"wood":    lineupCSV(theirLine),
	})
	rec := serve(Handler(lineup.New(weekTree), &fakeSource{weekStats: fixtureStats()}), http.MethodGet, "/2025/15")

	l := renderedLayout(t, rec.Body.String()+narrowCardCSS)

	if len(l.Columns) != 2 {
		t.Fatalf("measured %d columns, want 2", len(l.Columns))
	}
	ours, theirs := l.Columns[0], l.Columns[1]
	// The li stretches to its shared row, so only the name itself shows the wrap.
	if ours.FirstPlayer.Height <= theirs.FirstPlayer.Height {
		t.Fatalf("first player heights = %v and %v, want the wrapping name's taller", ours.FirstPlayer.Height, theirs.FirstPlayer.Height)
	}
	if ours.SecondLi.Top != theirs.SecondLi.Top {
		t.Errorf("second li tops = %v and %v, want equal", ours.SecondLi.Top, theirs.SecondLi.Top)
	}
}

func TestTheTotalSitsOnTheTeamNameLine(t *testing.T) {
	rec := serve(Handler(lineup.New(fixtureWeek()), &fakeSource{weekStats: fixtureStats()}), http.MethodGet, "/2025/15")

	l := renderedLayout(t, rec.Body.String())

	if len(l.Columns) != 2 {
		t.Fatalf("measured %d columns, want 2", len(l.Columns))
	}
	for i, c := range l.Columns {
		total, h2, first := c.Total, c.H2, c.FirstLi
		if total.Top >= h2.Bottom || total.Bottom <= h2.Top {
			t.Errorf("column %d total %v–%v does not overlap h2 %v–%v vertically", i, total.Top, total.Bottom, h2.Top, h2.Bottom)
		}
		if total.Left < h2.Right {
			t.Errorf("column %d total left %v, want at or right of h2 right %v", i, total.Left, h2.Right)
		}
		if total.Bottom > first.Top || h2.Bottom > first.Top {
			t.Errorf("column %d total bottom %v and h2 bottom %v, want both at or above first li top %v", i, total.Bottom, h2.Bottom, first.Top)
		}
	}
}

func TestNothingSitsAboveTheHeading(t *testing.T) {
	rec := serve(Handler(lineup.New(fixtureWeek()), &fakeSource{weekStats: fixtureStats()}), http.MethodGet, "/2025/15")

	l := renderedLayout(t, rec.Body.String())

	if len(l.Columns) != 2 {
		t.Fatalf("measured %d columns, want 2", len(l.Columns))
	}
	for i, c := range l.Columns {
		if c.Header == nil {
			t.Fatalf("column %d has no header", i)
		}
		measured := map[string]*rect{"h2": c.H2, "total": c.Total, "first li": c.FirstLi, "second li": c.SecondLi, "first player": c.FirstPlayer}
		for name, r := range measured {
			if r != nil && r.Top < c.Header.Top {
				t.Errorf("column %d %s top %v is above header top %v", i, name, r.Top, c.Header.Top)
			}
		}
	}
}

func TestTheBandSpansTheCard(t *testing.T) {
	rec := serve(Handler(lineup.New(fixtureWeek()), &fakeSource{weekStats: fixtureStats()}), http.MethodGet, "/2025/15")

	l := renderedLayout(t, rec.Body.String())

	if len(l.Columns) != 2 {
		t.Fatalf("measured %d columns, want 2", len(l.Columns))
	}
	for i, c := range l.Columns {
		if c.Header == nil {
			t.Fatalf("column %d has no header", i)
		}
		// The column rect includes its 1px border; the band meets the border's inner edge.
		innerLeft, innerRight := c.Column.Left+1, c.Column.Right-1
		if math.Abs(c.Header.Left-innerLeft) > 1 || math.Abs(c.Header.Right-innerRight) > 1 {
			t.Errorf("column %d header spans %v–%v, want within 1px of the card's inner edges %v–%v", i, c.Header.Left, c.Header.Right, innerLeft, innerRight)
		}
	}
}

func TestAWrappedTeamNameKeepsTheHeadingsAligned(t *testing.T) {
	weekTree := weekFS(2025, 15, map[string]string{
		"bojjaes":                     lineupCSV(ourLine),
		"extraordinarilylongteamname": lineupCSV(theirLine),
	})
	rec := serve(Handler(lineup.New(weekTree), &fakeSource{weekStats: fixtureStats()}), http.MethodGet, "/2025/15")

	l := renderedLayout(t, rec.Body.String()+narrowCardCSS)

	if len(l.Columns) != 2 {
		t.Fatalf("measured %d columns, want 2", len(l.Columns))
	}
	ours, theirs := l.Columns[0], l.Columns[1]
	if theirs.H2.Height <= ours.H2.Height {
		t.Errorf("h2 heights = %v and %v, want the long name's taller", ours.H2.Height, theirs.H2.Height)
	}
	if ours.Header.Top != theirs.Header.Top || ours.Header.Height != theirs.Header.Height {
		t.Errorf("headers = top %v height %v and top %v height %v, want equal", ours.Header.Top, ours.Header.Height, theirs.Header.Top, theirs.Header.Height)
	}
	if ours.Total.Top != theirs.Total.Top {
		t.Errorf("total tops = %v and %v, want equal", ours.Total.Top, theirs.Total.Top)
	}
	for i, c := range l.Columns {
		if c.Total.Right > c.Column.Right {
			t.Errorf("column %d total right %v is outside the column right %v", i, c.Total.Right, c.Column.Right)
		}
	}
	if ours.FirstLi.Top != theirs.FirstLi.Top {
		t.Errorf("first li tops = %v and %v, want equal", ours.FirstLi.Top, theirs.FirstLi.Top)
	}
}
