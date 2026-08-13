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
    "rush_yd": 0, "rec_yd": 167, "rush_td": 0, "rec_td": 2,
    "td_40_plus": 0, "fum_lost": 0
  },
  "points": 19
}
```

Player, season, and week are still constants in `cmd/server/main.go`. The server
calls the live Sleeper API, so it needs network access; a fetch failure returns
502 rather than a misleading `0.0`.

Tests and compile check:

```bash
go test ./...
go build ./...
```