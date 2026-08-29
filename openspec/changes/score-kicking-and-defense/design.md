## Context

`internal/score` today fetches one Sleeper weekly aggregate, maps a player's entry onto an
all-integer `StatLine`, and computes points with a `Points` function that sums terms without
branching on position. That last property is the reason this change is small: kicking and defensive
scoring are new terms, not a new code path.

The constraint that shapes every decision here is that **the provider credits several stats to the
wrong player relative to how the rules read them.** This was verified on 2026-08-20 against a live
fetch of 2025 week 14 (2,142 entries), and is recorded in
`docs/basic-memory/2026-08-20-sleeper-aggregate-td-and-defensive-keys-sit-on-the-wrong-rows.md`.
Week 14 contains exactly one pick-six, which made attribution readable directly.

`docs/scoring.md` was amended the same day to scope its implementation deviations to this
aggregate-only stage and to record the attribution hazards.

## Goals / Non-Goals

**Goals:**

- Score kickers and defenders, who are already on rostered CSVs and currently return a silent `0`.
- Finish the unqualified touchdown rule, which is an existing underpayment on offensive players.
- Carry sacks at half-sack granularity without truncation.
- Make the deliberate exclusions legible in the code, so a later reader does not "fix" them.

**Non-Goals:**

- No play-by-play. Forced fumbles, safeties, and the 40+ bonus on defensive and return touchdowns
  wait for it, and none is attempted approximately.
- No second provider surface, no additional network call, no new dependency.
- No position awareness. A stat line does not know whether its player is a kicker.
- No team-defense scoring. The league is IDP; team rows are not players.

## Decisions

### Sacks become a `float64` field; every other stat stays `int`

`idp_sack` carries values in {0.5, 1.0, 1.5, …} and the rules pay 3 per sack, so a half sack pays
1.5. The existing transform reads every stat through `int(raw[key])`, which truncates 0.5 to 0 and
loses the points silently.

*Alternative considered:* store half-sacks as an integer count of halves and halve it in the
calculator. That keeps `StatLine` uniformly integer, at the cost of a field whose stored value never
matches the stat anyone quotes — "3 halves" for 1.5 sacks. Rejected: the domain object's stated
purpose is to hold the stat as the league scores it, and an honest mixed-type struct is cheaper to
read than a unit trap.

The transform needs a `statFloat` reader beside the existing `stat` one. The underlying payload is
already decoded as `map[string]float64`, so this removes a conversion rather than adding one.

### The touchdown term is an explicit allowlist, with exclusions named in the code

```
6 × (pass_td + rush_td + rec_td + idp_def_td + st_td)
```

Not scored, and each named in a comment where a reader would look for it:

| Stat | Sits on | Scoring it would |
| --- | --- | --- |
| `pass_int_td` | the quarterback who **threw** the pick-six | pay him +6 for a play the rules dock 3 |
| `pass_sack` | the quarterback **sacked** | pay the victim the tackler's 3 |
| `kr_td`, `pr_td`, `def_td`, `safe` | **team rows** (`TEAM_SEA`, `BUF`) | never fire on a player lookup, but mislead |
| `td`, `anytime_tds` | mixed aggregates | double-count against the flavors above |

And in the recovery term, one more with the same shape:

| Stat | Sits on | Scoring it would |
| --- | --- | --- |
| `fum_rec` | the player who recovered **their own team's** fumble | pay 2 for a play that was never a turnover |

Every one of these is individually plausible. `anytime_tds` in particular looks like a shortcut for
the whole rule and is not one — it covers offense and special teams only, and the week-14 pick-six
defender does not carry it. The comment is the deliverable here as much as the code.

*Alternative considered:* filter team rows defensively in the transform. Unnecessary — lookups are
by rostered player ID and never touch a non-numeric key. Noted rather than coded.

### Fumble recoveries sum three keys, two of which are usually absent

`idp_fum_rec + st_fum_rec + def_st_fum_rec` reproduces the turnover-qualified recovery set 268/269,
per the 2026-08-09 full-season validation. The IDP key alone undercounts, missing special-teams
recoveries.

The reason `idp_fum_rec` is clean is that own-team recoveries land in the separate, non-IDP
`fum_rec` key. That makes `fum_rec` the single most dangerous omission in this change after
`pass_int_td`: it is the obvious name for the rule, and adding it would quietly convert a
turnover-qualified term into a raw one. It is named in the exclusion comment for that reason.

