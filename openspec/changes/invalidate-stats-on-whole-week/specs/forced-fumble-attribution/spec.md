## ADDED Requirements

### Requirement: A whole-week fetch that changes plays invalidates the week's cached stats

When a whole-week fetch for a season and week completes and its merge added a play to the store or
replaced a held play, the system SHALL invalidate that season and week's cached stats. The next read
of that week SHALL then be scored from the store's updated contents instead of serving a result
scored before the fetch landed.

A whole-week fetch that fails, or whose merge changed no play, SHALL NOT invalidate anything. This
applies to every reason a whole-week fetch is started: a cold week, a gap, and the post-game refresh.

#### Scenario: A cold week's late whole-week fetch is visible on the next read

- **WHEN** a week is read cold, the cold-week wait runs out before the whole-week fetch completes,
  and the whole-week fetch then adds a turnover forced fumble outside the poll
- **THEN** the next read of that week, within the cache TTL, credits that forced fumble

#### Scenario: A whole-week fetch that changes nothing does not invalidate

- **WHEN** a post-game refresh returns only plays already held, with no newer `updated_at`
- **THEN** the week's cached stats are not invalidated

#### Scenario: A failed whole-week fetch does not invalidate

- **WHEN** a whole-week fetch fails
- **THEN** the week's cached stats are not invalidated

#### Scenario: Poll merges do not invalidate

- **WHEN** a read's recent-plays poll adds plays to the store
- **THEN** that alone does not invalidate the week's cached stats
