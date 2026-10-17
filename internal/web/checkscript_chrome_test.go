package web

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/eshifrin/bojjaes/internal/lineup"
)

// reloadedMarker is added to every render after the first, so the DOM Chrome
// dumps says whether the page reloaded itself.
const reloadedMarker = `<p id="reloaded">second render</p>`

type checkRun struct {
	dom       string
	pageLoads int32
	checks    int32
}

// runCheckScript serves the real page to headless Chrome with the given first
// delay, answers its version checks with answer, and lets virtual time run
// for budget. Every render after the first carries reloadedMarker and a delay
// of zero, so a reload happens at most once.
func runCheckScript(t *testing.T, firstDelay int, budget time.Duration, answer http.HandlerFunc) checkRun {
	t.Helper()
	if testing.Short() {
		t.Skip("running the check script needs Chrome; skipped under -short")
	}
	chrome := chromePath()
	if chrome == "" {
		t.Skip("no Chrome: set BOJJAES_CHROME or install Google Chrome")
	}

	source := &fakeSource{weekStats: fixtureStats(), fetchedAt: asOfInstant}
	body := serve(Handler(lineup.New(fixtureWeek()), source, noGames{}), http.MethodGet, "/2025/15").Body.String()
	served := fmt.Sprintf(`data-next-check="%d"`, checkDelaySeconds)
	if !strings.Contains(body, served) {
		t.Fatalf("the page does not carry %s to replace", served)
	}
	withDelay := func(seconds int) string {
		return strings.Replace(body, served, fmt.Sprintf(`data-next-check="%d"`, seconds), 1)
	}

	var run checkRun
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{season}/{week}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if atomic.AddInt32(&run.pageLoads, 1) == 1 {
			w.Write([]byte(withDelay(firstDelay)))
			return
		}
		w.Write([]byte(withDelay(0) + reloadedMarker))
	})
	mux.HandleFunc("GET /{season}/{week}/version", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&run.checks, 1)
		answer(w, r)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, chrome, "--headless", "--disable-gpu", "--dump-dom",
		fmt.Sprintf("--virtual-time-budget=%d", budget.Milliseconds()), srv.URL+"/2025/15").Output()
	if err != nil {
		t.Fatalf("running Chrome: %v", err)
	}
	run.dom = string(out)
	return run
}

func answerVersion(version string, nextCheckSeconds int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"version":%q,"nextCheckSeconds":%d}`, version, nextCheckSeconds)
	}
}

// The script is otherwise tested by reading its text. These run it: each case
// is one answer from the version endpoint and what the page then does over
// three and a half virtual seconds at a one-second delay.
func TestTheCheckScriptInChrome(t *testing.T) {
	held := versionToken(asOfInstant)
	const budget = 3500 * time.Millisecond

	tests := []struct {
		name       string
		firstDelay int
		answer     http.HandlerFunc
		wantReload bool
		// wantChecks is exact when the chain should have stopped, and a
		// minimum when it should have kept going.
		wantChecks int32
		keepsGoing bool
	}{
		{name: "a later version reloads", firstDelay: 1, answer: answerVersion("a-later-fetch", 1), wantReload: true, wantChecks: 1},
		{name: "an empty version reloads", firstDelay: 1, answer: answerVersion("", 1), wantReload: true, wantChecks: 1},
		{name: "the same version keeps checking", firstDelay: 1, answer: answerVersion(held, 1), wantChecks: 2, keepsGoing: true},
		{name: "a zero delay in a response stops the checks", firstDelay: 1, answer: answerVersion(held, 0), wantChecks: 1},
		{name: "a zero first delay never checks", firstDelay: 0, answer: answerVersion("a-later-fetch", 1), wantChecks: 0},
		{name: "a longer delay in a response is followed", firstDelay: 1, answer: answerVersion(held, 60), wantChecks: 1},
		// A well-formed body under an error status: only the status says not
		// to believe it.
		{name: "an error status does not reload and tries again", firstDelay: 1, answer: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			answerVersion("a-later-fetch", 1)(w, r)
		}, wantChecks: 2, keepsGoing: true},
		{name: "a missing route does not reload and tries again", firstDelay: 1, answer: http.NotFound, wantChecks: 2, keepsGoing: true},
		{name: "an unknown shape does not reload and tries again", firstDelay: 1, answer: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{}`))
		}, wantChecks: 2, keepsGoing: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := runCheckScript(t, tt.firstDelay, budget, tt.answer)

			if got := strings.Contains(run.dom, reloadedMarker); got != tt.wantReload {
				t.Errorf("reloaded = %t, want %t (%d page loads, %d checks)", got, tt.wantReload, run.pageLoads, run.checks)
			}
			if tt.keepsGoing && run.checks < tt.wantChecks {
				t.Errorf("made %d checks, want at least %d", run.checks, tt.wantChecks)
			}
			if !tt.keepsGoing && run.checks != tt.wantChecks {
				t.Errorf("made %d checks, want exactly %d", run.checks, tt.wantChecks)
			}
		})
	}
}
