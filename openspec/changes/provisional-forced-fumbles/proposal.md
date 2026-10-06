## Why

When the play store is cold or has a gap, forced fumbles are counted from a 300-play poll while the
whole-week fetch runs in the background. The fetch lands in 3–12 s, but the partial counts are
cached for the 5-minute stats TTL and the page reloads only every 5 minutes, so readers see
underscored defenders for up to 5 minutes (up to ~10 for a reader who loads a cached result just
before it expires). Under scale-to-zero this happened on most game-day visits. With the machine kept
running it still happens after every deploy and on the first read after a lull of more than 300
plays.

## What Changes

- The play store reports a week as **provisional** while a whole-week fetch is filling a known hole
  (an empty store or a gap) and has not yet succeeded.
- Weekly stats built from a provisional forced-fumble count carry that flag. The stats cache holds a
  provisional entry for a short TTL (about 15 s) instead of 5 minutes.
- A provisional matchup page reloads itself after about 15 s instead of 5 minutes and says that
  forced fumbles are still loading. The refresh script still contains no interpolated page data.
- A week with a known hole stays marked incomplete until a whole-week fetch succeeds. A read while it
  is incomplete and no fetch is running starts a new one, which closes today's hole where a failed
  cold fetch left the week partial until the post-game refresh. A failed fetch drops the provisional
  flag, so retries happen at the normal TTL rather than every 15 s.
- A successful poll that returns fewer than the poll limit **is** the whole week. It counts as a
  successful whole-week fetch, and no 5000-play fetch is started. Future and early weeks stop
  triggering a whole-week fetch on every read.
- `fly.toml` sets `min_machines_running = 1`, landing the always-on machine that ADR 0005 already
  plans. Suspend was considered and rejected: cache expiry is measured on Go's monotonic clock, which
  may not advance while a VM is suspended, so pre-suspend entries could be served as fresh.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `forced-fumble-attribution`: a short poll counts as a whole-week read. A hole stays open until a
  whole-week fetch succeeds. The store reports whether a week's count is provisional. (This
  capability is still in the unarchived `score-forced-fumbles` change, so that change is archived
  first.)
- `weekly-stats-cache`: a provisional week is cached for a short TTL rather than five minutes.
- `matchup-page`: a provisional page refreshes on a short interval and states that forced fumbles are
  still loading.

## Impact

- `internal/sleeper/playstore.go`: tracks incomplete weeks, applies the short-poll rule, and returns
  provisional status from `ForcedFumbles`.
- `internal/sleeper/sleeper.go`: `Client.WeekStats` passes the flag into the weekly stats.
- `internal/score/weekstats.go`: `WeekStats` gains a provisional flag.
- `internal/statscache/cache.go`: TTL per entry.
- `internal/web/matchup.go` and `matchup.html`: provisional marker, refresh interval, and as-of note.
- `fly.toml`; the `docs/architecture.md` deployment table.
- Upstream volume: a cold or gap read now costs at most a couple of extra aggregate fetches while the
  whole-week fetch runs. The machine costs about $0.45 a week to keep running.
