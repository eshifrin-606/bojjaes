# How the HMFFL League Is Scored

These are the league's rules, as the commissioners run them — the spreadsheet's behavior, not this
codebase's. They are the target the code aims at, and they stay true whether or not any of it is
built yet.

For what the code actually scores today, read
[`openspec/specs/player-week-score/spec.md`](../openspec/specs/player-week-score/spec.md). Where the
two disagree, this document is right and the code is behind.

### Passing

| Action                       | Points |
| ----------------------------- | -----: |
| TD Pass                       |      6 |
| 2XP Pass                      |      2 |
| 200 passing yards             |      3 |
| Each additional 50 yards      |      1 |
| Interception thrown           |     -3 |

### Rushing / Receiving

| Action                                          | Points |
| ------------------------------------------------ | -----: |
| 80 yds rushing OR receiving, or 100 combined      |      3 |
| Each additional 10 yards                          |    0.5 |

### Scoring

| Action                       | Points |
| ----------------------------- | -----: |
| TD scored                     |      6 |
| Field Goal                    |      3 |
| 2XP scored                    |      2 |
| XP                             |      1 |
| Safety (solo credit only)     |      2 |

### Defense

| Action                                    | Points |
| ------------------------------------------ | -----: |
| Interception                                |      6 |
| Forced fumble that results in turnover      |      4 |
| Fumble recovery that results in turnover    |      2 |
| Sack                                        |      3 |
| 0.5 sack                                    |    1.5 |

### Misc

| Action                                          | Points |
| ------------------------------------------------ | -----: |
| Fumble lost (turnover)                            |     -3 |
| Bonus (any TD play of 40+ yards or FG 50+)        |      1 |

### Clarifications

Confirmed 2026-08-08. These resolve ambiguities in the tables above.

- **Yardage thresholds are floor-based off the threshold.** 249 passing yards = 3 pts;
  250 = 4 pts. Same shape for the rushing/receiving 10-yard increments.
- **The rush/rec bonus is awarded once.** A player with 80 rushing *and* 80 receiving gets a
  single 3-pt award (they qualify under the 100-combined clause), not two.
- **"TD scored" is unqualified** — it includes kick/punt return TDs and defensive TDs. A
  defender's pick-six is 6 (interception) + 6 (TD) = **12**, plus the 40+ bonus if the return
  was 40+ yards (13).
- **The 40+ yard TD bonus pays every player credited on the play.** A 45-yard TD pass is +1 to
  the QB *and* +1 to the receiver.
- **"40+ yards" is inclusive.** Confirmed 2026-08-09. A TD play of exactly 40 yards earns the
  bonus.

## Planned implementation deviations

**None of these is implemented yet.** They describe how the MVP scoreboard will knowingly diverge
from the rules above while it scores kicking and defense from Sleeper's REST weekly aggregate alone.
None is a rules change; the rules above stand regardless.

These deviations are **scoped to the aggregate-only stage**, not permanent positions. Per the
2026-08-12 amendment to [ADR 0003](adr/0003-sleeper-as-initial-stat-provider.md), all three are
computable from Sleeper's GraphQL play-by-play surface. They are deferred because opening that
surface is a second provider path with its own risks, not because the rules are unreachable. Each
one lifts when play-by-play lands.

- **Safety will not be scored at all.** The aggregate carries a per-player `idp_safe`, but nothing
  in it confirms the "solo credit only" qualifier, so we omit the 2 points rather than risk awarding
  them on shared credit. Play-by-play decides solo vs. assisted per play.
- **The 40+ bonus will not be applied to defensive or return TDs.** It applies to passing, rushing,
  and receiving TDs only, where the provider buckets at exactly our threshold. Per-player defensive
  return yardage *is* present in the aggregate, but as a **weekly sum** — a defender with two
  interception returns has no way to attribute distance to the one that scored. The obstacle is
  aggregation, not absence.
- **Forced fumbles will not be scored at all.** The rule pays only when the fumble results in a
  turnover, and turnover qualification is a property of the play, not of any player's aggregate stat
  line — so no aggregate key can carry it. Paying the raw count would overpay roughly 44% of forced
  fumbles, a systematic one-directional disagreement with the spreadsheet. An earlier plan was to pay
  it and flag the award as provisional; we prefer a visible omission to a knowingly wrong award,
  consistent with how safety is handled above. The 4 points arrive with play-by-play, which decides
  turnover qualification per play.

### Attribution hazards

Not deviations — the rules above are scored correctly — but the provider puts several
plausible-looking stats on the wrong player. Recorded here because getting one wrong produces a
large, confident, wrong number on a memorable play. Verified against 2025 week 14.

- A **pick-six pays the defender 12** (6 interception + 6 TD) and costs the passer 3. The provider
  also credits the *passer* with an "interception returned for a touchdown" stat. It must not feed the
  TD rule, or the quarterback is paid 6 for throwing one.
- Likewise **sacks taken** by a quarterback and **sacks recorded** by a defender are separate stats.
  Only the defender's pays 3.
- **Kick-return, punt-return, and defensive touchdowns appear on team rows**, which the provider
  mixes in with players. Only the individual special-teams touchdown stat pays a rostered player.
- **Forced fumbles are credited to opposite players in the aggregate and the play-by-play feeds** —
  the forcer in one, the fumbler in the other. See ADR 0003.

### Open rules questions

These need a commissioner ruling, not more data:

- **Own-team fumble recovery after an interception return.** The provider credits a recovery; the
  interception was already the turnover. Does the 2 points pay?
- **Shared-credit safety.** The solo/assisted distinction rests on two clean observations. Confirm
  against a safety with shared credit before the exclusion above is lifted.
- **Is the provider's 40+ touchdown bucket inclusive at exactly 40?** Our rule is inclusive
  (confirmed 2026-08-09); the provider's boundary is unverified. This affects the already-shipped
  passing, rushing, and receiving path, not only defense.
