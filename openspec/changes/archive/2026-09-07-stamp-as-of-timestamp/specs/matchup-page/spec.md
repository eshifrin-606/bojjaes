## ADDED Requirements

### Requirement: The page states when its stats were fetched

The page SHALL show the instant its weekly stats were fetched from the provider, once, above or
below both columns and never inside either. The instant SHALL be the time the fetch that produced
the served stats completed — so a page served from cache states the original fetch, not the time of
the request it answered.

The instant SHALL be rendered both machine-readably and human-readably: a `<time>` element whose
`datetime` attribute carries the fetch instant as an RFC 3339 string, with visible text showing
that instant as a wall-clock time in `America/Chicago`, the league's timezone.

The visible text SHALL be labelled so a reader reads it as when the server fetched the data, not as
the current time and not as when the upstream stats last changed. The label SHALL NOT claim the
page is "live" or "current".

The rendering SHALL carry no relative phrasing ("4 minutes ago"); that is left to a later change,
which the machine-readable `datetime` attribute exists to enable.

#### Scenario: The fetch instant appears once, machine- and human-readable

- **WHEN** a page is served whose stats were fetched at a known instant
- **THEN** the page contains exactly one `<time>` element whose `datetime` is that instant in RFC
  3339 and whose visible text is that instant as an `America/Chicago` wall-clock time

#### Scenario: A cache hit reports the fetch, not the request

- **WHEN** the stats are served from a cache entry whose fetch completed three minutes before this
  request
- **THEN** the stated instant is that fetch's completion time, three minutes ago, not the current
  request time

#### Scenario: The label does not overclaim currency

- **WHEN** the page renders its as-of line
- **THEN** the surrounding text identifies the instant as when the data was fetched and nowhere
  describes the page as live or current

#### Scenario: The as-of line implies no winner

- **WHEN** the two columns total 56 and 42 and the as-of line is rendered
- **THEN** the as-of line references neither total, sits outside both columns, and adds no margin,
  difference, or winner to the page
