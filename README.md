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

Hit the endpoint:

```bash
curl -s localhost:8080/score | jq
```

```json
{
  "stats": {
    "player_id": "9493", "season": 2025, "week": 14,
    "pass_yd": 0, "rush_yd": 0, "rec_yd": 167,
    "pass_td": 0, "rush_td": 0, "rec_td": 2,
    "td_40_plus": 0, "two_pt": 0, "pass_int": 0, "fum_lost": 0
  },
  "points": 19
}
```

`GET /score` is a fixed smoke test: Puka Nacua's 2025 week 14, a settled result
that should never change. Score any players with `POST /scores`:

```bash
curl -s localhost:8080/scores -d '{
  "season": 2025, "week": 14,
  "player_ids": ["9493", "8138", "4034"]
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
        "pass_td": 0, "rush_td": 0, "rec_td": 2,
        "td_40_plus": 0, "two_pt": 0, "pass_int": 0, "fum_lost": 0
      },
      "points": 19
    },
    {
      "stats": {
        "player_id": "8138", "season": 2025, "week": 14,
        "pass_yd": 0, "rush_yd": 80, "rec_yd": 31,
        "pass_td": 0, "rush_td": 0, "rec_td": 0,
        "td_40_plus": 0, "two_pt": 0, "pass_int": 0, "fum_lost": 1
      },
      "points": 0.5
    }
  ],
  "no_stats": ["4034"]
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

## Scoring a roster

`scripts/scores.sh` scores a saved list of players against a running server, so
you read names instead of Sleeper IDs. Needs `curl` and `jq`.

```bash
go run ./cmd/server        # one shell
scripts/scores.sh 2025 14  # another
```

```
season 2025 week 14
Amon-Ra St. Brown        3.5
Puka Nacua               19
Luther Burden III        0
Tee Higgins              15.5
Ja'Marr Chase            0
```

```bash
scripts/scores.sh <season> <week> [players-file]
```

The players file defaults to `scripts/players.csv` — one `id,name` per line,
blank lines and `#` comments skipped, output in file order. Names are local
labels the server never sees, so a wrong name pairs silently with the right
stats. `SERVER=http://host:port` points the script somewhere other than
`localhost:8080`.

A player the server put in `no_stats` prints `no stats`, which is not the same
as a real `0` — see the `no_stats` caveat above for why the difference matters.
Server errors exit non-zero with the server's message rather than printing a
partial roster.

Tests and compile check:

```bash
go test ./...
go build ./...
```