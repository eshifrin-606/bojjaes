# Summary

This repo is to assist fantasy football team owners in the he man fantasy football league.

Bojjaes on top.

## Local setup

Requires Go 1.26.5 (see `go.mod`). No other dependencies, env vars, or config.

Run the server:

```bash
go run ./cmd/server
```

It logs `listening on :8080` and stays in the foreground (Ctrl-C to stop).

### The matchup page

`GET /{season}/{week}` serves that week's matchup as an HTML page: the two teams
of that week's lineup directory, their nine starters with points, and each
lineup's total. The Bojjaes always hold the left column.

```
open http://localhost:8080/2025/15
```

No team appears in the URL — the week directory in the lineup tree names both.
A week that is not exactly one Bojjaes matchup is refused rather than guessed
at: a week we never played is a `404`, a malformed week directory is a
`500`, and a failed upstream fetch is a `502` rather than a page of zeros.

Lineups are hand-edited at `internal/lineup/data/<season>/<week>/<team>.csv`.
The tree is compiled into the binary with `//go:embed`, so the server reads no
file off the working directory — and an edited lineup reaches the page only
after a rebuild.

The page shows both totals and nowhere shows their difference. A starter whose
game has not kicked off is indistinguishable from one who was inactive, so it
carries no margin, win probability, or leader highlight.

### The scoring endpoint

Hit the endpoint. `POST /scores` takes a season, a week, and up to a roster's
worth of player IDs — the equivalent of the old `GET /score` smoke test is
Nacua's settled 2025 week 14:

```bash
curl -s localhost:8080/scores -d '{
  "season": 2025, "week": 14, "player_ids": ["9493"]
}' | jq
```

It is interim: it exists so `scripts/scores.sh` and `scripts/fantasycast.sh` can
run, and it retires with them when the matchup page replaces them.

Score a whole lineup the same way:

```bash
curl -s localhost:8080/scores -d '{
  "season": 2025, "week": 14,
  "player_ids": ["9493", "4227", "12561"]
}' | jq
```

```json
{
  "season": 2025,
  "week": 14,
  "scores": [
    {
      "stats": {
        "player_id": "9493", "season": 2025, "week": 14,
        "pass_yd": 0, "rush_yd": 0, "rec_yd": 167,
        "pass_td": 0, "rush_td": 0, "rec_td": 2, "def_td": 0, "return_td": 0,
        "td_40_plus": 0, "two_pt": 0,
        "pass_int": 0, "int_caught": 0, "fum_lost": 0, "fum_rec_turnover": 0,
        "sack": 0, "fg_made": 0, "xp_made": 0, "fg_50_plus": 0
      },
      "points": 19
    },
    {
      "stats": {
        "player_id": "4227", "season": 2025, "week": 14,
        "pass_yd": 0, "rush_yd": 0, "rec_yd": 0,
        "pass_td": 0, "rush_td": 0, "rec_td": 0, "def_td": 0, "return_td": 0,
        "td_40_plus": 0, "two_pt": 0,
        "pass_int": 0, "int_caught": 0, "fum_lost": 0, "fum_rec_turnover": 0,
        "sack": 0, "fg_made": 1, "xp_made": 1, "fg_50_plus": 0
      },
      "points": 4
    }
  ],
  "no_stats": ["12561"]
}
```

Season and week are read as the **regular season** — there is no way to ask for
preseason or postseason. At most 26 player IDs per request, the league's maximum
roster size.

Players Sleeper has no entry for come back in `no_stats` rather than failing the
request. **Absence does not identify its own cause**: a player whose game has not
kicked off, one who was inactive or a healthy scratch, a typo'd player ID, and a
week that has not been played yet all look identical in the payload. The server
reports the ID and leaves the diagnosis to you — so check `no_stats` before
treating `scores` as a complete lineup.

The server calls the live Sleeper API, so it needs network access. A fetch
failure returns 502 rather than a misleading `0.0`.

## Scoring a lineup

`scripts/scores.sh` scores a saved list of players against a running server, so
you read names instead of Sleeper IDs. Needs `curl` and `jq`.

```bash
go run ./cmd/server        # one shell
scripts/scores.sh 2025 14  # another
```

```
STARTERS
Amon-Ra St. Brown             3.5
Puka Nacua                     19
Luther Burden III               0
Colston Loveland                6
Jaylen Warren                   6
Josh Allen                     31
Harrison Butker                 4
Odafe Oweh                      3
Derrick Harmon           no stats
TOTAL                        72.5

BENCH
```

```bash
scripts/scores.sh <season> <week> [team|players-file]
```

