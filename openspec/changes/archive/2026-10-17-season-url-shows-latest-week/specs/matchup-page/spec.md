## ADDED Requirements

### Requirement: A season alone shows its latest week

The server SHALL serve `GET /{season}`, where the segment is a decimal integer, with the matchup page
for the greatest week the lineup tree holds a directory for in that season, per `lineup-source`'s
latest-week answer. The response SHALL be the page itself, not a redirect: its status and body SHALL
be what `GET /{season}/{week}` answers for that week, including that week's refusals when its
directory is not a well-formed matchup.

`GET /{season}/{week}` SHALL continue to serve every week, the latest included. Neither URL SHALL
redirect to the other.

A segment that is not an integer, or a season outside the plausible range `internal/score`
validates, SHALL be refused with `400 Bad Request` and SHALL NOT reach the lineup tree.

A season with no week directories in the tree SHALL be refused with `404 Not Found`.

The stats provider SHALL be asked for the latest week only.

#### Scenario: A season shows its highest week

- **WHEN** the tree holds 2026 weeks 1 and 2, and `GET /2026` is requested
- **THEN** the response is `200` with the page labelled `2026 · Week 2`, and it carries no
  `Location` header

#### Scenario: A gap does not change the week shown

- **WHEN** the tree holds 2026 weeks 1 and 3 but not week 2, and `GET /2026` is requested
- **THEN** the page shown is week 3's

#### Scenario: The latest week still has its numbered URL

- **WHEN** the tree holds 2026 weeks 1 and 2, and `GET /2026/2` is requested
- **THEN** the response is `200` with week 2's page, not a redirect

#### Scenario: A non-numeric segment is refused

- **WHEN** `GET /twentytwentysix` is requested
- **THEN** the response is `400 Bad Request` and no directory is read

#### Scenario: An out-of-range season is refused

- **WHEN** `GET /1899` is requested
- **THEN** the response is `400 Bad Request`

#### Scenario: A season with no weeks is a 404

- **WHEN** the tree holds no directories under 2027, and `GET /2027` is requested
- **THEN** the response is `404 Not Found`

#### Scenario: A latest week that is not a matchup is refused as its own URL is

- **WHEN** the tree's `2026/3/` directory holds three rosters and no higher week exists under 2026
- **THEN** `GET /2026` answers `500`, as `GET /2026/3` does

#### Scenario: Only the latest week is fetched

- **WHEN** the tree holds 2026 weeks 1 and 2, and `GET /2026` is requested
- **THEN** the stats provider is asked for 2026 week 2 exactly once and for no other week

## MODIFIED Requirements

### Requirement: The page names its week and links to the adjacent weeks of its season

The page SHALL show a week bar above both columns and outside either one. The bar SHALL hold a
label naming the season and week the page shows, in the form `2026 · Week 2`.

The bar SHALL hold a link to the previous week at its left when that week exists, and a link to the
next week at its right when that week exists. The previous week is the same season, week − 1. The
next week is the same season, week + 1. A link SHALL NOT cross into another season: week 1 has no
previous link, and a season's last week has no next link, even if the other season has lineups.
Whether a week exists SHALL be decided as `lineup-source` decides it, so a week whose directory is
not a well-formed matchup still gets a link.

Each link SHALL be an ordinary hyperlink. A link to the season's latest week, as `lineup-source`'s
latest-week answer names it, SHALL point to the season's page (`/{season}`); a link to any other
week SHALL point to that week's page (`/{season}/{week}`). The previous link SHALL carry
`rel="prev"` and the next link `rel="next"`. Each link's text SHALL name the week it leads to.

The bar SHALL be the same whichever URL served the page: `/{season}` and `/{season}/{week}` for the
same week show the same label and the same links.

Deciding which links to show and where they point SHALL NOT call the stats provider, and the page
SHALL still fetch only the week it shows.

When a link is absent, its space SHALL stay empty rather than collapse: the label's horizontal
position SHALL NOT depend on which links are present. The label and both links SHALL fit on one line
on a phone held in portrait. The bar SHALL be styled identically whatever either column's points
are.

#### Scenario: A middle week links both ways

- **WHEN** the tree holds 2026 weeks 1 through 4, and `GET /2026/2` is requested
- **THEN** the page's label reads `2026 · Week 2`, a `rel="prev"` link points to `/2026/1`, and a
  `rel="next"` link points to `/2026/3`

#### Scenario: The link to the latest week is the season URL

- **WHEN** the tree holds 2026 weeks 1, 2, and 3, and `GET /2026/2` is requested
- **THEN** the page's `rel="next"` link reads `Week 3 ›` and points to `/2026`

#### Scenario: The first week has no previous link

- **WHEN** the tree holds 2026 weeks 1 and 2, and `GET /2026/1` is requested
- **THEN** the page has a `rel="next"` link to `/2026` and no `rel="prev"` link

#### Scenario: The latest week has no next link

- **WHEN** the tree holds 2026 weeks 1 and 2, and `GET /2026/2` is requested
- **THEN** the page has a `rel="prev"` link to `/2026/1` and no `rel="next"` link

#### Scenario: The season page has the latest week's bar

- **WHEN** the tree holds 2026 weeks 1 and 2, and `GET /2026` is requested
- **THEN** the page's label reads `2026 · Week 2`, a `rel="prev"` link points to `/2026/1`, and
  there is no `rel="next"` link

#### Scenario: Links do not cross a season boundary

- **WHEN** the tree holds 2025 week 16 and 2026 week 1, and no 2025 week 17 or 2026 week 0
- **THEN** the 2025 week 16 page has no next link and the 2026 week 1 page has no previous link

