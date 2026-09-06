## ADDED Requirements

### Requirement: The page refreshes itself while its tab is visible

The served page SHALL re-fetch itself on a recurring interval of approximately five minutes, so a
page left open during a game does not go silently stale.

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

### Requirement: Returning to a stale tab refreshes immediately

When the tab becomes visible again and at least the refresh interval has elapsed since the page
was last loaded, the page SHALL refresh immediately rather than waiting out the remainder of a
timer.

A reader returning to the tab is the moment the page is most likely to be stale and most likely to
be believed, so what they see on returning SHALL NOT be older than the refresh interval by more
than the time the reload itself takes.

#### Scenario: A tab hidden past the interval refreshes on return

- **WHEN** the tab has been hidden for longer than the refresh interval and becomes visible
- **THEN** the page refreshes at once

#### Scenario: A brief switch away does not refresh

- **WHEN** the tab is hidden and made visible again well within the refresh interval
- **THEN** the page does not refresh on return and the existing interval continues

### Requirement: The refresh script carries no page data

The refresh script SHALL contain no interpolated value from the rendered page. No team name, player
name, point value, season, or week SHALL be written into the script element.

Team and player names reach the page from file names and hand-edited CSV files. Keeping them out of
the script keeps every one of them in the HTML text contexts the existing escaping requirement
already covers, rather than in a JavaScript context.

#### Scenario: A roster name never reaches the script

- **WHEN** a page is rendered for a week whose rosters contain any team and player names
- **THEN** the script element in the response contains none of those names

## MODIFIED Requirements

### Requirement: The page implies no winner

The rendered markup, its styling, and any script it carries SHALL NOT contain a margin, a
difference between the two totals, a win probability, a progress bar, a leader highlight, or any
styling or behaviour that distinguishes the leading column from the trailing one.

A starter whose game has not kicked off is indistinguishable on this page from one who was
inactive, so any winner-implying element would present an unsettled reading as a result.

The refresh SHALL NOT create one by other means: it SHALL NOT compare a newly fetched total against
the one previously displayed, and SHALL NOT animate, highlight, or otherwise mark a total that has
changed since the last load.

#### Scenario: No margin appears

- **WHEN** the two columns total 56 and 42
- **THEN** the page shows both totals and nowhere shows their difference

#### Scenario: The leading column is styled like the trailing one

- **WHEN** one column's total is higher than the other's
- **THEN** the two columns carry the same markup and classes, with nothing marking either as ahead

#### Scenario: A refresh marks no change

- **WHEN** a refresh renders a total different from the one the previous load showed
- **THEN** the new page is rendered exactly as a first load of it would be, with nothing marking
  which values moved
