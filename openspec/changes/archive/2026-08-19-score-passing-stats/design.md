## Context

`Points` in `internal/score/calc.go` implements the rushing/receiving table of `docs/scoring.md` and
nothing else. It is position-agnostic — it reads stat fields, not positions — so wide receivers,
running backs, and tight ends already score correctly, since their production lands entirely in
those fields. Quarterbacks do not: their rushing counts and their passing does not.

Scored by hand against 2025 regular week 14:

| player | scored today | missing |
| --- | --- | --- |
| Josh Allen | 7.0 | 251 pass yd, 3 pass TD, 1 2-pt pass |
| Patrick Mahomes | 0 | 160 pass yd, 3 interceptions |

Allen's 7.0 is the shape of the problem — a plausible number, not an obviously empty one. Mahomes'
0 is worse in a different way: three interceptions should put him at -9, so the current score is
wrong in the player's favor and indistinguishable from a quiet day.

Every stat needed is already in the weekly payload the server fetches. Confirmed present in the live
2025 week 14 aggregate: `pass_yd`, `pass_td`, `pass_int` (39 players), `pass_td_40p` (3 players), and
the three conversion keys `pass_2pt` (4), `rush_2pt` (2), and `rec_2pt` (2). Sparse keys are the
normal case and already read as zero through `statLineFrom`'s missing-key handling, so sparseness
needs no new code.

The relevant rules, from `docs/scoring.md`:

| action | points |
| --- | --- |
| TD pass | 6 |
| 2XP pass, or 2XP scored | 2 |
| 200 passing yards | 3 |
| each additional 50 yards | 1 |
| interception thrown | -3 |
| any TD play of 40+ yards | 1 |

Two clarifications recorded there bear directly on this change. Yardage thresholds are floor-based
off the threshold, so 249 passing yards is 3 points and 250 is 4. And the 40+ bonus pays every
player credited on the play, so a 45-yard touchdown pass is +1 to the quarterback and +1 to the
receiver.

## Goals / Non-Goals

**Goals:**

- Score a quarterback's passing production under the league's passing rules.
- Score a two-point conversion consistently for every player credited on the play.
- Leave every currently-correct score bit-for-bit unchanged.
- Keep the floor-based threshold rule expressed once, so the passing and rushing/receiving tables
  cannot drift apart in how they round.
- Add no upstream request, dependency, or lookup.

**Non-Goals:**

- Kicking and defensive scoring. Substantially larger slices with their own questions — safeties,
  return touchdowns, turnover-qualified forced fumbles — and neither is needed to make a quarterback
  correct.
- Marking a score as partial. A response still cannot say which rule categories it covers, so a
  kicker still scores 0 rather than reporting itself unscored. Worth doing, but it is a response-shape
  change affecting every endpoint, and it should land once rather than once per category.
- Position awareness of any kind. The calculator stays a function of stats, which is why adding a
  category is additive rather than a branch.

## Decisions

### The passing yardage award stacks with the rushing/receiving award

A dual-threat quarterback with 250 passing and 80 rushing yards earns 4 for the passing and 3 for the
rushing — 7, not a single best-of-both award.

The alternative is folding passing into the existing `max()`, which would treat all yardage as one
competing pool. It was rejected on the source text: `docs/scoring.md` puts passing and
rushing/receiving in separate tables, and the "awarded at most once" clarification is written about
the rushing/receiving clauses specifically — it exists to stop 80 rushing *and* 80 receiving from
paying twice, which is a within-table concern about the same yards being counted under two clauses.
Passing yards are different yards on different plays. Folding them in would also silently reduce
scores for the exact players this change exists to fix.

### `thresholdBonus` is parameterized rather than duplicated

The existing helper hardcodes the rushing/receiving increment shape:

```go
func thresholdBonus(yards, threshold int) float64 {
    if yards < threshold { return 0 }
    return 3 + 0.5*float64((yards-threshold)/10)
}
```

Passing is the same shape with different constants: award 3 at 200, then 1 per 50. The helper gains
the increment size and per-increment value as parameters, and both tables call it.

