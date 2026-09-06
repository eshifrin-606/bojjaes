## Context

`internal/score` holds four concerns whose reasons to change have nothing to do with each other:

| Concern | Files | Changes when |
| --- | --- | --- |
| Domain stat line | `stats.go` | the league scores a new category |
| Scoring rules | `calc.go` | `docs/scoring.md` changes |
| Sleeper adapter | `sleeper.go`, `sleeper_test.go` (645 lines) | Sleeper changes a key, or a provider is added |
| HTTP/JSON transport | `handler.go`, `batch.go` | the wire format changes |

The `player-week-score` spec already states the boundary — "Vendor stat keys (`rec_yd`,
`rush_td_40p`, …) SHALL NOT appear outside the Sleeper transform" — and
[ADR 0002](../../../docs/adr/0002-live-scoreboard-backend.md) decision #1 states it as architecture.
Neither is enforced: one package means one namespace, so the rule survives on discipline.

The forcing function is the matchup page. `internal/web` needs to score two rosters from one fetch.
With today's shape its only options are to call `POST /scores` over HTTP from inside the same
process, or to receive `map[string]map[string]float64` — Sleeper's decoded payload — as a parameter.

## Goals / Non-Goals

**Goals:**

- Package boundaries that make the vendor rule a compile error rather than a convention.
- A way to score a set of players from one fetch that does not involve HTTP and does not expose the
  provider's payload shape.
- Behaviour preserved exactly for `POST /scores` and therefore for `scripts/**`.
- Leave the TTL cache — the next backlog item — as something wired in `main`, not edited into a
  consumer.

**Non-Goals:**

- Any change to a scoring rule, a stat mapping, or the `POST /scores` wire format.
- The cache itself, a second provider, or the matchup page.
- Splitting `StatLine` from `Points`. They are one concept: a new scoring category adds a field and
  a term together, always.

## Decisions

**Three packages, split along the file boundaries that already exist.**

```
internal/score    StatLine, Points, Week      — pure; no I/O, no HTTP, no vendor keys
internal/sleeper  FetchWeek → score.Week      — the only package where a vendor key appears
internal/api      POST /scores + DTOs         — the JSON layer
cmd/server        composition root            — the only package that names internal/sleeper
```

`score` keeps its name and import path: it is the package that shrinks rather than moves, so most of
the diff is deletion. `sleeper` and `api` are new homes for files that move with their tests.

**`score.Week` is the seam, and it lives in the neutral package.** If the weekly snapshot type
belonged to `sleeper`, every consumer would import `sleeper` to name it and the boundary would be
decorative. So `score` owns the type and `sleeper` builds one:

```go
// package score
type Week struct {
    season, week int
    players      map[string]StatLine
}
func (w Week) Player(id string) (StatLine, bool)

// package sleeper
func FetchWeek(ctx context.Context, baseURL string, season, week int) (score.Week, error)
```

**`FetchWeek` transforms every player in the payload, not only the ones asked for.** This is the one
behavioural shape change in an otherwise mechanical split, and it is what lets the raw map's
lifetime end inside `FetchWeek`.

The alternative — a lazy `Week` wrapping a lookup closure over the raw payload — transforms only the
~18 players a request wants. Both keep the vendor shape out of `score`; the trade is when the
mapping happens. Eager was chosen because:

- The cost is noise. Transforming every player is ~25 map reads each across a few thousand entries,
  against a request that already pays a 200 ms – 1.2 s fetch and a JSON decode that built those same
  few thousand maps. The transform is a fraction of the decode that produced its input.
- A plain struct is constructible in a test. `web` and `api` tests can build a `Week` literally,
  with no `httptest` server and no Sleeper fixture. A closure-based `Week` can only be built by code
  that knows how to make the closure.
- A finished value cannot acquire a concurrency problem. `net/http` serves every request on its own
  goroutine, and the coming TTL cache is the program's first shared mutable state — one `Week` held
  in a field and handed to concurrent handlers. A map that is built once and never written is
  trivially safe to read concurrently. A lazy `Week` invites the obvious optimisation — memoise the
  transform — which writes to a shared map on what looks like a read path, and in Go a concurrent
  map read and write is an unrecoverable `fatal error`, not a subtle race.

