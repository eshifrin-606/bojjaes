## Context

`internal/web` renders `/{season}/{week}` from a single `matchup.html` template with `html/template`
and no client-side code at all. Every request costs one `StatsSource.WeekStats` call, and there is
no cache in front of it yet.

[ADR 0004](../../../docs/adr/0004-web-frontend-stack.md) decision #1 committed to "no JavaScript
framework, no build step, no bundler — the small amount of client-side behaviour it needs (a refresh
timer) is a few lines of vanilla JS," and decision #5 committed to client-polls/server-caches with
"roughly every 5 minutes, and only while the tab is visible." This change is that timer and nothing
else. The server cache it is designed to sit in front of does not exist yet.

The design constraint that outranks everything here is ADR 0004 decision #8: the page must not imply
a winner. A refresh is a new place that constraint can be violated — by diffing against what was
previously on screen — so the spec binds the script as well as the markup.

## Goals / Non-Goals

**Goals:**

- The page re-fetches roughly every five minutes while its tab is visible.
- A hidden tab costs nothing upstream.
- Returning to a tab that has gone stale shows current scores without the reader reaching for
  reload.
- The whole thing is small enough to read at a glance inside the template.

**Non-Goals:**

- The as-of timestamp. It is the other half of the freshness story and its own backlog line; this
  change neither adds it nor depends on it.
- The TTL/single-flight cache. This change increases upstream request volume and deliberately does
  not fix that.
- Partial updates, fetch-and-swap, SSE, WebSockets, or any push.
- A configurable interval, a user-visible refresh control, or a countdown.
- Any diff, animation, or highlight of what changed between loads — explicitly forbidden, not
  merely omitted.

## Decisions

### Reload the page, do not fetch and swap

`location.reload()` re-runs the whole server-rendered path we already have and trust. The
alternative — `fetch` the same URL, parse it, and replace `<main>` — would need the page to have an
opinion about its own DOM, and it buys nothing on a document that is two short lists.

The one thing swapping would buy is the ability to keep showing the last good scores when a refresh
fails. That is a real cost of this decision and it is recorded under Risks rather than designed
around; if it bites, swapping is a contained follow-up that changes no server code.

Rejected alongside it: `<meta http-equiv="refresh" content="300">`. It is fewer lines than any
script, but it cannot be conditioned on visibility, which is half the requirement. A hidden tab
would keep hitting Sleeper all week.

### One interval with a visibility guard, plus a visibility handler for the stale case

Two small pieces:

- A `setInterval` at the refresh interval whose body reloads only when
  `document.visibilityState === "visible"`.
- A `visibilitychange` listener that reloads immediately when the tab becomes visible and the
  elapsed time since load is already past the interval.

The alternative was to clear the interval on hide and restart it on show. It produces the same
observable behaviour, but it has to reason about how much of the current period had elapsed when
the tab was hidden — which is exactly the bookkeeping the guard-plus-elapsed-check avoids. Checking
the condition at fire time and checking staleness on return is less state.

Browsers heavily throttle timers in background tabs, which is a help rather than a problem: the
throttled ticks find `visibilityState === "hidden"` and do nothing. The design does not depend on
the throttling, only on the guard.

"Elapsed since load" comes from a `Date.now()` captured when the script first runs, not from a
timestamp rendered by the server — the script interpolates nothing, per the spec.

### The interval is a literal in the template

`5 * 60 * 1000`, written in the script. Not a Go constant piped into the template, not a config
value, not a data attribute. There is one caller, the value is fixed by an ADR, and threading it
through the view model would put a number in `matchup.go` whose only purpose is to come back out in
`matchup.html`.

If a test wants to pin the interval, it pins the rendered text. That is a weaker assertion than
pinning a Go constant, and it is the right weight for a value that is only ever going to be read by
a browser.

### The script goes at the end of the document, inline, in `matchup.html`

No separate `.js` file and therefore no second `//go:embed` entry, no route to serve it, and no
extra request. When the embed-everything backlog line lands it will not change this decision — the
script is small enough that a separate file would cost a round trip to save nothing.

Inline script inside `html/template` puts the content in a JavaScript context, where the template
package escapes differently than it does in HTML text. Since nothing is interpolated into it, that
distinction never comes up — which is precisely why the spec forbids interpolation rather than
trusting the escaping to be right.

### What the tests can and cannot pin

Go tests can assert what the response body contains: that a script is present, that it references
`visibilityState`, that it carries the interval, and that no roster or team name appears inside the
script element. They cannot assert that a timer fires, that a hidden tab stays quiet, or that a
returning reader gets a fresh page — no browser runs in `go test`.

So the loop is: red-green on the rendered contract, then hand verification against a running server
for the three behaviours that only exist in a browser. The tasks name those explicitly rather than
leaving them to "and then check it works." This is the same split the project already accepts for
`scripts/*.sh` as an interim UI, applied here to the one part of the page that runs somewhere Go
cannot reach.

## Risks / Trade-offs

**Every refresh is a Sleeper fetch until the TTL cache lands** → Three open tabs become three
upstream calls every five minutes, where today an idle tab makes none. At three viewers that is
about one call a minute at peak, against an API whose limits are unprobed (ADR 0003). Accepted as
the smaller half of a two-part change: the cache is the next backlog line, and the visibility guard
already keeps this from scaling with tabs left open overnight. If we deploy publicly before the
cache exists, the cache goes first.

**A refresh during a Sleeper outage replaces a good page with a 502** → The handler refuses to serve
a zeroed page, correctly, so a failed fetch is an error page. Under reload-and-not-swap, that error
page replaces scores the reader was looking at a moment ago, and the reader has no way back except
their own reload. This is the sharpest edge in the change. It is not mitigated here: the TTL cache
makes it much rarer by absorbing brief upstream failures, and fetch-and-swap would remove it
entirely at the cost of the complexity rejected above. Worth revisiting after one live Sunday.

**Scores can move down between refreshes** → Sleeper revises stats (ADR 0003), so a total can drop
across a reload with nothing on the page explaining why. Pre-existing — it happens on a manual
reload today — but refreshing makes it something the reader watches happen rather than something
they cause. The as-of timestamp does not fix it either. Noted, not addressed.

**Refreshing without an as-of timestamp is freshness the reader cannot verify** → After this change
the page is quietly current; if the timer breaks, or a tab is restored from bfcache in a state
where the script never re-runs, the page is quietly stale and looks identical. The timestamp is the
answer and it is not in this change. This argues for taking that backlog line next, and is worth
saying out loud rather than discovering on a Sunday.

**Reload discards client state** → There is none: no scroll position worth preserving on a
two-column page, no form, no expanded sections. Recorded so that the first feature that adds client
state knows it has to revisit this.

**Three viewers reloading in the same second** → Real, and irrelevant at this scale; no jitter is
added. Single-flight in the coming cache is the proper answer, and adding randomised jitter now
would be complexity in the wrong layer.