The alternative — a separate `passingYardBonus` with its own integer-division floor — was rejected
because the floor-based rule is a single league clarification covering both tables, and duplicating
it invites the two copies to diverge under a future edit (one switching to rounding, say). One
parameterized helper keeps the rounding decision in one place. The cost is two more arguments at each
call site, which the call sites already make readable by passing literals from the rules table.

### `TD40Plus` absorbs passing touchdowns rather than gaining a sibling field

The field already sums 40+ yard rushing and receiving touchdowns because the bonus is a flat +1
regardless of how the touchdown was scored. Passing joins the same sum: `pass_td_40p + rush_td_40p +
rec_td_40p`.

A separate `PassTD40Plus` field was considered and rejected — it would carry no information the sum
does not, and `Points` would immediately add the two back together. The stat line is a scoring input,
not a box score; where the rules do not distinguish, neither should it.

This is safe against double-counting even though a single 45-yard touchdown pass sets `pass_td_40p`
on the quarterback and `rec_td_40p` on the receiver: those are two different players' stat lines, and
the rules intend both to be paid.

### Two Sleeper key names are traps, and both are named here deliberately

`pass_int` is interceptions *thrown*, which is what the -3 rule penalizes. Sleeper also carries `int`
and `idp_int` for interceptions *caught*; mapping either would pay the penalty to the wrong player.

`pass_rush_yd` is not passing yards. It is passing plus rushing yards combined — a convenience
aggregate in the same family as `rush_rec_yd`, which this codebase already ignores. It holds for
every entry carrying it in the live 2025 week 14 payload (Josh Allen: 251 + 78 = 329), and it appears
on non-quarterbacks, including the running back already in `internal/score/testdata/week14.json`.
Mapping it in place of `pass_yd` would fold a quarterback's rushing yards into their passing bonus
and hand a 200-yard rusher a passing award they never earned.

Both errors produce plausible numbers rather than failures, which is why the mapping tests assert on
distinct non-zero values per key rather than on a total.

### Two-point conversions are scored in all three flavors, not just passing

`docs/scoring.md` states the conversion twice — "2XP pass" in the passing table and "2XP scored" in
the scoring table — but it is one play paying two players, and Sleeper reports it as three parallel
keys (`pass_2pt`, `rush_2pt`, `rec_2pt`) that all appear in the same weekly payload with the same
shape.

Taking only `pass_2pt` because this change is nominally "passing" was the initial scope, and it was
wrong. It would have paid the passer 2 points and the player who actually scored nothing, which is
not a smaller version of the rule — it is a different, incorrect rule. Each additional key is one
term in the same sum, so the tidier scope boundary bought nothing and cost a known defect.

### The stat line grows; nothing else changes shape

`StatLine` gains `PassYd`, `PassTD`, `PassInt`, and a single `TwoPt` count summing the passing,
rushing, and receiving conversion keys — the rules pay 2 regardless of flavor, so, as with `TD40Plus`,
the stat line does not distinguish where the calculator would not. Every field is an `int` that reads
zero when absent, so a non-quarterback's stat line and score are unchanged, and both endpoints keep
their current JSON shape apart from the four added stat fields.

## Risks / Trade-offs

- **A quarterback's score still omits nothing visible.** After this change a quarterback is correct,
  but a kicker still returns 0 and a defender still returns 0, with no marker distinguishing "scored
  zero" from "not scored by any implemented rule". → Unchanged by this work and explicitly out of
  scope; the spec's purpose statement continues to name the remaining gaps.
- **Sleeper's passing keys are sparse, so a mapping typo reads as zero rather than failing.** A
  misspelled key silently produces a plausible score. → Covered by tests that map a fixture entry
  with known non-zero values for every new key, which a typo turns red.
- **Parameterizing `thresholdBonus` touches the code path that currently scores every player.** →
  The existing calculator tests pin the rushing/receiving numbers exactly and must stay green
  unmodified through the refactor; that is what makes the refactor safe to do first.
