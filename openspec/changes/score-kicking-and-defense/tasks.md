Each numbered pair below is one red-green iteration: write the test, run it, confirm it fails for
the stated behavioral reason, then make the smallest change that passes it. Tests come from the
delta spec's scenarios. Do not implement a later group's stats to make an earlier group's test pass.

## 1. Sacks force the stat line's first fractional field

- [x] 1.1 Test: a stat line crediting half a sack scores 1.5. Fails — no sack stat exists.
- [x] 1.2 Add a fractional `Sack float64` field to `StatLine` and a `3 × Sack` term to `Points`.
      Confirm green.
- [x] 1.3 Test: 1.5 sacks scores 4.5, and 2 sacks scores 6. Confirm the award is proportional
      rather than tabulated, and that no rounding was introduced.
- [x] 1.4 Test: a stat line with no sacks is unaffected — the score is exactly what the existing
      rules produce. Guards against a stray constant in the new term. No new test was needed: the
      20 pre-existing cases all carry `Sack: 0`, and a mutation to `3*s.Sack + 1` fails every one
      of them.

## 2. Kicking points

- [x] 2.1 Test: 3 field goals made, none from 50+, scores 9. Fails — no field goal stat.
- [x] 2.2 Add `FGMade` and its `3 ×` term. Confirm green, and confirm distance affects nothing.
- [x] 2.3 Test: 4 extra points made scores 4. Fails. Add `XPMade` and its `1 ×` term.
- [x] 2.4 Test: 2 field goals, one from 50+, scores 7. Fails — the bonus is unimplemented. Add
      `FG50Plus` and its `1 ×` bonus term.

## 3. Defensive points

- [x] 3.1 Test: 2 interceptions caught scores 12. Fails — interceptions caught do not exist, and
      must not be confused with `PassInt`, which is a -3 penalty.
- [x] 3.2 Add `IntCaught` and its `6 ×` term. Confirm green.
- [x] 3.3 Test: a stat line carrying 3 interceptions *thrown* still scores -9, and a stat line
      carrying 3 interceptions *caught* scores 18. Confirms the two stats did not get crossed.
- [x] 3.4 Test: 1 fumble recovery resulting in a turnover scores 2. Add the bare `FumRec` field so
      the failure is a wrong number rather than a build error; no term yet.
- [x] 3.5 Add the `2 × FumRec` term. Confirm green.

## 4. Finish the unqualified touchdown rule

- [x] 4.1 Test: a defensive touchdown alone scores 6. Fails — the touchdown term covers only
      passing, rushing, and receiving.
- [x] 4.2 Add `DefTD` to the existing 6-point touchdown term rather than a new term, keeping the
      rule in one place. Confirm green.
- [x] 4.3 Test: an interception caught and returned for a touchdown scores 12. Should pass from
      3.2 and 4.2 together; it asserts the two rules compose as `docs/scoring.md` says.
- [x] 4.4 Test: a receiver with 40 receiving yards and a punt-return touchdown is paid 6 for the
      return. Add the bare `ReturnTD` field so the failure is a wrong number rather than a build
      error; leave it out of the term.
- [x] 4.5 Add `ReturnTD` to the same 6-point touchdown term. Confirm green.
- [x] 4.6 Test: the 40+ yard bonus still applies to a 40+ receiving touchdown. Regression guard on
      the existing offensive path.

## 5. Assert the deliberate exclusions

These should pass without production changes. They exist so that a later contributor who removes an
exclusion breaks a test that explains why it was there.

