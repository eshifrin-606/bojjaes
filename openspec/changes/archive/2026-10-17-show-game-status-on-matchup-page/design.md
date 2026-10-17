## Context

The matchup page renders two lineups from one cached fetch of the week's stat lines. A page request
asks the cache for the week; a miss calls Sleeper, a hit does not. The entry lives five minutes, and
the page reloads itself every five minutes while visible.

Nothing on the page says where a player's NFL game stands. This change adds one more Sleeper call,
the week's games, holds it in a second cache entry, and shows each starter's game state.

What the feed looks like, from `GET api.sleeper.com/scores/nfl/regular/{season}/{week}` on
2026-10-11 and 12:

- A JSON array, one object per game. Top level: `game_id`, `status`, `start_time` (epoch
  milliseconds, a number). In `metadata`: `home_team`, `away_team`, `quarter` (a string, or null).
- `status` was `pre_game`, `in_game`, or `complete`. Halftime stays `in_game`.
- `metadata.quarter` was `""` or null before kickoff, `"3"` in a numbered quarter, `"HALF"`, `"F"`,
  and `"F/OT"`.
- A week with no games (2026 week 19) is `200` with `[]`. Past seasons and future weeks return
  their games.
- The 32 team codes are exactly the closed set `lineup-source` already accepts.
- Edge cache: `s-maxage=10` for the current week.

Constraints carried over from the existing design:

- `score` and `lineup` import neither `sleeper` nor `web`; `cmd/server` is the only package that
  names a provider.
- The page script holds no page data. Everything it reads comes from the DOM.
- The cache never fetches on its own; only a page request does.

## Goals / Non-Goals

**Goals:**

- Each starter row states its game: kickoff day and time, the quarter, final, or bye.
- Times appear in the reader's timezone, with Central as what the server writes.
- The games are cached with the same guarantees as the stats, through one cache implementation.
- A broken game feed costs the page its game states and nothing else.

**Non-Goals:**

- Changing how often stats are fetched (`slow-stats-polling-when-idle`).
- The game clock, NFL scores, opponents.
- Putting the game fetch into the stats version token.
- Overriding the lineup team from stats rows, or detecting a stale lineup team.
- Lineup lock.

## Decisions

### D1. The REST scores endpoint, not GraphQL

`/scores/nfl/regular/{season}/{week}` on the host and HTTP client `internal/sleeper` already uses.

*Alternatives:* the GraphQL `scores` query returns the same objects but would be the first GraphQL
call in the codebase. `/schedule/nfl/regular/{season}` is smaller but has no kickoff time and no
quarter.

### D2. A provider-neutral `internal/games` package holds the domain

`web` needs the games and `sleeper` produces them, and `web` must not import `sleeper`. So the
types live in a third package, as `score` does for stat lines.

```
games.Game   Home, Away string
             State   (Upcoming | InProgress | Complete | Other)
             Kickoff time.Time
             Period  string   // provider's quarter text, as sent
             Status  string   // provider's status text, as sent
games.Week   built from a season, a week and a slice of Game
             ForTeam(team) (Game, bool)
             Empty() bool
```

`Other` exists so a status never seen before is carried, not dropped or guessed at.

`games` imports nothing from the project. `sleeper` and `web` import it.

### D3. `sleeper` translates; it does not word

`Client.WeekGames(ctx, season, week) (games.Week, error)` decodes the array and maps:

| Sleeper | `games` |
|---|---|
| `status: pre_game` | `Upcoming` |
| `status: in_game` | `InProgress` |
| `status: complete` | `Complete` |
| any other `status` | `Other`, with `Status` set to the text |
| `start_time` | `Kickoff`, from epoch milliseconds |
| `metadata.quarter` | `Period`; null becomes `""` |
| `metadata.home_team`, `away_team` | `Home`, `Away` |

A game missing either team code is skipped. A non-200 or undecodable body is an error. `[]` is a
`Week` with no games, not an error.

No display wording is chosen here. `Final` and `Q3` are the page's decisions (D7).

### D4. One generic cache, renamed `weekcache`

`statscache.Cache` becomes `weekcache.Cache[V]`: keyed by season and week, holding any `V`.

- Its source is a function, `func(ctx, season, week) (V, error)`, so `client.WeekStats` and
  `client.WeekGames` are passed as method values and the cache needs no interface per value type.
- Its read method is `AsOf(ctx, season, week) (V, time.Time, error)`. `FetchedAt` and `Invalidate`
  keep their names.
- `New` takes a label used in its log lines. The stats cache keeps `statscache`, so its existing
  log lines do not change; the games cache uses `gamescache`.

`web` declares what it consumes, as now, with the method renamed to match:

```
StatsSource   AsOf(ctx, season, week) (score.WeekStats, time.Time, error)
GamesSource   AsOf(ctx, season, week) (games.Week, time.Time, error)
```

Go matches interfaces by method shape, so `*Cache[score.WeekStats]` satisfies `StatsSource` and
`*Cache[games.Week]` satisfies `GamesSource` with no adapter type in `cmd/server`.

*Alternatives:*

- A second, copied cache for games. Rejected: two copies of single-flight and failure handling to
  keep in step, and the next change replaces the expiry rule in both.
- Keep `WeekStatsAsOf` and add adapter structs in `cmd/server`. Rejected: more code than renaming
  one interface method and its test fakes.
- Keep the package name. Rejected: a `statscache` holding games misleads, and the rename is one
  directory move and two imports.

