## MODIFIED Requirements

### Requirement: Plays are polled cheaply and gaps trigger a whole-week fetch

Each time a week's stats are fetched, the system SHALL poll the provider's recent-plays endpoint for
that season and week with a bounded limit (about 300). It SHALL merge the result into the store.

The system SHALL start a whole-week fetch of the week's plays when either of these holds:

- the store holds no plays for the week, or
- the oldest play returned by the poll is newer than the newest play held, which leaves a gap.

The whole-week fetch SHALL run in the background, at most one at a time per week. Its result SHALL
be merged into the store on the same terms as a poll. Only a fetch started for a week whose store
held no plays SHALL be waited on, and only under the cold-week wait below.

Once a week's whole-week fetch has succeeded, the system SHALL NOT fetch that week whole again during
the same process, except to fill a gap or under the post-game refresh below. For a past week, this
means one whole-week fetch per deployment.

#### Scenario: First read of a week seeds it in the background

- **WHEN** a week's stats are fetched, the store holds no plays for it, and the cold-week wait is
  zero
- **THEN** a whole-week fetch is started in the background, and the stats are returned without
  waiting for it

#### Scenario: A gap triggers a whole-week fetch

- **WHEN** the oldest play in a poll is newer than the newest play held for the week
- **THEN** a whole-week fetch is started in the background, and the stats are returned without
  waiting for it, whatever the cold-week wait

#### Scenario: An overlapping poll does not

- **WHEN** a poll's oldest play is no newer than the newest play held
- **THEN** no whole-week fetch is started

#### Scenario: Concurrent gaps share one whole-week fetch

- **WHEN** a whole-week fetch for a week is still running and another gap is detected for that week
- **THEN** no second whole-week fetch is started

### Requirement: Play-by-play never blocks or fails a page

Fetching a week's stats SHALL NOT wait on a whole-week play fetch, except under the cold-week wait
and never for longer than it allows. The recent-plays poll SHALL be bounded by a short timeout of
its own.

If play-by-play cannot be read, the system SHALL log the failure and compute forced fumbles from the
plays already held, which is zero when none are held. The weekly stats SHALL still be returned.

A forced-fumble award arriving minutes to an hour after the play is acceptable.

#### Scenario: Poll fails with nothing held

- **WHEN** the recent-plays poll fails and the store holds no plays for the week
- **THEN** the weekly stats are returned, with forced fumbles credited only from a whole-week fetch
  that completed within the cold-week wait, and the failure is logged

#### Scenario: Poll fails with plays held

- **WHEN** the recent-plays poll fails and the store already holds plays for the week
- **THEN** forced fumbles are computed from the held plays, and the failure is logged

#### Scenario: Whole-week fetch fails

- **WHEN** a background whole-week fetch fails
- **THEN** the failure is logged, the store keeps what it held, and a later read may start another
  whole-week fetch

## ADDED Requirements

### Requirement: A cold week waits a bounded time for its whole-week fetch

When a week's stats are fetched and the store held no plays for that week before the poll, the
system SHALL wait for that week's whole-week fetch for up to the configured cold-week wait before
computing forced fumbles. If a whole-week fetch for the week is already running, the system SHALL
wait on that fetch and SHALL NOT start another.

If the whole-week fetch completes within the wait, forced fumbles SHALL be computed from the store
including its plays. If the wait runs out, or the caller's context ends first, forced fumbles SHALL
be computed from what is held, and the whole-week fetch SHALL continue in the background as it does
without a wait. A wait of zero SHALL NOT wait at all.

#### Scenario: Whole-week fetch completes within the wait

- **WHEN** a cold week's stats are fetched with a nonzero wait, and the whole-week fetch completes
  before the wait runs out
- **THEN** forced fumbles from plays outside the poll are credited on that same fetch

#### Scenario: Wait runs out

- **WHEN** a cold week's whole-week fetch is still running when the wait runs out
- **THEN** the stats are returned with forced fumbles from the polled plays only, and the whole-week
  fetch's result is merged into the store when it completes

#### Scenario: Caller's context ends first

- **WHEN** the caller's context is done before both the whole-week fetch and the wait
- **THEN** the stats are returned with forced fumbles from what is held

#### Scenario: Whole-week fetch fails within the wait

- **WHEN** a cold week's whole-week fetch fails before the wait runs out
- **THEN** the stats are returned promptly with forced fumbles from the polled plays, and the failure
  is logged

### Requirement: The cold-week wait comes from the environment, with a local default

The cold-week wait SHALL be read from the `PLAYS_COLD_WAIT` environment variable as a Go duration
string. When the variable is unset or empty, the wait SHALL be `30s`. The deployed configuration
SHALL set it to `0`.

A value that does not parse as a duration, or that is negative, SHALL stop the server at startup with
an error naming the variable. The server SHALL NOT start with a guessed wait.

#### Scenario: Set to a duration

- **WHEN** `PLAYS_COLD_WAIT` holds `10s`
- **THEN** the cold-week wait is 10 seconds

#### Scenario: Set to zero

- **WHEN** `PLAYS_COLD_WAIT` holds `0`
- **THEN** the cold-week wait is zero, and a cold week does not wait

#### Scenario: Unset or empty

- **WHEN** `PLAYS_COLD_WAIT` is not present, or is the empty string
- **THEN** the cold-week wait is 30 seconds

#### Scenario: Malformed value

- **WHEN** `PLAYS_COLD_WAIT` holds `ten`, or `-5s`
- **THEN** the server does not start, and the error names `PLAYS_COLD_WAIT`
