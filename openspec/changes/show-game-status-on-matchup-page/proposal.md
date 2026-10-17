## Why

The matchup page shows a starter's points and nothing about their game. A dash can mean the game has
not kicked off, the player is on bye, or the player did not record a stat, and a number can be final
or still moving. The reader has to check somewhere else to know which.

Sleeper publishes each week's games with status, kickoff time, and quarter, so the page can say it.

## What Changes

- The server fetches the week's games from Sleeper: `GET api.sleeper.com/scores/nfl/regular/{season}/{week}`,
  one unauthenticated call returning every game with its two teams, `status`
  (`pre_game` / `in_game` / `complete`), `start_time`, and `metadata.quarter`.
- Each starter is matched to a game by the **NFL team on their lineup record**. A team with no game
  that week is on bye. A lineup team of `FA` has no game and is shown as neither.
- Each starter row shows one game state:

  | State | Shown |
  |---|---|
  | Upcoming | weekday and kickoff time |
  | In progress | Sleeper's quarter text: `Q` and the number for a numbered quarter, `HALF` as sent, `Live` if Sleeper sends none; no game clock |
  | Complete | `Final`, or `Final/OT` where Sleeper reports overtime |
  | Bye | Bye |

  Sleeper's wording is trusted apart from spelling out `F` as `Final`. Text outside the values seen
  so far is shown as sent, not mapped.

- Times are rendered by the server in `America/Chicago` inside a `<time>` element carrying the exact
  instant, as the as-of line is today. The page script rewrites every `<time>` into the **browser's
  timezone**; where the script does not run, Central stays. This applies to the existing as-of line
  too, so the page never mixes zones.
- The stats cache is **generalised over its value type** as a behaviour-preserving refactor, done
  first, and the week's games are cached through it with the same fixed TTL, single-flight, and
  failure rules the stats have today.
- A failed or missing game fetch **does not fail the page**: scores are served as now and the game
  state is left out. A week Sleeper returns no games for is treated the same way, so it does not show
  every starter on bye.
- Game fetches are logged like the other upstream calls.

Not in this change:

- Any change to how often stats are fetched. That is `slow-stats-polling-when-idle`, which builds on
  the generic cache and the game feed introduced here.
- The game clock, scores of the NFL games, opponent names, or a "live" indicator beyond the quarter.
- Lineup lock at kickoff.
- Putting the game fetch into the stats version token. A status change reaches an open page on its
  next reload, at most about five minutes after the server learns of it.
- Overriding the lineup team with the team or game on a player's stats row. The lineup file is the
  single source for the join.
- Correcting a stale lineup team. A player traded after the file was written is matched to their old
  team's game until the file is fixed.

## Capabilities

### New Capabilities

- `game-schedule`: fetching a week's NFL games from the provider, and answering for a team in that
  week whether its game is upcoming (and when), in progress (and which quarter), complete, or absent.

### Modified Capabilities

- `matchup-page`: each starter row shows its game state; instants are shown in the browser's
  timezone with `America/Chicago` as the unscripted fallback, the as-of line included; the page is
  still served when the game feed is unavailable.
- `lineup-source`: a record's team stops being display text only. It is used to find the player's
  game, while the id stays the only thing that identifies the player.
- `weekly-stats-cache`: the cache's guarantees hold for any weekly fetch, not only stats, and the
  week's games are held under them.
- `upstream-call-logging`: game fetches are logged.

## Impact

- `internal/sleeper`: a new fetch for the scores endpoint, on the host and client already in use.
- `internal/statscache`: renamed `internal/weekcache`; `Cache` becomes generic and its read method
  is renamed `AsOf`. `web`'s source interface takes the same name, so no adapter is needed.
- `internal/games`: new provider-neutral package for a week's games, as `score` is for stat lines.
- `internal/lineup`: the comment and spec wording on `Team`; no parsing change. The closed set of
  team codes already matches Sleeper's (`JAX`, `LAR`, `LV`, `WAS`).
- `internal/web`: new per-starter field and markup in `matchup.html`; the script gains the timezone
  rewrite, which reads `<time datetime>` from the DOM and so keeps the script free of page data.
- Upstream volume: one more call per week per TTL, about 12 an hour while a page is being read.
- Risk: the endpoint is undocumented, like the stats and plays endpoints already relied on. Observed
  on 2026-10-11 only; statuses for postponed, delayed, or cancelled games have not been seen.