**None of the three exclusions is a calculator test.** Forced fumbles and safeties reach no
`StatLine` field at all, so a calc test for them could only exist by adding the very field the
exclusion is meant to prevent. The 40+ bonus on defensive and return touchdowns is the same shape
once you try to write it: `StatLine` carries no return distance — that needs the play-by-play data
this change does not introduce (`proposal.md`), because the aggregate reports return yardage only as
a weekly sum with no distance attributable to the scoring play (`spec.md`, "Scoring rules excluded
at the aggregate stage"). All three are asserted at the payload level in 6.15.

- [x] 5.1 Both halves of the "interception returned 63 yards for a touchdown scores 12, not 13"
      scenario are already covered, and neither is a new calc test. The scoring half is the `pick
      six` case from 4.3, which is the only form the scenario takes as calculator input once the
      unexpressible 63 yards is dropped — writing it again under another name would duplicate it.
      The exclusion half belongs to 6.15, whose payload carries `idp_int_ret_yd` and
      `idp_fum_ret_yd` and asserts they score 0 and reach no stat-line field. `Points` adds
      `TD40Plus` unconditionally; what keeps a defensive touchdown from earning the bonus is that
      nothing ever populates `TD40Plus` from a defensive play, which only the transform can assert.

## 6. Transform: map the new stats

- [x] 6.1 Test: a payload entry carrying `fgm`, `fgm_50p`, and `xpm` populates each kicking field
      distinctly, so a mistyped key cannot read as zero. Fails — unmapped.
- [x] 6.2 Map the kicking keys. Use `fgm_50p` alone for the bonus; it equals `fgm_50_59 + fgm_60p`
      with zero mismatches across the verified week, so summing is redundant.
- [x] 6.3 Test: a payload entry crediting `idp_sack: 0.5` yields 0.5 on the stat line, not 0.
      Fails — the existing reader truncates through `int`.
- [x] 6.4 Add a `statFloat` reader beside `stat` and map `idp_sack` through it.
- [x] 6.5 Test: `idp_int` and `idp_def_td` populate their fields, and `pass_int` still lands on the
      passer's penalty. Fails — the defensive keys are unmapped.

      Both fields already exist on `StatLine`, so the failure is a wrong number rather than a build
      error. The `pass_int` half keeps the caught and thrown interceptions from being crossed at the
      mapping site, the way 3.3 does at the calculator.

- [x] 6.6 Map `idp_int` to `IntCaught` and `idp_def_td` to `DefTD`. Confirm green.

      Use `idp_int`, not the generic `int`. The week-14 pick-six defender was observed carrying
      `idp_int: 1` alongside `idp_def_td: 1`, and that is the only interception-caught key this
      project has verified on a real row. `int` is unverified here — the existing comment in
      `sleeper.go` names it from Sleeper's documentation, not from observation.

- [x] 6.7 Test: a payload entry carrying `pass_int_td` — a quarterback who threw a pick-six —
      scores -3, not +3 or +6.

      **This is the change's most expensive possible mistake.** A guard, not a red-green step: it
      passes on correct code, because the key is never mapped. Verify it has teeth by *temporarily*
      mapping `pass_int_td` into the touchdown term, watching the test fail, then reverting. Do not
      commit the temporary mapping.

- [x] 6.8 Test: a payload entry carrying `pass_sack` contributes nothing to the sack count.

      Confirms sacks taken were not crossed with sacks recorded — `pass_sack` sits on the
      quarterback who was sacked, not the defender who sacked him. Same shape as 6.7: verify by
      temporarily mapping `pass_sack` into `Sack`, then revert.

- [x] 6.9 Comment the excluded keys at the mapping site, naming the row each one sits on.

      The keys are `pass_int_td`, `pass_sack`, `fum_rec`, `kr_td`, `pr_td`, `def_td`, `td`, and
      `anytime_tds`. `anytime_tds` is not a shortcut for the rule — it omits defensive touchdowns.
      Done as two comments rather than eight: one above the touchdown keys covering every excluded
      TD flavor, one on `Sack` for `pass_sack`. `fum_rec` moved to 6.13, which is where its
      mapping site first exists.

- [ ] 6.10 Test: `st_td` populates the return touchdown field. Fails — unmapped.

- [ ] 6.11 Map `st_td` to `ReturnTD`. Confirm green.

- [ ] 6.12 Test: `idp_fum_rec`, `st_fum_rec`, and `def_st_fum_rec` sum into one recovery count.
      Fails — none of the three is mapped.

      Give the three keys distinct values so a dropped term cannot pass as a coincidence. The IDP
      key alone undercounts: it misses special-teams recoveries.

- [ ] 6.13 Map the three-key sum to `FumRec`. Confirm green, and comment the `fum_rec` exclusion
      here.

      `fum_rec` is the exclusion that needs the comment most after `pass_int_td`: it is the
      obvious-looking name for "fumble recovery" and is the wrong key — it holds **own-team**
      recoveries, which are not turnovers, and excluding it is precisely what leaves `idp_fum_rec`
      turnover-qualified. Moved here from 6.9, which ran before this mapping existed.

- [ ] 6.14 Test: a payload entry carrying only `idp_fum_rec`, with the two special-teams keys
      absent entirely, reads as that count rather than erroring.

      These keys are sparse — week 14 contains zero occurrences of either — so most real entries
      take this path.

- [ ] 6.15 Test: a payload entry carrying `idp_ff`, `idp_safe`, `idp_int_ret_yd`, and
      `idp_fum_ret_yd` and nothing else scores **0**, and none of those values reaches a stat-line
      field.

      This is where the forced-fumble and safety exclusions from section 5 are actually asserted,
      at the only level where they are expressible. Comment the test with why each is excluded —
      the turnover-qualification rule and its ~44% overpay for `idp_ff`, the unconfirmed
      solo-credit qualifier for `idp_safe`. Guard, not red-green: verify it has teeth by
      temporarily mapping one key, then revert.

## 7. Fixtures

- [ ] 7.1 Extend `internal/score/testdata/week14.json` with real entries for a kicker, a defender
      with fractional sacks, the week-14 pick-six defender, and the quarterback who threw it.
      Real payload shapes, so the fixture cannot drift from the provider's actual key set.
- [ ] 7.2 Hand-build the special-teams fumble-recovery entry. Week 14 contains zero occurrences of
      `st_fum_rec` and `def_st_fum_rec`, so a captured fixture leaves 6.12 untested while green.
- [ ] 7.3 Update `TestFetchWeekly`'s entry-count assertion. `internal/score/sleeper_test.go:52`
      asserts `len(weekly) == 6`; 7.1 and 7.2 add entries, so that number must move with the
      fixture. This is an expected update, not a regression. The batch tests are unaffected — their
      absent-player IDs are `"nope"`, which no new entry claims.
- [ ] 7.4 Confirm the existing offensive fixture cases still produce the same scores. The touchdown
      term and the stat line both changed shape; no offensive player's total should move.

## 8. Documentation

- [ ] 8.1 Refresh the `README.md` examples. The `/score` and `/scores` stat blocks match the served
      JSON today and go stale *because of* this change — the stat line gains fields. Separately, the
      `scripts/scores.sh` sample output lists players who are not in `scripts/bojjaes.csv` (Tee
      Higgins, Ja'Marr Chase), which is stale already. Regenerate both from the real roster, which
      puts a kicker and two defenders scoring nonzero in the README as a side effect.
- [ ] 8.2 Fix the players-file default while in that paragraph. `README.md:112` and
      `scripts/scores.sh:22` both default to `scripts/players.csv`, which does not exist — the
      roster is `scripts/bojjaes.csv`, and `scripts/players` is a *directory*. The script's no-arg
      form is broken today. Adjacent to this change rather than caused by it, but it sits in the
      same lines 8.1 is rewriting.
- [ ] 8.3 Update the stale doc comments this change falsifies:
      - `internal/score/calc.go` — `Points`' comment says "kicking and defensive scoring are not"
        implemented. It becomes false the moment section 2 lands.
      - `openspec/specs/player-week-score/spec.md` — the capability's Purpose paragraph says kicking
        and defensive scoring "are not yet specified". Archiving applies the requirement deltas but
        does not rewrite Purpose prose, so it needs an explicit edit.
- [ ] 8.4 Run `go test ./...` and `go build ./...` clean.

## 9. Deferred

- [ ] 9.1 Test: 1 field goal made alongside 2 missed field goals and a missed extra point scores 3,
      asserting that misses were never given a term. Moved here from 2.5, unresolved: it was written
      as a calculator test, but `StatLine` has no missed-kick field and the design forbids adding
      one (`design.md`, "Exclusions are asserted by test, not just omitted") — so there is no input
      a calc test can set to express "missed 2". It is the same category as the forced-fumble and
      safety exclusions, which section 5 pushes to 6.15 as transform tests over the payload's
      `fgmiss` / `xpmiss` keys. Decide the level before writing it; the kicking requirement's
      "Misses are not penalised" scenario is unasserted until then.
