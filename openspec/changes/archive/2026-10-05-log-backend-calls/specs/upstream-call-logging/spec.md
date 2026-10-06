## ADDED Requirements

### Requirement: Successful aggregate fetches are logged

When the weekly stats aggregate is fetched successfully, the system SHALL log one line naming the season, the week, the number of rows and the fetch duration.

#### Scenario: Aggregate fetched

- **WHEN** a week's stats are fetched from Sleeper and the aggregate call succeeds
- **THEN** a line `sleeper stats {season} w{week}: {rows} rows in {duration}` is logged

### Requirement: Successful play polls are logged

When the recent-plays poll succeeds, the system SHALL log one line naming the season, the week, the number of plays returned and the poll duration.

#### Scenario: Poll succeeds

- **WHEN** the recent-plays poll for a week returns
- **THEN** a line `sleeper plays poll {season} w{week}: {n} plays in {duration}` is logged

### Requirement: Whole-week fetch starts are logged with a reason

When a read starts a whole-week play fetch, the system SHALL log the season, the week and the reason. If the store held no plays for the week, the reason SHALL be `cold`. If it held plays and the polled plays left a gap, the reason SHALL be `gap`. If neither applies and the week has gone quiet since its last whole fetch, the reason SHALL be `postgame`. A read that joins a whole-week fetch already running SHALL NOT log a start.

#### Scenario: Cold week

- **WHEN** the store holds no plays for a week and a read starts a whole-week fetch
- **THEN** a line `sleeper whole-week {season} w{week}: started (cold)` is logged

#### Scenario: Gap

- **WHEN** the store holds plays and the oldest polled play is newer than the newest held play
- **THEN** a line `sleeper whole-week {season} w{week}: started (gap)` is logged

#### Scenario: Post-game refresh

- **WHEN** a week has gone quiet with no whole fetch since, and there is no cold store or gap
- **THEN** a line `sleeper whole-week {season} w{week}: started (postgame)` is logged

#### Scenario: Joining a running fetch

- **WHEN** a whole-week fetch for the week is already in flight
- **THEN** no start line is logged

### Requirement: Successful whole-week fetches are logged

When a whole-week play fetch succeeds, the system SHALL log one line naming the season, the week, the number of plays returned and the fetch duration.

#### Scenario: Whole-week fetch succeeds

- **WHEN** a background whole-week fetch returns plays
- **THEN** a line `sleeper whole-week {season} w{week}: {n} plays in {duration}` is logged
