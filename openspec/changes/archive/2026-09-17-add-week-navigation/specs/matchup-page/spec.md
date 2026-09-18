## ADDED Requirements

### Requirement: The page names its week and links to the adjacent weeks of its season

The page SHALL show a week bar above both columns and outside either one. The bar SHALL hold a
label naming the season and week the page shows, in the form `2026 · Week 2`.

The bar SHALL hold a link to the previous week at its left when that week exists, and a link to the
next week at its right when that week exists. The previous week is the same season, week − 1. The
next week is the same season, week + 1. A link SHALL NOT cross into another season: week 1 has no
previous link, and a season's last week has no next link, even if the other season has lineups.
Whether a week exists SHALL be decided as `lineup-source` decides it, so a week whose directory is
not a well-formed matchup still gets a link.

Each link SHALL be an ordinary hyperlink to that week's page (`/{season}/{week}`). The previous link
SHALL carry `rel="prev"` and the next link `rel="next"`. Each link's text SHALL name the week it
leads to.

Deciding which links to show SHALL NOT call the stats provider, and the page SHALL still fetch only
the week it shows.

When a link is absent, its space SHALL stay empty rather than collapse: the label's horizontal
position SHALL NOT depend on which links are present. The label and both links SHALL fit on one line
on a phone held in portrait. The bar SHALL be styled identically whatever either column's points
are.

#### Scenario: A middle week links both ways

- **WHEN** the tree holds 2026 weeks 1, 2, and 3, and `GET /2026/2` is requested
- **THEN** the page's label reads `2026 · Week 2`, a `rel="prev"` link points to `/2026/1`, and a
  `rel="next"` link points to `/2026/3`

#### Scenario: The first week has no previous link

- **WHEN** the tree holds 2026 weeks 1 and 2, and `GET /2026/1` is requested
- **THEN** the page has a `rel="next"` link to `/2026/2` and no `rel="prev"` link

#### Scenario: The latest week has no next link

- **WHEN** the tree holds 2026 weeks 1 and 2, and `GET /2026/2` is requested
- **THEN** the page has a `rel="prev"` link to `/2026/1` and no `rel="next"` link

#### Scenario: Links do not cross a season boundary

- **WHEN** the tree holds 2025 week 16 and 2026 week 1, and no 2025 week 17 or 2026 week 0
- **THEN** the 2025 week 16 page has no next link and the 2026 week 1 page has no previous link

#### Scenario: A gap in a season is not skipped

- **WHEN** the tree holds 2026 weeks 1 and 3 but not week 2, and `GET /2026/3` is requested
- **THEN** the page has no `rel="prev"` link

#### Scenario: A broken neighbouring week still gets a link

- **WHEN** the tree's 2026 week 3 directory holds three rosters, and `GET /2026/2` is requested
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
