## ADDED Requirements

### Requirement: The form saves with a Save button

When a lineup store is configured, the slots form SHALL show a `Save` button beside `Copy`, and SHALL
carry the season and week it shows as hidden fields. Pressing `Save` SHALL post the form to
`POST /lineup` (see `lineup-save`) without copying anything to the clipboard. With no store, the
form SHALL show no `Save` button.

#### Scenario: Save is offered

- **WHEN** a store is configured and `GET /lineup` shows 2026 week 12
- **THEN** the slots form has a `Save` button that posts to `/lineup`, and hidden fields
  `season=2026` and `week=12`

#### Scenario: No store

- **WHEN** no store is configured
- **THEN** the page has no `Save` button

#### Scenario: Save does not copy

- **WHEN** `Save` is pressed with the clipboard available
- **THEN** the form is posted and nothing is written to the clipboard

### Requirement: The form says whether the week's lineup is saved

When there is a week to set, the page SHALL state whether the lineup tree holds the team's lineup
for that week: `Saved for week <n>` when it does; otherwise `Not saved for week <n>`, followed by
the week the slots start from when they start from an earlier lineup.

#### Scenario: Saved

- **WHEN** the tree holds wood's lineup for 2026 week 12, the current week
- **THEN** the page says `Saved for week 12`

#### Scenario: Starting from an earlier week

- **WHEN** the tree's newest bojjaes lineup at or before 2026 week 12 is week 11
- **THEN** the page says `Not saved for week 12` and that the slots start from week 11

#### Scenario: After a save

- **WHEN** a save of bojjaes for week 12 succeeds and the browser follows the redirect
- **THEN** the page says `Saved for week 12` and the slots show the saved lineup

## MODIFIED Requirements

### Requirement: The form writes nothing

The `GET` routes `/lineup`, `/lineup.csv`, and `/lineup.txt` SHALL NOT write to the lineup tree.
`/lineup.csv` and `/lineup.txt` SHALL answer only GET. Writing is `POST /lineup`'s alone, as
`lineup-save` defines it; with no store configured, `POST /lineup` SHALL answer 405.

#### Scenario: A POST with no store

- **WHEN** no store is configured and `POST /lineup` is requested
- **THEN** it answers 405

#### Scenario: A POST to the download

- **WHEN** `POST /lineup.csv` is requested
- **THEN** it answers 405
