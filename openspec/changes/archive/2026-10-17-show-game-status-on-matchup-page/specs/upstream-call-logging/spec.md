## ADDED Requirements

### Requirement: Successful game fetches are logged

When a week's games are fetched successfully, the system SHALL log one line naming the season, the
week, the number of games and the fetch duration.

#### Scenario: Games fetched

- **WHEN** a week's games are fetched from Sleeper and the call succeeds
- **THEN** a line `sleeper games {season} w{week}: {n} games in {duration}` is logged

### Requirement: Game cache hits are logged with their age

When a request for a week's games is served from a fresh cache entry, the system SHALL log one line
naming the season, the week and how long ago that entry was fetched. The line SHALL be told apart
from a stats cache hit by its prefix.

#### Scenario: Fresh games entry served

- **WHEN** a week's games are requested and the cache holds an entry for that week younger than the
  TTL
- **THEN** a line `gamescache hit: {season} week {week}, {age} old` is logged

#### Scenario: A stats hit keeps its own prefix

- **WHEN** a week's stats are served from a fresh cache entry
- **THEN** the line logged begins `statscache hit:` and no `gamescache` line is logged for it
