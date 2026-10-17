## MODIFIED Requirements

### Requirement: Attribution matches the official record

Attribution over the recorded 2026 week 2 and week 3 plays SHALL agree with every row of
`internal/sleeper/plays/testdata/ff-test-players.csv`. A row marked `ff_turnover=yes` SHALL credit
that Sleeper player with one turnover-qualified forced fumble on that play. A row marked `no` SHALL
credit no one from that play. Plays SHALL be joined to rows by quarter and game clock.

#### Scenario: Ground-truth rows reproduce

- **WHEN** attribution runs over the recorded week 2 and week 3 fixtures
- **THEN** each ground-truth row's player is credited exactly when its row says `yes`
