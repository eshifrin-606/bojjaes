## Why

The league pays 4 points for a forced fumble that results in a turnover, and the scoreboard pays
nothing for it today. The weekly aggregate cannot say which forced fumbles were turnovers, so the
rule was deferred until play-by-play landed. Defenders on both rosters are underscored on exactly the
plays the league remembers. Official 2026 scores confirm the rule as written: week 2's non-turnover
forced fumbles paid 0, and week 3's Will Anderson scored 13.5 = 7.5 (sacks) + 2 (recovery) + 4
(forced fumble).

## What Changes

- Add a provider-neutral count of turnover-qualified forced fumbles to the stat line, and pay 4 points
  per fumble in `score.Points`.
- Read Sleeper play-by-play from the REST `plays/nfl/recent` endpoint, merging polled plays into a
  per-week store keyed by play ID. A whole-week fetch fills gaps and seeds past weeks. The weekly
  aggregate stays the source for every other stat.
- Attribute each forced fumble per play, taking turnover status from the fumbler's own row
  (`fum_lost > 0`). The forcer is named in the play description and resolved to a player row on the
  opposing team. Play rows carry only player IDs, so names and teams come from the weekly aggregate.
  Ambiguous or unmatched plays, including a forcer absent from the aggregate, award nothing and are
  logged.
- A play-by-play failure never blocks or fails the page. Forced fumbles fall back to what is already
  held, which is zero if nothing is held, and the failure is logged. An award may arrive minutes to an
  hour late.
- Remove forced fumbles from the "excluded at the aggregate stage" requirement. Separately, drop the
  stale claim that the 40+ yard bonus on defensive and return touchdowns needs play-by-play. That
  bonus is not a league rule (corrected 2026-10-04): it is paid on offensive touchdowns only.
- Safeties stay unscored. They are out of scope.

## Capabilities

### New Capabilities

- `forced-fumble-attribution`: deciding from Sleeper play-by-play which player forced each fumble and
  whether it was a turnover, plus how those plays are fetched and held per week.

### Modified Capabilities

- `player-week-score`: the stat line gains a turnover-qualified forced-fumble count, which the
  defensive rules pay 4 points each. The aggregate-stage exclusion no longer lists forced fumbles or
  the defensive/return 40+ bonus. The snapshot is built from the aggregate plus play-by-play
  attribution.

## Impact

- `internal/score`: `StatLine` gains a field and `Points` gains a term. The `Points` doc comment is
  corrected.
- `internal/sleeper`: new play-by-play fetch, per-week play store, description parser, and
  attribution. `Client` merges forced-fumble counts into the weekly snapshot.
- `cmd/server/main.go`: wires the play store into the Sleeper client.
- New upstream: `https://api.sleeper.com/plays/nfl/recent` (the same host as the stats API). It
  is CDN-cached for 300 s. A whole-week fetch is about 3.9 MB and takes 3–12 s.
- `internal/sleeper/testdata`: trimmed 2026 week 2 and week 3 play and weekly-aggregate fixtures.
- `openspec/specs/player-week-score/spec.md` and `docs/scoring.md`: deviation text updated on
  archive. ADR 0003 is not edited by this change.