Lineups live under `internal/lineup/data/<season>/<week>/`, and the third
argument is read against that directory. It defaults to `bojjaes.csv` there — so
`scripts/scores.sh 2025 15` reads the week 15 lineup without being told to — and
a bare team name picks another file from the same week:

```bash
scripts/scores.sh 2025 14 wood   # internal/lineup/data/2025/14/wood.csv
```

An argument containing a `/` or ending in `.csv` is used as a path instead, so
files outside the tree still work:

```bash
scripts/scores.sh 2025 14 scripts/teams/aroma.csv
```

Because the shorthand is built from the season and week you asked for, it can
never read another week's lineup.

A players file is one `id,name` per line, blank lines and `#` comments skipped,
output in file order within each section. Names are local labels the server
never sees, so a wrong name pairs silently with the right stats.
`SERVER=http://host:port` points the script somewhere other than
`localhost:8080`.

The **first nine records are the starters** and the rest are the bench. That is
positional only: the file carries no position column, so nothing checks that
rows 1-9 form a legal lineup, and reordering two lines changes who starts. The
file *is* the lineup card. The `BENCH` heading prints even when nothing follows
the ninth record, as it does for every lineup file in this repo today.

`TOTAL` sums the starters only. Bench players are scored and printed — useful
for "should I have started him" — but never totalled, since bench points count
for nothing.

A player the server put in `no_stats` prints `no stats`, which is not the same
as a real `0` — see the `no_stats` caveat above for why the difference matters.
Such a starter contributes nothing to `TOTAL`, and `TOTAL` carries no marker
saying so: the `no stats` line sits directly above it. In the example above,
`72.5` is eight players, not nine. Server errors exit non-zero with the
server's message rather than printing a partial lineup.

The report carries no season or week heading of its own — the season and week
are your own arguments — and points sit in a fixed-width column so a report can
be set beside another one without ragging. That is what the matchup report does.

## A matchup, side by side

`scripts/fantasycast.sh` prints two lineups as columns, each column a full
`scores.sh` report, with the season and week stated once above both.

```bash
scripts/fantasycast.sh <season> <week> <team> [team2]
```

```
season 2025 week 14

STARTERS                           STARTERS
Amon-Ra St. Brown             3.5  CeeDee Lamb                     5
Puka Nacua                     19  Brock Bowers                    6
Luther Burden III               0  Quinshon Judkins                0
Colston Loveland                6  Breece Hall                     0
Jaylen Warren                   6  Bijan Robinson                  0
Josh Allen                     31  Patrick Mahomes                -9
Harrison Butker                 4  Tyler Loop                     10
Odafe Oweh                      3  Myles Garrett                   3
Derrick Harmon           no stats  Micah Parsons                   0
TOTAL                        72.5  TOTAL                          15

BENCH                              BENCH
```

One team name means that team is the Bojjaes' opponent, so the common case is
the shortest command. Two names show those two teams and the Bojjaes do not
appear. **Argument order is column order**: the Bojjaes, or the first team you
name, is the left column.

```bash
scripts/fantasycast.sh 2025 14 wood            # bojjaes vs wood
scripts/fantasycast.sh 2025 16 gonads bojjaes  # gonads on the left
```

Team names resolve exactly as they do for `scores.sh`, because they are handed
straight to it — a name means the same file in both tools.

The two `TOTAL` lines face each other, and **nothing computes a margin**. A
starter whose game has not kicked off is indistinguishable from one who was
inactive, so a difference printed on Sunday morning would read as a settled
deficit when it is nothing of the kind. Read the two totals and the `no stats`
lines above them together.

Columns are independent lists, not a positional matchup — the lineup file has no
position column, so row *n* on the left does not face row *n* on the right. When
one lineup is longer, its extra lines simply print with nothing to their right;
neither column is padded or truncated to make the `BENCH` headings line up.

**Each team is scored in its own request**, one `POST /scores` per column. So the
server's per-request player cap applies per lineup rather than per matchup, and
two full lineups work where a merged request would not. Both reports are
captured before anything prints, so a bad team name or a server error exits
non-zero with that message and no half-drawn matchup.

Tests and compile check:

```bash
go test ./...
go build ./...
```
## Design docs

[docs/package-dependencies.md](docs/package-dependencies.md) maps how the Go packages relate — a
diagram, each package's imports, and where the next dependency is likely to land. Start there
before deciding where new code belongs. Decisions behind the current shape are in
[docs/adr/](docs/adr/), and the league's scoring rules in [docs/scoring.md](docs/scoring.md).
