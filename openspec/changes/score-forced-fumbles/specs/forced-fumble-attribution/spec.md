## ADDED Requirements

### Requirement: Only plays with a play-by-play forced fumble are considered

The system SHALL consider a play for forced-fumble attribution only when at least one row on that
play carries the provider's forced-fumble stat (`idp_ff > 0`).

In play-by-play, that stat sits on the **fumbler's** row. The weekly aggregate puts it on the
forcer's row instead. The row carrying it SHALL be treated as the fumbler and SHALL NOT be credited
with the forced fumble.

An overturned fumble, or a fumble on a play nullified by penalty, can keep its "forced by" text in
the description while carrying no forced-fumble stat. A play like that SHALL award nothing.

#### Scenario: The flagged row is the fumbler

- **WHEN** a play's only forced-fumble stat is on the ball carrier's row
- **THEN** the ball carrier is not credited with a forced fumble

#### Scenario: Forced-by text without the stat awards nothing

- **WHEN** a play's description says "FUMBLES, forced by" but no row carries the forced-fumble stat,
  as on a no-play or an overturned fumble
- **THEN** no player is credited with a forced fumble from that play

#### Scenario: Team rows never take part

- **WHEN** a play includes rows whose player ID is not numeric, such as `CAR`
- **THEN** those rows are neither fumblers nor forcers

### Requirement: Turnover status is judged per fumble

A forced fumble SHALL count only when that fumble was a turnover. The system SHALL decide this from
the fumbler's own row on that play: the fumble is a turnover when that row's fumbles-lost stat is
greater than zero. Turnover status SHALL NOT be inferred from other rows or from the play as a whole.

#### Scenario: A lost fumble counts

- **WHEN** the fumbler's row carries a positive fumbles-lost stat and the forcer is resolved
- **THEN** the forcer is credited with one turnover-qualified forced fumble

#### Scenario: A fumble that went out of bounds does not count

- **WHEN** the fumbler's row carries a forced-fumble stat but no fumbles-lost stat, as on Jihaad
  Campbell's 2026 week 2 forced fumble that went out of bounds
- **THEN** no one is credited from that fumble

#### Scenario: Two fumbles on one play are judged separately

- **WHEN** one play holds two fumbles and only one of them was lost, as in 2026 week 3 HOU@IND Q1
  11:28: Will Anderson forced D.Jones' lost fumble, then Anderson fumbled, forced by Alie-Cox, and the
  ball was not lost
- **THEN** Anderson is credited with one turnover-qualified forced fumble and Alie-Cox with none

### Requirement: The forcer is resolved from the play description

For each fumbler, the system SHALL find the forcer by reading `<F.Last> FUMBLES, forced by <F.Last>`
pairs from the play description. It SHALL read only the text after the last occurrence of
"overturned", compared case-insensitively. Text before that point describes a play that no longer
stands.

The pair SHALL be selected by the fumbler's abbreviated name, built from first initial, a period, and
last name. The forcer SHALL be the row on the same play whose abbreviated name matches the pair's
forcer and whose team is not the fumbler's team. Every row on the play SHALL be searched, including
rows with no stats, because the forcer's row can be empty. Generational suffixes (`Jr.`, `Sr.`,
`II`, `III`, `IV`, `V`) SHALL be ignored on both sides of the comparison.

The sacker is not necessarily the forcer. Credit SHALL follow the "forced by" name, not the sack.

#### Scenario: Forcer row with empty stats

- **WHEN** the forcer's row on the play carries no stats, as for Maxx Crosby in 2026 week 3 LV@NO Q4
  2:21 on a lost fumble
- **THEN** Crosby is credited with one turnover-qualified forced fumble

#### Scenario: Sacker and forcer differ

- **WHEN** one player is credited with the sack and the description names another as forcer, as in
  2026 week 3 HOU@IND Q4 4:45, where Anderson sacked and D.Hunter forced a lost fumble
- **THEN** D.Hunter is credited with the forced fumble and Anderson is not

#### Scenario: A reviewed play is read from its final version

- **WHEN** the description includes an overturned version of the play before the final version, and
  the forced fumble appears only in the final version
- **THEN** the forcer is resolved from the text after the last "overturned"

#### Scenario: Suffixes do not block a match

- **WHEN** the description names a player with a suffix, such as "K.Moore II", and the player's row
  carries the name without it
- **THEN** the names match

