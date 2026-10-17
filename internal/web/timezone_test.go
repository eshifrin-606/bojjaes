package web

import (
	"context"
	"html"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The instant comes from the element the server rendered, so the script
// carries no page data of its own.
func TestTheRewriteReadsEachInstantFromItsTimeElement(t *testing.T) {
	script := servedScript(t)

	for _, want := range []string{`querySelectorAll("time[datetime]")`, `getAttribute("datetime")`} {
		if !strings.Contains(script, want) {
			t.Errorf("script does not contain %s:\n%s", want, script)
		}
	}
}

// Naming no timezone is what makes the browser use the reader's own; the
// locale is fixed so the text keeps the server's shape everywhere.
func TestTheRewriteFormatsInTheBrowsersTimezone(t *testing.T) {
	script := servedScript(t)

	for _, want := range []string{`Intl.DateTimeFormat("en-US"`, "formatToParts"} {
		if !strings.Contains(script, want) {
			t.Errorf("script does not contain %s:\n%s", want, script)
		}
	}
	if strings.Contains(script, "timeZone:") {
		t.Errorf("script names a timezone instead of using the browser's:\n%s", script)
	}
}

// Each element gets its own try, so one instant the browser cannot format
// leaves that element's Central text and the rest still rewritten.
func TestEachRewriteIsInsideItsOwnTry(t *testing.T) {
	script := servedScript(t)

	_, perElement, found := strings.Cut(script, `querySelectorAll("time[datetime]")`)
	if !found {
		t.Fatalf("script selects no time elements:\n%s", script)
	}
	guarded, _, found := strings.Cut(perElement, "} catch (")
	if !found || !strings.Contains(guarded, "try {") || !strings.Contains(guarded, "formatToParts") {
		t.Errorf("the per-element rewrite is not inside a try:\n%s", perElement)
	}
}

// domInZone is the DOM headless Chrome builds from body with the page script
// run under the given TZ, which is how the browser learns the reader's zone.
func domInZone(t *testing.T, tz, body string) string {
	t.Helper()
	if testing.Short() {
		t.Skip("running the timezone rewrite needs Chrome; skipped under -short")
	}
	chrome := chromePath()
	if chrome == "" {
		t.Skip("no Chrome: set BOJJAES_CHROME or install Google Chrome")
	}

	file := filepath.Join(t.TempDir(), "page.html")
	if err := os.WriteFile(file, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, chrome, "--headless", "--disable-gpu", "--dump-dom", "file://"+file)
	cmd.Env = append(os.Environ(), "TZ="+tz)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("running Chrome: %v", err)
	}
	return string(out)
}

// timeText is the text of the time element stamped with instant.
func timeText(t *testing.T, dom string, instant time.Time) string {
	t.Helper()
	stamp := instant.Format(time.RFC3339)
	for _, tm := range renderedTimes(dom) {
		if tm[0] == stamp {
			return html.UnescapeString(tm[1])
		}
	}
	t.Fatalf("no time element for %s in:\n%s", stamp, dom)
	return ""
}

func servedInZone(t *testing.T, tz string, kickoff, fetchedAt time.Time) string {
	t.Helper()
	source := &fakeSource{weekStats: fixtureStats(), fetchedAt: fetchedAt}
	body := serve(Handler(chaseTree(), source, upcomingCIN(kickoff)), http.MethodGet, "/2025/15").Body.String()
	return domInZone(t, tz, body)
}

func TestARewriteInNewYork(t *testing.T) {
	kickoff := time.Date(2026, 10, 11, 17, 0, 0, 0, time.UTC)
	fetchedAt := time.Date(2026, 10, 11, 22, 7, 0, 0, time.UTC)

	dom := servedInZone(t, "America/New_York", kickoff, fetchedAt)

	if got, want := timeText(t, dom, kickoff), "Sun 1:00 PM"; got != want {
		t.Errorf("kickoff reads %q, want %q", got, want)
	}
	if got, want := timeText(t, dom, fetchedAt), "Sun, Oct 11 2026 6:07 PM EDT"; got != want {
		t.Errorf("as-of reads %q, want %q", got, want)
	}
}

// Monday evening in Chicago is already Tuesday in London: the weekday has to
// come from the same zone as the time.
func TestARewriteInLondonMovesTheWeekday(t *testing.T) {
	kickoff := time.Date(2026, 10, 13, 0, 15, 0, 0, time.UTC)

	dom := servedInZone(t, "Europe/London", kickoff, asOfInstant)

	if got, want := timeText(t, dom, kickoff), "Tue 1:15 AM"; got != want {
		t.Errorf("kickoff reads %q, want %q", got, want)
	}
}

func TestARewriteInChicagoKeepsTheServedClock(t *testing.T) {
	for _, kickoff := range []time.Time{
		time.Date(2026, 10, 11, 17, 0, 0, 0, time.UTC),
		time.Date(2026, 10, 13, 0, 15, 0, 0, time.UTC),
	} {
		t.Run(kickoff.Format(time.RFC3339), func(t *testing.T) {
			want := kickoff.In(chicagoLoc).Format(kickoffLayout)

			dom := servedInZone(t, "America/Chicago", kickoff, asOfInstant)

			if got := timeText(t, dom, kickoff); got != want {
				t.Errorf("kickoff reads %q, want the served %q", got, want)
			}
		})
	}
}
