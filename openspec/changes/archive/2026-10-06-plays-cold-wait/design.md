## Context

`PlayStore.ForcedFumbles` polls 300 plays synchronously and starts the whole-week fetch in a
detached goroutine (`startWholeFetch`). `statscache` then holds the resulting `WeekStats` for 5
minutes. On a cold store, the first read is scored from the poll alone, and the whole-week result is
not served until the cache entry expires.

`score-forced-fumbles` D4 rejected waiting on the whole-week fetch because it costs 3–12 s on the
request path. This change brings the wait back, but only for a cold week, only up to a bound, and
only where that bound is nonzero. That keeps D4's concern on Fly (`0`) while fixing local debugging
(`30s`).

## Goals / Non-Goals

**Goals:**
- A cold week's first read can include whole-week forced fumbles.
- The wait is set per environment: local and Fly differ without a code change.
- A bad value fails loudly at startup.

**Non-Goals:**
- Invalidating `statscache` when a background fetch lands. That is the follow-up (option C).
- Waiting on gap fills or post-game refreshes.
- Any page-visible signal that forced fumbles may be incomplete.

## Decisions

### D1. Wait on a done channel owned by the in-flight fetch

`wholeInFlight` changes from `map[weekKey]bool` to `map[weekKey]chan struct{}`. `startWholeFetch`
returns the channel of the fetch it started, or of the one already running, and the channel is
closed when the fetch finishes (success or failure). `ForcedFumbles` records `cold` (store empty for
the week) before merging the poll. When `cold` and `coldWait > 0`, it selects on the done channel,
a `coldWait` timer, and `ctx.Done()`, then takes the snapshot.

Single-flight, the detached context, and the 30 s fetch budget are unchanged: the waiter only
observes the fetch and never owns it. A timed-out waiter leaves the fetch running.

*Alternative:* run the whole-week fetch synchronously when cold. Rejected: a timeout would cancel
the fetch, the next read would start over, and a second concurrent cold read would not share it.

### D2. The wait is a `NewPlayStore` argument

`NewPlayStore(baseURL, coldWait, logf)`. Existing tests pass `0` and keep today's behavior. A
constructor argument is clearer than a field set after construction, and unlike `statscache`'s
options (which are test seams), this is real configuration.

### D3. Resolve the env var in `cmd/server`, beside `PORT` and `LINEUP_VOLUME`

`resolvePlaysColdWait(getenv func(string) string) (time.Duration, error)` follows the
`resolveAddr` / `resolveLineupVolume` pattern: it takes a fake `getenv` and is a pure function.
Empty means `30s`. Otherwise it uses `time.ParseDuration` and rejects negatives. `main` exits with
`log.Fatalf` on error, before listening.

The default is the local value because local runs set no environment. Fly sets
`PLAYS_COLD_WAIT = '0'` in `fly.toml` `[env]`, which is checked in and reviewable, unlike a secret.

*Alternative:* default `0` and opt in locally. Rejected by choice: local is where the wait matters,
and a plain `go run` should get it.

## Risks / Trade-offs

- **Worst case locally runs into `writeTimeout` (30 s).** The wait honors the ctx that `statscache`
  passes in (30 s from the start of the fetch), and the aggregate fetch and poll come first, so a
  hung whole-week fetch can push the handler to about 30 s and the response write can be cut off.
  → Local only, and only when Sleeper is hung. Lower `PLAYS_COLD_WAIT` locally if it bites.
- **Fly cold starts still show partial forced fumbles for up to 5 min.** → Unchanged from today.
  Option C or a nonzero Fly value addresses it later.
- **A test that waits on a real timer is slow or flaky.** → Tests drive completion through the fake
  server and use short waits. The "wait runs out" case blocks the fake whole-week response and uses
  a wait of a few milliseconds.

## Migration Plan

Archive `score-forced-fumbles` first, so this delta has a main spec to modify. Deploy as usual. Fly
gets `0` from `fly.toml`, so behavior there is unchanged. To roll back, revert, or set the value
without a code change.
