package web

import (
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/eshifrin/bojjaes/internal/lineup"
)

func servedScript(t *testing.T) string {
	t.Helper()

	source := &fakeSource{weekStats: fixtureStats(), fetchedAt: asOfInstant}
	rec := serve(Handler(lineup.New(fixtureWeek()), source, noGames{}), http.MethodGet, "/2025/15")
	return renderedScript(t, rec.Body.String())
}

// The page's values reach the script through the DOM and the address bar, so
// the script's text stays the same for every week.
func TestTheCheckReadsItsInputsFromThePage(t *testing.T) {
	script := servedScript(t)

	for _, want := range []string{"dataset.version", "dataset.nextCheck", `location.pathname + "/version"`} {
		if !strings.Contains(script, want) {
			t.Errorf("script does not contain %s:\n%s", want, script)
		}
	}
}

// checkScript is the part of the script after the check URL is built: the
// version check itself, apart from the five-minute timer above it.
func checkScript(t *testing.T) string {
	t.Helper()

	_, check, found := strings.Cut(servedScript(t), `"/version"`)
	if !found {
		t.Fatal("script builds no version URL")
	}
	return check
}

// The check has its own guard: the timer's guard above it does not keep a
// hidden tab from asking.
func TestTheCheckIsGuardedByTabVisibility(t *testing.T) {
	check := checkScript(t)

	if !strings.Contains(check, "fetch(checkUrl") {
		t.Fatalf("the check makes no request to the version URL:\n%s", check)
	}
	guard, _, _ := strings.Cut(check, "fetch(checkUrl")
	if !strings.Contains(guard, `document.visibilityState === "visible"`) {
		t.Errorf("no visibility guard stands before the check's request:\n%s", check)
	}
}

var versionParsePattern = regexp.MustCompile(`(?i)(Number|parseInt|parseFloat|BigInt|Date)\([^)]*version`)

// The token is opaque: any change in it, including to empty, is a reload, and
// nothing in the script may read meaning into it.
func TestTheCheckReloadsWhenTheVersionsAreNotEqual(t *testing.T) {
	check := checkScript(t)

	if !strings.Contains(check, ".version !== version") {
		t.Fatalf("the check does not compare the reported version with the page's by strict inequality:\n%s", check)
	}
	_, onMismatch, _ := strings.Cut(check, ".version !== version")
	if !strings.Contains(onMismatch, "location.reload()") {
		t.Errorf("a mismatch does not reload the page:\n%s", check)
	}
	if m := versionParsePattern.FindString(servedScript(t)); m != "" {
		t.Errorf("the script parses the version (%q); it may only compare it", m)
	}
}

// A chained timeout, not an interval: each wait is the one the server last
// gave, and a zero ends the chain.
func TestTheNextCheckIsScheduledFromTheResponse(t *testing.T) {
	check := checkScript(t)

	for _, want := range []string{"setTimeout", "scheduleCheck(held.nextCheckSeconds)", "scheduleCheck(firstDelay)"} {
		if !strings.Contains(check, want) {
			t.Errorf("the check does not contain %s:\n%s", want, check)
		}
	}
	if strings.Contains(check, "setInterval") {
		t.Errorf("the check repeats on a fixed interval, which cannot follow a changed delay:\n%s", check)
	}

	_, scheduler, found := strings.Cut(check, "function scheduleCheck(")
	if !found {
		t.Fatalf("the check has no scheduleCheck function:\n%s", check)
	}
	guard, _, _ := strings.Cut(scheduler, "setTimeout")
	if !strings.Contains(guard, "> 0") {
		t.Errorf("a zero delay is not kept from scheduling a check:\n%s", scheduler)
	}
}

// A hidden tab asks nothing, but must not fall out of the chain: it waits the
// same delay again, so a returning reader is checked within one delay.
func TestAHiddenTabWaitsAgainWithoutAsking(t *testing.T) {
	check := checkScript(t)

	_, hidden, found := strings.Cut(check, "} else {")
	if !found {
		t.Fatalf("the check's visibility guard has no hidden branch:\n%s", check)
	}
	hidden, _, _ = strings.Cut(hidden, "}")
	if !strings.Contains(hidden, "scheduleCheck(delay)") {
		t.Errorf("a hidden tab does not wait the same delay again: %q", hidden)
	}
	if strings.Contains(hidden, "fetch(") {
		t.Errorf("a hidden tab makes a request: %q", hidden)
	}
}

// A failure says nothing about the stats, so it must never reload: a server
// that is briefly down, or an older build with no such route, leaves the page
// as it is and the check tries again at the delay it last used.
func TestAFailedCheckDoesNotReload(t *testing.T) {
	check := checkScript(t)

	_, onFailure, found := strings.Cut(check, ".catch(")
	if !found {
		t.Fatalf("the check handles no failure:\n%s", check)
	}
	onFailure, _, _ = strings.Cut(onFailure, "});")
	if !strings.Contains(onFailure, "scheduleCheck(delay)") {
		t.Errorf("a failed check does not try again at the delay it last used: %q", onFailure)
	}
	if strings.Contains(onFailure, "location.reload") {
		t.Errorf("a failed check reloads the page: %q", onFailure)
	}

	// An error status and a body of the wrong shape are failures too; without
	// these an undefined version would compare unequal and reload.
	for _, want := range []string{"response.ok", `typeof held.version !== "string"`, `typeof held.nextCheckSeconds !== "number"`} {
		if !strings.Contains(check, want) {
			t.Errorf("the check does not treat an unreadable response as a failure (no %s):\n%s", want, check)
		}
	}
}