The one known miss is `idp_fum_rec = 1` credited on an interception return where the interception
was already the turnover — the same case as the first open question below. The requirement records
this fidelity rather than leaving it only here.

The wrinkle for testing: `st_fum_rec` and `def_st_fum_rec` had **zero occurrences** in week 14. They
are sparse keys that appear only in weeks a qualifying play happened, so a fixture captured from an
arbitrary week leaves this branch untested while the suite stays green. The fixture for this branch
is hand-built rather than captured.

### Interceptions caught are read from `idp_int`, not `int`

The existing comment in `sleeper.go` names both `int` and `idp_int` as interceptions caught. Only
`idp_int` has been observed on a real row: the week-14 pick-six defender (8487) carries
`idp_int: 1` alongside `idp_def_td: 1`, and `docs/probe-espn-sleeper.md` row 1.16 maps the league's
IDP interception rule onto `idp_int` as well. `int` is unverified here, and a generic-sounding key in a
payload that also carries team rows is exactly the kind of thing this change has already been burned
by. The verified key wins; the comment is tightened to say which one is evidenced.

### `Points` grows terms; no function is restructured

The existing `thresholdBonus` helper is untouched — kicking and defensive rules are flat per-event
rates with no thresholds, so they need nothing from it. `Points` gains four lines. No branch on
position is introduced, preserving the property its comment already claims.

### Exclusions are asserted by test, not just omitted

A rule that is deliberately unscored is indistinguishable from one that was forgotten unless
something says so. Forced fumbles, safeties, and the defensive 40+ bonus each get a test asserting
they contribute **zero**, so removing the exclusion later breaks a test that explains why it existed.

All three are *transform* tests. Forced fumbles and safeties reach no `StatLine` field at all, so
writing them against the calculator would require adding the very field the exclusion exists to
prevent. The defensive 40+ bonus looked like a calculator test — both `IntCaught` and `DefTD` reach
the stat line — but the excluded thing is not a term: `Points` adds `TD40Plus` unconditionally, and
what withholds the bonus is that nothing populates `TD40Plus` from a defensive play. Asserting that
needs a distance the stat line does not carry, which is the play-by-play data this change excludes.

## Risks / Trade-offs

- **A future contributor maps `pass_int_td` as "the missing TD stat."** → Named in a code comment at
  the mapping site, asserted by a test ("a quarterback who threw a pick-six scores -3"), and recorded
  in `docs/scoring.md` under Attribution hazards. Three independent tripwires, because this is the
  most expensive mistake available in this change.
- **Half-sack truncation reappears** if a later stat is added through the integer reader by habit. →
  The `1.5 sacks pays 4.5` test covers the calculator; the transform test covers the reader.
- **Fumble-recovery branch is under-exercised** by real fixtures. → Hand-built fixture, plus a test
  asserting the sparse keys' absence reads as zero rather than erroring.
- **Kickers gain a large score swing.** A kicker previously scoring `0` may now post double digits.
  This is correct, but it will visibly change any saved comparison against the spreadsheet. → Not
  mitigated; it is the point of the change.
- **The excluded rules mean defenders are systematically underpaid** relative to the spreadsheet,
  most visibly on forced fumbles. → Accepted and documented as stage-scoped. A visible omission is
  preferred to a knowingly wrong award, consistent with how safety is already handled.

## Migration Plan

None required. No stored state, no API contract break — the response gains fields and some players'
totals rise. Rollback is reverting the commit.

Two prose statements go stale and neither is rewritten by applying the deltas, so both are explicit
tasks (8.3): the capability's Purpose paragraph in `openspec/specs/player-week-score/spec.md`
("kicking and defensive scoring are not yet specified"), and the doc comment on `Points` in
`internal/score/calc.go`, which says the same thing about the implementation.

## Open Questions

These need a commissioner ruling rather than more data, and none blocks this change:

- **Own-team fumble recovery after an interception return.** The provider credits a recovery; the
  interception was already the turnover. Does the 2 points pay? Affects the recovery term's edge.
- **Shared-credit safety.** The solo/assisted distinction rests on two clean observations, and must
  be confirmed before the safety exclusion is lifted.
- **Is the provider's 40+ touchdown bucket inclusive at exactly 40?** Our rule is inclusive. This
  affects the already-shipped passing, rushing, and receiving path, not the work here.
