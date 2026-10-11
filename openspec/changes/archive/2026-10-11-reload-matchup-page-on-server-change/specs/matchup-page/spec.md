## MODIFIED Requirements

### Requirement: The page refreshes itself while its tab is visible

The served page SHALL re-fetch itself on a recurring interval of approximately five minutes, so a
page left open during a game does not go silently stale.

The refresh SHALL be conditioned on the tab being visible at the moment it would fire. While the
tab is hidden the page SHALL make no refresh request, so upstream load is bounded by who is
actually looking rather than by how many tabs are open.

The refresh SHALL be a reload of the same URL. The page SHALL NOT fetch a fragment, swap markup in
place, or hold a persistent connection.

The version check defined below is the one other request the page may make. It is a short request
that returns no markup, and its only effect on the page is to cause the same reload of the same URL.

#### Scenario: A visible tab refreshes on the interval

- **WHEN** the page has been open and visible for the refresh interval
- **THEN** the page re-requests its own URL and renders the newly fetched scores

#### Scenario: A hidden tab makes no request

- **WHEN** the page's tab is hidden and the refresh interval elapses
- **THEN** no request is made for the page and no stat fetch is triggered on its behalf

#### Scenario: An hour hidden is not an hour of requests

- **WHEN** the page's tab is hidden for an hour spanning several refresh intervals
- **THEN** the number of refresh requests made while it was hidden is zero, not one per interval

#### Scenario: A version check swaps no markup

- **WHEN** a version check finds the server's version differs from the page's
- **THEN** the page reloads its own URL, and nothing from the check's response is written into the
  document

## ADDED Requirements

### Requirement: The page states the version of the stats it was rendered from

The page SHALL carry a stats version: a token that names the fetch the served stats came from. Two
pages rendered from the same fetch SHALL carry the same token, and pages rendered from different
fetches SHALL carry different tokens, including two fetches that complete within the same second.

The token SHALL be rendered once, as an attribute of an element outside the script. It is opaque to
the page: the script SHALL compare it for equality only and SHALL NOT parse it.

#### Scenario: The version names the fetch

- **WHEN** a page is served whose stats were fetched at a known instant
- **THEN** the page carries exactly one stats version, and it is the token for that instant

#### Scenario: A cache hit carries the original fetch's version

- **WHEN** two pages are served from the same cached fetch, minutes apart
- **THEN** both carry the same stats version

#### Scenario: Two fetches in one second have different versions

- **WHEN** two pages are served from two fetches that completed within the same second
- **THEN** their stats versions differ

### Requirement: The server reports a week's current stats version

`GET /{season}/{week}/version` SHALL report the stats version the server currently holds for that
season and week, in the same token form the page carries, so the two can be compared as strings.

When the server holds no completed fetch for that week, the endpoint SHALL report an empty version.
This covers a week never fetched, a week whose stats were invalidated, a week whose fetch is in
flight, and a server that has restarted.

Answering SHALL NOT call the stat provider, SHALL NOT start or wait on a fetch, and SHALL NOT read
the lineup tree. The response SHALL be marked as not to be stored by any cache.

The season and week SHALL be validated as the page's are: a segment that is not a number, or a
season or week out of range, SHALL be refused with status 400.

#### Scenario: The endpoint reports the version a fresh page carries

- **WHEN** a page for week 4 is served and the version endpoint for week 4 is then asked
- **THEN** the endpoint's version equals the page's stats version

#### Scenario: Invalidation empties the version

- **WHEN** week 4's stats are fetched and then invalidated, and the version endpoint is asked
- **THEN** the version reported is empty

#### Scenario: A refetch changes the version

- **WHEN** week 4's stats are fetched, invalidated, and fetched again, and the version endpoint is asked
- **THEN** the version reported is the second fetch's, and differs from the first page's

#### Scenario: Asking costs nothing upstream

- **WHEN** the version endpoint is asked for a week, whether or not that week has been fetched
- **THEN** no call is made to the stat provider

#### Scenario: A malformed week is refused

- **WHEN** `GET /2026/five/version` is requested
- **THEN** the response status is 400 and no version is reported

#### Scenario: The response is not cacheable

- **WHEN** the version endpoint answers
- **THEN** the response carries `Cache-Control: no-store`

### Requirement: The page reloads when the server's stats version differs from its own

While its tab is visible, the page SHALL check the version endpoint for its own season and week and
compare the reported version with the one it carries. When they differ, including when the reported
version is empty, the page SHALL reload its own URL.

The check SHALL be conditioned on the tab being visible at the moment it would fire. While the tab is
hidden the page SHALL make no check request.

A check that fails or cannot be read SHALL NOT reload the page. The page SHALL check again after the
delay it last used.

The reload SHALL be subject to every rule that binds the timed refresh: it compares no totals, marks
nothing as changed, and keeps no state across loads.

#### Scenario: A superseded page reloads

- **WHEN** a visible page carries one stats version and a check reports a different one
- **THEN** the page reloads its own URL

#### Scenario: An invalidated week reloads the page

- **WHEN** a visible page's week is invalidated on the server and the next check reports an empty
  version
- **THEN** the page reloads its own URL, and that reload's request fetches the week afresh

#### Scenario: A matching version does nothing

- **WHEN** a check reports the same stats version the page carries
- **THEN** the page does not reload

#### Scenario: A hidden tab makes no check

- **WHEN** the page's tab is hidden when a check would fire
- **THEN** no request is made to the version endpoint

#### Scenario: A failed check does not reload

- **WHEN** a check request fails
- **THEN** the page does not reload, and checks again after the same delay

### Requirement: The server directs how often the page checks

The delay before each check SHALL come from the server, not from a constant in the script. The served
page SHALL state the delay before the first check, as an attribute outside the script, and every
version endpoint response SHALL state the delay before the next one.

A delay of zero SHALL mean stop: the page SHALL make no further check until it is next loaded. The
timed refresh is unaffected by a stop.

The server SHALL give the same delay, approximately ten seconds, on every page and every response.
Varying it by week or by time is left to a later change, which this requirement exists to allow
without changing the script.

#### Scenario: The page states its first delay

- **WHEN** a page is served
- **THEN** it carries the delay before its first check, and that delay is the server's default

#### Scenario: The endpoint states the next delay

- **WHEN** the version endpoint answers
- **THEN** the response carries the delay before the next check, and it is the server's default

#### Scenario: The page follows a changed delay

- **WHEN** a check response states a delay different from the one the page last used
- **THEN** the page waits the newly stated delay before its next check

#### Scenario: A zero delay stops the checks

- **WHEN** a check response states a delay of zero
- **THEN** the page makes no further check, and still reloads on its five-minute interval