If the transform ever does show up in a profile, moving to a lazy `Week` is a change to one struct
that no consumer can see. `BenchmarkTransform` and `BenchmarkDecode` in `internal/sleeper` record the
starting numbers, over a 3,000-player payload — a real week's size, since the fixture is a trimmed
11-player slice:

| | ns/op | B/op | allocs/op |
| --- | --- | --- | --- |
| Transform | 716,000 | 637,000 | 3,009 |
| JSON decode of the same payload | 13,009,000 | 9,273,000 | 193,670 |

(Apple M5, Go 1.26.) The eager transform costs about 0.7 ms — 5% of the 13 ms decode that produced
its input, and under half a percent of the 200 ms – 1.2 s fetch that precedes both. The decision
rests on that measurement rather than on the argument for it.

**Consumers declare the interface they need; `main` supplies the implementation.** Neither `api` nor
`web` imports `internal/sleeper`:

```go
// declared by the consumer, satisfied by a sleeper-backed value main constructs
type WeekSource interface {
    Week(ctx context.Context, season, week int) (score.Week, error)
}
```

This is ADR 0002's provider interface, finally with a compiler behind it. It also makes the TTL
cache additive: a caching `WeekSource` that wraps another one, wired in `main`, with no consumer
edited. And it removes `baseURL` from handler signatures, where it was a parameter the handler
carried but never used itself.

**`GET /score` is deleted rather than rehoused.** It scores one hardcoded player for one hardcoded
week and prints to stdout. Under the split it has no honest home: the fixed-player fixture is a test
of the adapter, so it becomes one — a `sleeper` test asserting the settled week 14 Nacua line — and
the endpoint, its `fetchTarget` helper, and the `NacuaPlayerID`/`TargetSeason`/`TargetWeek`
constants go with it. This also closes the backlog's "drop the walking-skeleton `fmt.Printf`" item.

**`POST /scores` moves unchanged and is labelled interim.** It exists so `scripts/scores.sh` and
`scripts/fantasycast.sh` run, those scripts are interim UI the page replaces, and the endpoint
retires with them. Saying so in a comment and in the README is cheap and keeps a future reader from
treating it as a designed public API.

**JSON tags stay on `StatLine`.** The wire format is `api`'s concern, so strictly the tags belong on
a DTO there — but that means a second 25-field struct and a copy function, to decouple two things
that have never moved independently and whose only consumer is our own shell scripts. Noted here so
the next person knows it was seen and declined, not missed.

**Test files move with their code, unedited where possible.** `sleeper_test.go` is 645 lines and
almost entirely a stat-key mapping table; it becomes `internal/sleeper`'s suite with its package
clause changed and its `testdata/week14.json` moved alongside. A test that has to change to compile
is a signal the split cut through something, and is worth a second look rather than a quick fix.

## Risks / Trade-offs

- **A four-package refactor with no behaviour change is a large diff that reviews poorly** → It is
  sequenced as its own change, ahead of the page, so the page's diff is new code rather than new
  code tangled with moves. The task list moves one concern at a time with a green suite between each.

- **`go test ./...` passing is weaker evidence than usual here, because the tests moved too** →
  `POST /scores` is pinned from outside the code as well: run `scripts/scores.sh` and
  `scripts/fantasycast.sh` against the rebuilt binary and diff their output against the same
  commands run before the split.

- **Eager transform is a real, if small, cost paid on every fetch** → Benchmarked in this change
  rather than asserted, and reversible behind `Week`'s method set if the number ever justifies it.

- **Deleting `GET /score` removes the one endpoint that was trivially curl-able** → The equivalent
  request is documented in the README alongside the endpoint that replaces it, and the settled-week
  assertion it embodied becomes a test that runs on every commit rather than a URL someone remembers
  to visit.

- **The split could be read as premature structure for a small program** → It is not speculative:
  the second consumer exists and is designed, and the interface it needs is the one ADR 0002 already
  specified. Doing it after the page lands means moving the page's code too.
