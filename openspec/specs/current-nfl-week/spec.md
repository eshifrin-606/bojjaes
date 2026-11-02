# current-nfl-week Specification

## Purpose

Read the current NFL season, week, and phase from the provider, so a caller can tell which week a
lineup is set for, or that no week is settable. It governs the read, what counts as settable, and
how long the answer is cached. The league's own last week is a league rule and is not decided here.

## Requirements
### Requirement: The current NFL week is read from the provider in one call

The system SHALL read the current NFL season, week, and season phase from the provider with one
request that takes no season or week. The phase SHALL be one of preseason, regular season,
postseason, offseason, or unrecognised.

The provider's season SHALL be read as a number even though it is sent as text. The provider's
phases `pre`, `regular`, `post`, and `off` SHALL be preseason, regular season, postseason, and
offseason. Any other phase SHALL be unrecognised, not an error.

#### Scenario: A regular-season week is read

- **WHEN** the provider answers season `"2026"`, phase `regular`, week 9
- **THEN** the current week is season 2026, regular season, week 9

#### Scenario: The known phases

- **WHEN** the provider answers phase `pre`, `post`, or `off`
- **THEN** the phase is preseason, postseason, or offseason respectively

#### Scenario: An unrecognised phase is kept, not refused

- **WHEN** the provider answers phase `exhibition`
- **THEN** the read succeeds and the phase is unrecognised

### Requirement: Only a regular-season week is settable

The current week SHALL answer whether it has a settable week, and when it does, SHALL give that
season and week. A week SHALL be settable only when the phase is regular season and the week is
at least 1. The league's own last week is not this capability's concern. In every other case
the answer SHALL be that no week is settable.

#### Scenario: A mid-season week is settable

- **WHEN** the current week is season 2026, regular season, week 9
- **THEN** the settable week is season 2026 week 9

#### Scenario: The last NFL regular-season week is settable

- **WHEN** the current week is regular season, week 18
- **THEN** week 18 is settable

#### Scenario: Week zero is not settable

- **WHEN** the current week is regular season, week 0
- **THEN** no week is settable

#### Scenario: Outside the regular season nothing is settable

- **WHEN** the phase is preseason, postseason, offseason, or unrecognised, with week 1
- **THEN** no week is settable

### Requirement: A failed state read is an error

When the provider does not answer 200, its body cannot be read as the state object, or its season
is not a number, the read SHALL fail with an error. It SHALL NOT return a current week in place of
the error, so a failure is never mistaken for "no settable week".

#### Scenario: The provider answers with an error status

- **WHEN** the provider answers 503
- **THEN** the read fails with an error

#### Scenario: The body is not the state object

- **WHEN** the provider answers 200 with a JSON list
- **THEN** the read fails with an error

#### Scenario: The season is not a number

- **WHEN** the provider answers season `""`
- **THEN** the read fails with an error

### Requirement: The current week is served from cache for an hour

A successful read SHALL be served from memory for one hour, measured from when the fetch
completed; the next request after that SHALL fetch afresh. Concurrent requests while no fresh
answer is held SHALL share one upstream call. A failed read SHALL NOT be stored. Every caller SHALL
receive the instant the answer it is served was fetched.

#### Scenario: A second request within the hour makes no upstream call

- **WHEN** the current week is requested twice, ten minutes apart
- **THEN** the provider is called once and both callers get the same answer and fetch instant

#### Scenario: A request after the hour fetches again

- **WHEN** the current week is requested twice, sixty-one minutes apart
- **THEN** the provider is called twice

#### Scenario: A failure is not remembered

- **WHEN** the first request fails and a second follows a minute later
- **THEN** the second request calls the provider again

