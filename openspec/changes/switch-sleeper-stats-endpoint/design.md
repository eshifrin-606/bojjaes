## Context

`fetchWeekly` decodes `api.sleeper.app/v1/stats/nfl/regular/S/W` into
`map[string]map[string]float64`, and `statLineFrom` maps one entry into a `score.StatLine`.
`FetchWeekStats` runs every entry through `statLineFrom`, skipping null entries. The tests serve
`testdata/week14.json`, an 11-entry trimmed copy of 2025 wk 14 in map shape that includes hand-built
entries, from an `httptest` server that checks the request path.

The `.com` endpoint returns `[{player_id, stats, team, player{...}, ...}]`. See the proposal for the
parity check.

## Goals / Non-Goals

**Goals:**

- Same stat lines, same absence semantics, and the same scores from the new endpoint.
- Keep every Sleeper shape inside `internal/sleeper`.

**Non-Goals:**

- Exposing names or teams. The forced-fumble change adds those fields when it uses them.
- Any change to the stat-key mapping, `statscache`, or the page.

## Decisions

### D1. Decode rows, then index by player ID

`fetchWeekly` decodes `[]weeklyRow{PlayerID string; Stats map[string]float64}` and builds the same
`map[string]map[string]float64` it returns today. `statLineFrom` and `FetchWeekStats` are then
unchanged, and so are the null-entry and absence semantics: a row with `"stats": null` indexes to a
nil map, which `statLineFrom` already reports as absent.

*Alternative:* change `statLineFrom` to take a row. Rejected for this change: it rewrites ~20 tests
for no behavior. The forced-fumble change can revisit it if it needs the row.

### D2. Path and query

The URL is `{base}/stats/nfl/{season}/{week}?season_type=regular`, and `BaseURL` becomes
`https://api.sleeper.com`. The fixture server checks the path and the `season_type` query.

### D3. Convert the fixture, do not re-record it

`week14.json` is rewritten as a row array: one `{player_id, stats}` row per existing map entry, in
the same order, with the same stats, hand-built entries included. Existing test expectations then
hold unchanged, which is the evidence that the switch changes no scores. A re-recording would bring
in stat corrections since the fixture was trimmed (player 7591 is no longer in either live
endpoint's 2025 wk 14), and that would blur the evidence.

`team` and `player` are left out of the converted fixture. The forced-fumble change adds them with
the fields that read them.

### D4. Unplayed week

`.com` returns `[]` for an unplayed week. That decodes to an empty map, so the "an empty payload is
not an error" behavior carries over. The `{}` tests become `[]`.

### D5. Duplicate player IDs

None were seen in 2026 wk 3. If one appears, the later row wins, which is what a map decode did for a
repeated key. This is not tested; it is a guess about a shape never seen.

## Risks / Trade-offs

- **Payload ~3.7× larger** (2.1 MB raw, ~280 KB gzipped). → ~30 ms more transfer, more decode time.
  `BenchmarkDecode` measures it; record before/after in the PR. It runs at most once per TTL per
  week.
- **Longer stale-while-revalidate (600 s vs 300 s).** → A rarely read week can be served once up to
  ~10 min stale. A shorter `s-maxage` makes the steady state fresher. Recorded in ADR 0003.
- **Both endpoints are undocumented.** → Unchanged risk; ADR 0003 already carries it.

## Migration Plan

Deploy. To roll back, revert the commit; the map decoder and `.app` URL return.
