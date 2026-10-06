## 1. Aggregate fetch line

- [x] 1.1 Test: `Client.WeekStats` with a recording `logf` logs `sleeper stats {s} w{w}: {rows} rows in …` on success. Add the `Client.logf` field (nil is silent) and emit the line
- [x] 1.2 Wire `log.Printf` into `Client` in `cmd/server/main.go`

## 2. Play-by-play lines

- [x] 2.1 Test: a successful poll logs `sleeper plays poll {s} w{w}: {n} plays in …`. Emit it in `ForcedFumbles`
- [x] 2.2 Test: a cold week logs `sleeper whole-week {s} w{w}: started (cold)`. Pass a reason into `startWholeFetch` and log only when a flight starts
- [x] 2.3 Test: held plays with a gap log `started (gap)`
- [x] 2.4 Test: a quiet week with no whole fetch since logs `started (postgame)`
- [x] 2.5 Test: a read joining an in-flight whole fetch logs no start line
- [x] 2.6 Test: a successful whole fetch logs `sleeper whole-week {s} w{w}: {n} plays in …`. Emit it in `fetchWholeWeek`

## 3. Verify

- [x] 3.1 `go test ./...` passes; run locally and check the line sequence for one miss
