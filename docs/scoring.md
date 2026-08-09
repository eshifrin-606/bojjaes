# Scoring

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

## Implementation deviations

The rules above are the league's. These are places where the MVP scoreboard knowingly does not
implement them, per [ADR 0003](adr/0003-sleeper-as-initial-stat-provider.md). None is a rules
change.

- **Safety is not scored at all.** Our source cannot confirm the "solo credit only" qualifier, so
  we omit the 2 points rather than risk awarding them on shared credit.
- **The 40+ bonus is not applied to defensive or return TDs.** It is applied to passing, rushing,
  and receiving TDs only.
- **Forced fumbles are not turnover-qualified.** The 4 points are awarded on any forced fumble and
  flagged as provisional in the output; the true rule pays only when the fumble results in a
  turnover.