#### Scenario: A gap in a season is not skipped

- **WHEN** the tree holds 2026 weeks 1 and 3 but not week 2, and `GET /2026/3` is requested
- **THEN** the page has no `rel="prev"` link

#### Scenario: A broken neighbouring week still gets a link

- **WHEN** the tree holds 2026 weeks 2, 3, and 4, its week 3 directory holds three rosters, and
  `GET /2026/2` is requested
- **THEN** the page has a `rel="next"` link to `/2026/3`

#### Scenario: Only the shown week is fetched

- **WHEN** `GET /2026/2` is served and weeks 1 and 3 both exist
- **THEN** the stats provider is asked for 2026 week 2 exactly once and for no other week

#### Scenario: The label does not move when a link is absent

- **WHEN** the page is viewed in a browser once with both links and once with only a next link
- **THEN** the label's horizontal position is the same in both

#### Scenario: A phone fits the week bar on one line

- **WHEN** the page is viewed on a viewport 390 CSS px wide with both links present
- **THEN** the previous link, the label, and the next link share one line and do not overlap

### Requirement: The server reports a week's current stats version

`GET /{season}/{week}/version` SHALL report the stats version the server currently holds for that
season and week, in the same token form the page carries, so the two can be compared as strings.

`GET /{season}/version` SHALL report the stats version the server currently holds for the season's
latest week, resolved on every request as `GET /{season}` resolves it. When a higher week's
directory appears, the next answer SHALL be for that week.

When the server holds no completed fetch for that week, the endpoint SHALL report an empty version.
This covers a week never fetched, a week whose stats were invalidated, a week whose fetch is in
flight, and a server that has restarted.

Answering SHALL NOT call the stat provider and SHALL NOT start or wait on a fetch. The week form
SHALL NOT read the lineup tree. The season form SHALL read only the tree's directory listing and
SHALL NOT open a roster file. The response SHALL be marked as not to be stored by any cache.

The season and week SHALL be validated as the page's are: a segment that is not a number, or a
season or week out of range, SHALL be refused with status 400. A season with no week directories
SHALL be refused by the season form with status 404.

#### Scenario: The endpoint reports the version a fresh page carries

- **WHEN** a page for week 4 is served and the version endpoint for week 4 is then asked
- **THEN** the endpoint's version equals the page's stats version

#### Scenario: The season form reports the version the season page carries

- **WHEN** the tree holds 2026 weeks 1 and 2, `GET /2026` is served, and `GET /2026/version` is then
  asked
- **THEN** the endpoint's version equals the page's stats version

#### Scenario: A new week changes the season form's answer

- **WHEN** `GET /2026` is served while week 2 is the latest, then a 2026 week 3 directory appears,
  and `GET /2026/version` is asked
- **THEN** the version reported is the one held for week 3, which is empty if week 3 has not been
  fetched, and differs from the page's

#### Scenario: Invalidation empties the version

- **WHEN** week 4's stats are fetched and then invalidated, and the version endpoint is asked
- **THEN** the version reported is empty

#### Scenario: A refetch changes the version

- **WHEN** week 4's stats are fetched, invalidated, and fetched again, and the version endpoint is asked
- **THEN** the version reported is the second fetch's, and differs from the first page's

#### Scenario: Asking costs nothing upstream

- **WHEN** either form of the version endpoint is asked, whether or not the week has been fetched
- **THEN** no call is made to the stat provider

#### Scenario: The season form opens no roster

- **WHEN** `GET /2026/version` is asked
- **THEN** no roster file is opened

#### Scenario: A malformed week is refused

- **WHEN** `GET /2026/five/version` is requested
- **THEN** the response status is 400 and no version is reported

#### Scenario: A malformed season is refused

- **WHEN** `GET /twentytwentysix/version` is requested
- **THEN** the response status is 400 and no directory is read

#### Scenario: A season with no weeks has no version

- **WHEN** the tree holds no directories under 2027, and `GET /2027/version` is requested
- **THEN** the response status is 404

#### Scenario: The response is not cacheable

- **WHEN** either form of the version endpoint answers
- **THEN** the response carries `Cache-Control: no-store`

### Requirement: The page reloads when the server's stats version differs from its own

While its tab is visible, the page SHALL check the version endpoint for the URL it was loaded from
(`/{season}/{week}/version` for a week's page, `/{season}/version` for a season's page) and compare
the reported version with the one it carries. When they differ, including when the reported version
is empty, the page SHALL reload its own URL.

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

#### Scenario: A season page checks the season form

- **WHEN** the page was loaded from `/2026`
- **THEN** its checks request `/2026/version`

#### Scenario: An open season page follows a new week

- **WHEN** a visible page loaded from `/2026` shows week 2, and a 2026 week 3 directory appears
- **THEN** the next check reports a different version, the page reloads `/2026`, and the reload
  shows week 3

#### Scenario: A matching version does nothing

- **WHEN** a check reports the same stats version the page carries
- **THEN** the page does not reload

#### Scenario: A hidden tab makes no check

- **WHEN** the page's tab is hidden when a check would fire
- **THEN** no request is made to the version endpoint

#### Scenario: A failed check does not reload

- **WHEN** a check request fails
- **THEN** the page does not reload, and checks again after the same delay

## REMOVED Requirements

### Requirement: A season alone redirects to its latest week

**Reason**: `/{season}` becomes the current week's own address rather than a hop to a numbered one.

**Migration**: Replaced by "A season alone shows its latest week". Numbered URLs keep working; only
the `302` from `/{season}` is gone.
