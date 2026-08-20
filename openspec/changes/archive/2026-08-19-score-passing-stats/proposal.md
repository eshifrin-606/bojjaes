## Why

The scorer implements only the rushing/receiving slice of `docs/scoring.md`, so a quarterback's ID
returns a confidently incomplete number rather than an error. In 2025 week 14, Josh Allen scores 7.0
today — his rushing line — while his 251 passing yards, 3 passing touchdowns, and 2-point conversion
pass contribute nothing. The number looks plausible, which is what makes it dangerous: nothing in the
response marks it as partial.

Passing is the largest remaining category and the cheapest to close. Every stat it needs is already
in the weekly payload the server fetches, so no new upstream call, dependency, or lookup is involved.

## What Changes

- Score the passing rules of `docs/scoring.md`: 6 points per passing touchdown, 2 per 2-point
  conversion pass, 3 points at 200 passing yards plus 1 per additional full 50 yards, and -3 per
  interception thrown.
- Extend the 40+ yard touchdown bonus to passing touchdowns, per the clarification that the bonus
  pays every player credited on the play.
- Score all three two-point conversions — passing, rushing, and receiving — at 2 points each. They are
  one rule in `docs/scoring.md`, they arrive as three parallel keys in the same payload, and splitting
  them across changes would pay a conversion to the passer while the player who scored it earned
  nothing.
- Map the new stats onto the domain stat line: `pass_yd`, `pass_td`, `pass_td_40p`, `pass_int`, and
  `pass_2pt`, `rush_2pt`, `rec_2pt`.
- The passing yardage award and the rushing/receiving yardage award are independent and stack. The
  "awarded at most once" rule stays confined to the three rushing/receiving clauses it was written
  for.

Not breaking. A player with no passing stats reads zero in every new field and scores exactly what
they score today; no existing response shape or endpoint changes.

Out of scope, deliberately: kicking and defensive scoring. Each is a substantial slice of the rules
with its own questions — safeties, return touchdowns, turnover-qualified forced fumbles — and neither
is needed to make a quarterback's score correct. Reporting *which* categories a score covers, so a
caller can tell a complete total from a partial one, stays a known gap rather than being solved here.

## Capabilities

### New Capabilities

None. This extends the existing scoring capability.

### Modified Capabilities

- `player-week-score`: the calculator gains the passing rules; the Sleeper transform maps the passing
  stats; the stat line carries them; the long-touchdown bonus covers passing touchdowns. The
  calculator also gains two-point conversions in all three flavors. The capability's purpose statement
  narrows its stated gap from "passing, kicking, defensive, and two-point-conversion scoring" to
  kicking and defense.

## Impact

- `internal/score/stats.go` — `StatLine` gains passing and two-point-conversion fields; `TD40Plus`'s
  comment widens to cover passing touchdowns.
- `internal/score/calc.go` — `Points` gains the passing terms, a passing yardage bonus alongside the
  existing rush/rec one, and a two-point-conversion term.
- `internal/score/sleeper.go` — `statLineFrom` maps the seven new Sleeper keys.
- `openspec/specs/player-week-score/spec.md` — amended in place.
- `README.md` — the documented sample response gains the new stat fields.
- No new dependencies, no new upstream requests, no change to any endpoint's shape.