### D5. The refactor lands first and changes no behaviour

Three steps, each with the existing suite green and no test assertions changed:

1. Rename the package directory and imports.
2. Make `Cache` generic over `V`, with the function source and the label.
3. Rename `WeekStatsAsOf` to `AsOf`, in the cache and in `web`'s interface and fakes.

Only then are games added. A failure in the refactor is then never confused with a failure in the
new behaviour.

### D6. Games are read after stats, and their failure is tolerated

The handler order becomes: validate, resolve the matchup, read both lineups, fetch stats, fetch
games, render. Stats failing still fails the page. Games failing is logged and the page renders with
no game state on any row.

A week whose game list is empty is treated the same as a failed fetch. Otherwise a week Sleeper has
no games for would show every starter on bye.

The two fetches run one after the other. On a miss that adds roughly 100 ms to the first render.

*Alternative:* fetch both concurrently. Not worth the goroutine and error plumbing for 100 ms once
per five minutes.

### D7. What a row shows

For each starter, from the lineup record's team:

| Case | Shown |
|---|---|
| Games unavailable, or the week has none | nothing |
| Team is `FA` | nothing |
| Team has no game in a non-empty week | `Bye` |
| `Upcoming` | kickoff as weekday and time, in a `<time>` element |
| `InProgress`, `Period` all digits | `Q` and the digits: `Q3` |
| `InProgress`, any other non-empty `Period` | the text as sent: `HALF` |
| `InProgress`, empty `Period` | `Live` |
| `Complete`, `Period` begins with `F` | `Final` in place of that `F`: `Final`, `Final/OT` |
| `Complete`, empty `Period` | `Final` |
| `Complete`, any other `Period` | the text as sent |
| `Other` | the provider's status text as sent |

This is a pure function in `web`, tested without a server.

`Live` is the one word here that is ours and not Sleeper's. It covers a game Sleeper calls
`in_game` before it has sent a quarter, which has not been observed.

### D8. The game state is the third item on the meta line

`WR · CIN` becomes `WR · CIN · Q3`, in a `<span class="game">` inside the existing `.meta` span.
When there is nothing to show, the separator and the span are both left out, so the line is
`WR · CIN` exactly as today.

The meta line may wrap on a narrow column but must stay inside it. Rows are shared across the two
columns by subgrid, so a wrapped line in one column grows the same row in the other and the columns
stay aligned.

*Alternative:* a third line per row. Rejected: nine more lines per column on a phone.

### D9. The server writes Central; the script rewrites to the reader's zone

Server text, in `America/Chicago`:

- Kickoff: `Sun 12:00 PM`, with no zone label. With one, the meta line wrapped to three lines at
  390px, the label itself splitting. An unscripted reader sees an unlabelled Central time; the
  as-of line on the same page still names the zone.
- As-of line: unchanged, `Sun, Oct 11 2026 5:07 PM CDT`.

The script, after load, for each `<time>` with a `datetime`:

- inside `.game`: weekday and time, with no zone label, since it is now the reader's own clock.
- inside `.as-of`: the same shape as the server's, with the reader's zone abbreviation.

It formats with `Intl.DateTimeFormat("en-US", …).formatToParts` and joins the parts itself. Recent
browsers put a narrow no-break space before `PM`; joining the parts with an ordinary space keeps the
text the same shape as the server's. Any exception leaves the server's text in place.

The script reads `datetime` from the DOM, so it still holds no page data.

*Alternative:* a timezone cookie or query parameter so the server renders the reader's zone.
Rejected: the first load would still be Central, and the cached page would vary by reader.

### D10. The stats version token is unchanged

It stays the stats fetch instant. Stats and games usually expire together, since one page render
filled both, so a stats refetch that changes the token usually coincides with a games refetch. After
a forced-fumble invalidation the two clocks drift apart, and a status change then reaches other open
tabs on their five-minute reload.

### D11. Logging

- `sleeper games {season} w{week}: {n} games in {duration}` on a successful fetch, beside the
  existing `sleeper stats` line.
- `gamescache hit:` and `gamescache miss:` lines in the shape of the `statscache` ones.

## Risks / Trade-offs

- [The endpoint is undocumented and was observed on two days] → A failed or undecodable response
  drops only the game states (D6). Unknown statuses are shown as sent (D7).
- [A stale lineup team shows the wrong game] → Accepted. Points accruing beside a future kickoff
  time is a visible sign the file needs a one-line fix.
- [Game state can be about five minutes behind] → Same staleness as scores today. The next change
  revisits expiry.
- [An unscripted reader sees kickoffs in an unlabelled Central time] → Accepted for the phone fit
  (D9). The as-of line still names the zone.
- [The rewritten text flashes from Central to local on load] → The script is inline at the end of
  the body and runs before first paint in practice. Not verified on a slow device.
- [Zone abbreviations outside the US render as `GMT+2`] → Acceptable for the as-of line. Kickoff
  shows no label.
- [The cache refactor touches every cache test's setup] → Done as its own steps with no assertion
  changes (D5), and run with `-race`.

## Migration Plan

One deploy, no data migration. Rollback is redeploying the previous image. The lineup files and the
volume are untouched.

## Open Questions

- Is `Live` the right word for an in-progress game with no quarter text, if that state exists?

One thing found while reading the specs, not caused by this change: `weekly-stats-cache` says a
stats-only read path "SHALL remain available", but `Cache` has had only `WeekStatsAsOf` for some
time. This change leaves that requirement as it is.