#### Scenario: A same-team namesake is not the forcer

- **WHEN** a player on the fumbler's team has the same abbreviated name as the forcer
- **THEN** that player is not credited

### Requirement: Unresolvable attributions award nothing and are logged

When a turnover fumble's forcer cannot be resolved to exactly one player, the system SHALL credit no
one for that fumble and SHALL log the play. This covers a fumbler with no matching description pair or
with more than one, and a forcer name matching no row on the opposing team or more than one.

#### Scenario: No matching pair

- **WHEN** a lost fumble's fumbler has no "FUMBLES, forced by" pair in the description
- **THEN** no one is credited and the play is logged

#### Scenario: Ambiguous forcer

- **WHEN** two opposing-team rows on the play match the forcer's abbreviated name
- **THEN** no one is credited and the play is logged

### Requirement: Attribution matches the official record

Attribution over the recorded 2026 week 2 and week 3 plays SHALL agree with every row of
`internal/sleeper/testdata/ff-test-players.csv`. A row marked `ff_turnover=yes` SHALL credit that
Sleeper player with one turnover-qualified forced fumble on that play. A row marked `no` SHALL credit
no one from that play. Plays SHALL be joined to rows by quarter and game clock.

#### Scenario: Ground-truth rows reproduce

- **WHEN** attribution runs over the recorded week 2 and week 3 fixtures
- **THEN** each ground-truth row's player is credited exactly when its row says `yes`

### Requirement: Plays are held per week, merged by play

The system SHALL hold each season and week's plays in a store keyed by play ID. Plays received later
SHALL be merged into the store. A received copy of a play SHALL replace the held copy when its
`updated_at` is newer, and SHALL be ignored otherwise.

Forced-fumble counts SHALL be computed from the store's current contents.

#### Scenario: A newer copy replaces the older

- **WHEN** a play already held is received again with a newer `updated_at` and different stats
- **THEN** the store holds the newer copy, and attribution reflects it

#### Scenario: A stale copy is ignored

- **WHEN** a play already held is received again with an older `updated_at`
- **THEN** the held copy is kept

### Requirement: Plays are polled cheaply and gaps trigger a whole-week fetch

Each time a week's stats are fetched, the system SHALL poll the provider's recent-plays endpoint for
that season and week with a bounded limit (about 300). It SHALL merge the result into the store.

The system SHALL start a whole-week fetch of the week's plays when either of these holds:

- the store holds no plays for the week, or
- the oldest play returned by the poll is newer than the newest play held, which leaves a gap.

The whole-week fetch SHALL run in the background, at most one at a time per week. Its result SHALL
be merged into the store on the same terms as a poll.

Once a week's whole-week fetch has succeeded, the system SHALL NOT fetch that week whole again during
the same process, except to fill a gap or under the post-game refresh below. For a past week, this
means one whole-week fetch per deployment.

#### Scenario: First read of a week seeds it in the background

- **WHEN** a week's stats are fetched and the store holds no plays for it
- **THEN** a whole-week fetch is started in the background, and the stats are returned without
  waiting for it

#### Scenario: A gap triggers a whole-week fetch

- **WHEN** the oldest play in a poll is newer than the newest play held for the week
- **THEN** a whole-week fetch is started in the background

#### Scenario: An overlapping poll does not

- **WHEN** a poll's oldest play is no newer than the newest play held
- **THEN** no whole-week fetch is started

#### Scenario: Concurrent gaps share one whole-week fetch

- **WHEN** a whole-week fetch for a week is still running and another gap is detected for that week
- **THEN** no second whole-week fetch is started

### Requirement: A week is refreshed whole once after it goes quiet

Recent-plays polling may not show corrections to old plays. To catch them, the system SHALL start one
background whole-week fetch for a week once no play has changed for an hour and no whole-week fetch
has completed since that last change. A week whose last whole-week fetch is newer than its last
change plus an hour SHALL NOT be refetched.

#### Scenario: A finished week is refreshed once

- **WHEN** a week's newest held change is more than an hour old and its last whole-week fetch
  predates that change
- **THEN** one background whole-week fetch is started

#### Scenario: A settled past week is not refetched

- **WHEN** a week's last whole-week fetch completed more than an hour after its newest held change
- **THEN** no whole-week fetch is started

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
- **THEN** the failure is logged, the store keeps what it held, and a later read may start another
  whole-week fetch
