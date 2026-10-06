## MODIFIED Requirements

### Requirement: Plays are polled cheaply and gaps trigger a whole-week fetch

Each time a week's stats are fetched, the system SHALL poll the provider's recent-plays endpoint for
that season and week with a bounded limit (about 300). It SHALL merge the result into the store.

A successful poll that returns fewer plays than its limit holds every play of the week. It SHALL
count as a successful whole-week fetch completed at the time of the poll, and SHALL close any hole
the week had. No whole-week fetch SHALL be started for that read.

Otherwise, the week SHALL be marked as having a **hole** when either of these holds:

- the store holds no plays for the week, or
- the oldest play returned by the poll is newer than the newest play held, which leaves a gap.

A hole SHALL stay open until a whole-week fetch for the week succeeds. While a week has an open hole,
a read SHALL start a whole-week fetch unless one is already running for that week or the last one
failed less than about five minutes ago.

The whole-week fetch SHALL run in the background, at most one at a time per week. Its result SHALL
be merged into the store on the same terms as a poll.

Once a week's whole-week fetch has succeeded, the system SHALL NOT fetch that week whole again during
the same process, except to fill a hole or under the post-game refresh below. For a past week, this
means one whole-week fetch per deployment.

#### Scenario: First read of a week seeds it in the background

- **WHEN** a week's stats are fetched, the store holds no plays for it, and the poll returns a full
  limit of plays
- **THEN** a whole-week fetch is started in the background, and the stats are returned without
  waiting for it

#### Scenario: A short poll is the whole week

- **WHEN** a poll succeeds with fewer plays than its limit and the store held no plays for the week
- **THEN** no whole-week fetch is started, and the week counts as fetched whole at the time of the
  poll

#### Scenario: An empty poll of a future week starts no fetch

- **WHEN** a poll for a week with no plays yet succeeds and returns no plays
- **THEN** no whole-week fetch is started, on this read or the next

#### Scenario: A failed poll is not a short poll

- **WHEN** the poll fails and the store holds no plays for the week
- **THEN** the week has an open hole and a whole-week fetch is started

#### Scenario: A gap triggers a whole-week fetch

- **WHEN** the oldest play in a full-limit poll is newer than the newest play held for the week
- **THEN** a whole-week fetch is started in the background

#### Scenario: An overlapping poll does not

- **WHEN** a poll's oldest play is no newer than the newest play held and the week has no open hole
- **THEN** no whole-week fetch is started

#### Scenario: Concurrent gaps share one whole-week fetch

- **WHEN** a whole-week fetch for a week is still running and another gap is detected for that week
- **THEN** no second whole-week fetch is started

#### Scenario: A hole outlives a failed fetch

- **WHEN** a whole-week fetch for a week with an open hole fails, and a read arrives more than five
  minutes later whose poll overlaps the held plays
- **THEN** a whole-week fetch is started, because the hole is still open

#### Scenario: A failed fetch is not retried at once

- **WHEN** a whole-week fetch for a week with an open hole failed less than five minutes ago and a
  read arrives
- **THEN** no whole-week fetch is started by that read

### Requirement: Play-by-play never blocks or fails a page

Fetching a week's stats SHALL NOT wait on a whole-week play fetch. The recent-plays poll SHALL be
bounded by a short timeout of its own.

If play-by-play cannot be read, the system SHALL log the failure and compute forced fumbles from the
plays already held, which is zero when none are held. The weekly stats SHALL still be returned.

A forced-fumble award arriving minutes to an hour after the play is acceptable.

#### Scenario: Poll fails with nothing held

- **WHEN** the recent-plays poll fails and the store holds no plays for the week
- **THEN** the weekly stats are returned with no forced fumbles credited, and the failure is logged

#### Scenario: Poll fails with plays held

- **WHEN** the recent-plays poll fails and the store already holds plays for the week
- **THEN** forced fumbles are computed from the held plays, and the failure is logged

#### Scenario: Whole-week fetch fails

- **WHEN** a background whole-week fetch fails
- **THEN** the failure is logged, the store keeps what it held, any open hole stays open, and a read
  about five minutes later or more may start another whole-week fetch

## ADDED Requirements

### Requirement: A count taken while a hole is being filled is provisional

The forced-fumble counts for a week SHALL be reported as **provisional** when, at the moment they
are computed, the week has an open hole and a whole-week fetch for it is running. They SHALL NOT be
provisional otherwise. In particular, a post-game refresh in flight for a week with no open hole does
not make the counts provisional, and neither does an open hole whose last fetch failed and has not
been retried.

The weekly stats built from provisional counts SHALL carry the provisional status. The stats
themselves SHALL be computed exactly as for non-provisional counts, from the plays held.

#### Scenario: A cold read is provisional

- **WHEN** a week's stats are fetched with no plays held and a full-limit poll, so a whole-week fetch
  starts
- **THEN** the returned stats are provisional

#### Scenario: A read after the fetch lands is not provisional

- **WHEN** a whole-week fetch that filled a hole has succeeded and the week is read again
- **THEN** the returned stats are not provisional

#### Scenario: A read during an in-flight hole fill is provisional

- **WHEN** a whole-week fetch for an open hole is still running and another read's poll overlaps the
  held plays
- **THEN** that read's stats are provisional

#### Scenario: A short poll is never provisional

- **WHEN** a poll returns fewer plays than its limit
- **THEN** the returned stats are not provisional

#### Scenario: A post-game refresh is not provisional

- **WHEN** a post-game whole-week fetch is running for a week with no open hole
- **THEN** the returned stats are not provisional

#### Scenario: A failed hole fill is not provisional until retried

- **WHEN** the whole-week fetch for an open hole has failed and a read arrives before a retry is
  allowed
- **THEN** the returned stats are not provisional
