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

- [ ] 2.1 Test: 3 field goals made, none from 50+, scores 9. Fails — no field goal stat.
- [ ] 2.2 Add `FGMade` and its `3 ×` term. Confirm green, and confirm distance affects nothing.
- [ ] 2.3 Test: 4 extra points made scores 4. Fails. Add `XPMade` and its `1 ×` term.
- [ ] 2.4 Test: 2 field goals, one from 50+, scores 7. Fails — the bonus is unimplemented. Add
      `FG50Plus` and its `1 ×` bonus term.
- [ ] 2.5 Test: 1 field goal made alongside 2 missed field goals and a missed extra point scores 3.
      Should pass without new production code; it asserts that misses were never given a term.

## 3. Defensive points

- [ ] 3.1 Test: 2 interceptions caught scores 12. Fails — interceptions caught do not exist, and
      must not be confused with `PassInt`, which is a -3 penalty.
- [ ] 3.2 Add `IntCaught` and its `6 ×` term. Confirm green.
- [ ] 3.3 Test: a stat line carrying 3 interceptions *thrown* still scores -9, and a stat line
      carrying 3 interceptions *caught* scores 18. Confirms the two stats did not get crossed.
- [ ] 3.4 Test: 1 fumble recovery resulting in a turnover scores 2. Fails. Add `FumRec` and its
      `2 ×` term.

## 4. Finish the unqualified touchdown rule

- [ ] 4.1 Test: a defensive touchdown alone scores 6. Fails — the touchdown term covers only
      passing, rushing, and receiving.
- [ ] 4.2 Add `DefTD` to the existing 6-point touchdown term rather than a new term, keeping the
      rule in one place. Confirm green.
- [ ] 4.3 Test: an interception caught and returned for a touchdown scores 12. Should pass from
      3.2 and 4.2 together; it asserts the two rules compose as `docs/scoring.md` says.
- [ ] 4.4 Test: a receiver with 40 receiving yards and a punt-return touchdown is paid 6 for the
      return. Fails — return touchdowns are unmapped. Add `ReturnTD` to the same term.
- [ ] 4.5 Test: the 40+ yard bonus still applies to a 40+ receiving touchdown. Regression guard on
      the existing offensive path.

## 5. Assert the deliberate exclusions

These should pass without production changes. They exist so that a later contributor who removes an
exclusion breaks a test that explains why it was there.

The forced-fumble and safety exclusions are **not** calculator tests. Neither stat reaches
`StatLine` at all, so there is no field to set — a calc test for them could only exist by adding the
very field the exclusion is meant to prevent. They are asserted at the payload level in 6.12.

- [ ] 5.1 Test: an interception returned 63 yards for a touchdown scores 12, not 13 — no 40+ bonus
      on defensive touchdowns at this stage. This one *is* a calc test: both stats it needs
      (`IntCaught`, `DefTD`) exist, and the excluded thing is a term rather than a field.

## 6. Transform: map the new stats

- [ ] 6.1 Test: a payload entry carrying `fgm`, `fgm_50p`, and `xpm` populates each kicking field
      distinctly, so a mistyped key cannot read as zero. Fails — unmapped.
- [ ] 6.2 Map the kicking keys. Use `fgm_50p` alone for the bonus; it equals `fgm_50_59 + fgm_60p`
      with zero mismatches across the verified week, so summing is redundant.
- [ ] 6.3 Test: a payload entry crediting `idp_sack: 0.5` yields 0.5 on the stat line, not 0.
      Fails — the existing reader truncates through `int`.
- [ ] 6.4 Add a `statFloat` reader beside `stat` and map `idp_sack` through it.
- [ ] 6.5 Test: `idp_int` and `idp_def_td` populate their fields, and `pass_int` still maps to the
      passer's penalty. Map the defensive keys. Use `idp_int`, not the generic `int`: the week-14
      pick-six defender was observed carrying `idp_int: 1` alongside `idp_def_td: 1`, and that is
      the only interception-caught key this project has verified on a real row. `int` is
      unverified here — the existing comment in `sleeper.go` names it from Sleeper's documentation,
      not from observation.
- [ ] 6.6 Test: a payload entry carrying `pass_int_td` — a quarterback who threw a pick-six — scores
      -3, not +3 or +6. **This is the change's most expensive possible mistake.** This is a guard,
      not a red-green step: it passes on correct code because the key is never mapped. Verify it has
      teeth by *temporarily* mapping `pass_int_td` into the touchdown term, watching the test fail,
      then reverting the mapping. Do not commit the temporary mapping.
- [ ] 6.7 Test: a payload entry carrying `pass_sack` contributes nothing to the sack count. Confirms
      sacks taken were not crossed with sacks recorded. Same shape as 6.6 — verify by temporarily
      mapping `pass_sack` into `Sack`, then revert.
- [ ] 6.8 Comment the exclusions at the mapping site — `pass_int_td`, `pass_sack`, `fum_rec`,
      `kr_td`, `pr_td`, `def_td`, `td`, `anytime_tds` — with the row each sits on. Note that
      `anytime_tds` is not a shortcut for the rule: it omits defensive touchdowns. `fum_rec` needs
      the comment most after `pass_int_td`: it is the obvious-looking name for "fumble recovery" and
      is the wrong key — it holds **own-team** recoveries, which are not turnovers, and excluding it
      is precisely what leaves `idp_fum_rec` turnover-qualified.
- [ ] 6.9 Test: `st_td` populates the return touchdown field. Map it.
- [ ] 6.10 Test: `idp_fum_rec`, `st_fum_rec`, and `def_st_fum_rec` sum into one recovery count. Map
      the three-key sum.
- [ ] 6.11 Test: a payload entry carrying only `idp_fum_rec`, with the two special-teams keys absent
      entirely, reads as that count rather than erroring. These keys are sparse and absent from most
      weeks.
- [ ] 6.12 Test: a payload entry carrying `idp_ff`, `idp_safe`, `idp_int_ret_yd`, and
      `idp_fum_ret_yd` and nothing else scores **0**, and none of those values appears on any
      stat-line field. This is where the forced-fumble and safety exclusions from section 5 are
      actually asserted, at the only level where they are expressible. Comment the test with why
      each is excluded — the turnover-qualification rule and its ~44% overpay for `idp_ff`, the
      unconfirmed solo-credit qualifier for `idp_safe`. Guard, not red-green: verify it has teeth by
      temporarily mapping one key, then revert.

## 7. Fixtures

- [ ] 7.1 Extend `internal/score/testdata/week14.json` with real entries for a kicker, a defender
      with fractional sacks, the week-14 pick-six defender, and the quarterback who threw it.
      Real payload shapes, so the fixture cannot drift from the provider's actual key set.
- [ ] 7.2 Hand-build the special-teams fumble-recovery entry. Week 14 contains zero occurrences of
      `st_fum_rec` and `def_st_fum_rec`, so a captured fixture leaves 6.10 untested while green.
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
