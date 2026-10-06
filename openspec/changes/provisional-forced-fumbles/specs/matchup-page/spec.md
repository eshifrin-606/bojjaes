## MODIFIED Requirements

### Requirement: The page refreshes itself while its tab is visible

The served page SHALL re-fetch itself on a recurring interval of approximately five minutes, so a
page left open during a game does not go silently stale. A page served from provisional stats SHALL
use an interval of approximately fifteen seconds instead, so the completed forced-fumble counts
replace the provisional ones soon after they land.

The refresh SHALL be conditioned on the tab being visible at the moment it would fire. While the
tab is hidden the page SHALL make no refresh request, so upstream load is bounded by who is
actually looking rather than by how many tabs are open.

The refresh SHALL be a reload of the same URL. The page SHALL NOT fetch a fragment, swap markup in
place, or hold a persistent connection.

#### Scenario: A visible tab refreshes on the interval

- **WHEN** the page has been open and visible for the refresh interval
- **THEN** the page re-requests its own URL and renders the newly fetched scores

#### Scenario: A hidden tab makes no request

- **WHEN** the page's tab is hidden and the refresh interval elapses
- **THEN** no request is made for the page and no stat fetch is triggered on its behalf

#### Scenario: An hour hidden is not an hour of requests

- **WHEN** the page's tab is hidden for an hour spanning several refresh intervals
- **THEN** the number of refresh requests made while it was hidden is zero, not one per interval

#### Scenario: A provisional page refreshes soon

- **WHEN** a page served from provisional stats has been open and visible for about fifteen seconds
- **THEN** the page re-requests its own URL

#### Scenario: A complete page keeps the five-minute interval

- **WHEN** a page is served from stats that are not provisional
- **THEN** its refresh interval is approximately five minutes

## ADDED Requirements

### Requirement: A provisional page says forced fumbles are still loading

A page served from provisional stats SHALL say, next to the statement of when its stats were
fetched, that forced fumbles are still loading. A page served from stats that are not provisional
SHALL NOT carry that note.

The note SHALL sit outside both columns and SHALL NOT name a team, a player, or a total.

The provisional status SHALL reach the refresh script through the rendered HTML, such as an
attribute on an element, and not through a value interpolated into the script element. The script
SHALL choose between two fixed intervals written in the script itself.

#### Scenario: A provisional page shows the note

- **WHEN** a page is served from provisional stats
- **THEN** the as-of line is accompanied by a note that forced fumbles are still loading

#### Scenario: A complete page shows no note

- **WHEN** a page is served from stats that are not provisional
- **THEN** the page contains no forced-fumbles-loading note

#### Scenario: The script stays free of page data

- **WHEN** a provisional page is rendered
- **THEN** the script element is identical to the one in a page that is not provisional
