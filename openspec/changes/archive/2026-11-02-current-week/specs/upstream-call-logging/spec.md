## ADDED Requirements

### Requirement: Successful state fetches are logged

When the current NFL week is fetched successfully, the system SHALL log one line naming the
season, the provider's phase text, the week, and the fetch duration.

#### Scenario: State fetched

- **WHEN** the current week is fetched from Sleeper and the call succeeds
- **THEN** a line `sleeper state: {season} {phase} w{week} in {duration}` is logged

### Requirement: State cache hits and misses are logged without a season or week

When the current week is served from a fresh cache entry, the system SHALL log one line naming how
long ago that entry was fetched. When it is fetched because no fresh entry is held, the system
SHALL log one miss line. Neither line SHALL name a season and week, since the cache is not keyed by
one. Lines from the weekly caches SHALL keep their existing form.

#### Scenario: Fresh state entry served

- **WHEN** the current week is requested and a fetched answer younger than an hour is held
- **THEN** a line `nflstate hit: current, {age} old` is logged

#### Scenario: State fetched on a miss

- **WHEN** the current week is requested and no fresh answer is held
- **THEN** a line `nflstate miss: current fetched in {duration}` is logged

#### Scenario: A weekly cache keeps its form

- **WHEN** a week's stats are served from a fresh cache entry
- **THEN** the line logged is `statscache hit: {season} week {week}, {age} old`
