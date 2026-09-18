## ADDED Requirements

### Requirement: A season alone redirects to its latest week

The server SHALL serve `GET /{season}`, where the segment is a decimal integer, by redirecting to
`/{season}/{week}` for the greatest week the lineup tree holds a directory for in that season, per
`lineup-source`'s latest-week answer. The redirect SHALL NOT depend on that week being a
well-formed matchup: the same "exists, not resolved" rule the week bar's neighbour links use.

A segment that is not an integer, or a season outside the plausible range `internal/score`
validates, SHALL be refused with `400 Bad Request` and SHALL NOT reach the lineup tree.

A season with no week directories in the tree SHALL be refused with `404 Not Found`.

Resolving the redirect SHALL read only the lineup tree's directory listing. It SHALL NOT open a
roster file and SHALL NOT call the stats provider.

#### Scenario: A season redirects to its highest week

- **WHEN** the tree holds 2026 weeks 1 and 2, and `GET /2026` is requested
- **THEN** the response redirects to `/2026/2`

#### Scenario: A gap does not change the target

- **WHEN** the tree holds 2026 weeks 1 and 3 but not week 2, and `GET /2026` is requested
- **THEN** the response redirects to `/2026/3`

#### Scenario: A non-numeric segment is refused

- **WHEN** `GET /twentytwentysix` is requested
- **THEN** the response is `400 Bad Request` and no directory is read

#### Scenario: An out-of-range season is refused

- **WHEN** `GET /1899` is requested
- **THEN** the response is `400 Bad Request`

#### Scenario: A season with no weeks is a 404

- **WHEN** the tree holds no directories under 2027, and `GET /2027` is requested
- **THEN** the response is `404 Not Found`

#### Scenario: A season whose only week is not a matchup still redirects

- **WHEN** the tree's `2026/3/` directory holds three rosters and no higher week exists under 2026
- **THEN** `GET /2026` redirects to `/2026/3`, and `/2026/3` itself is served (or refused) by the
  existing season/week handler

#### Scenario: Resolving the redirect makes no upstream fetch

- **WHEN** `GET /2026` is requested
- **THEN** the stats provider is not called
