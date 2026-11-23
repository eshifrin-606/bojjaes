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
the week the slots start from when they start from an earlier lineup. Once any slot select is
changed, the line SHALL read `Unsaved changes` instead, until the page is next loaded.

#### Scenario: Saved

- **WHEN** the tree holds wood's lineup for 2026 week 12, the current week
- **THEN** the page says `Saved for week 12`

#### Scenario: Starting from an earlier week

- **WHEN** the tree's newest bojjaes lineup at or before 2026 week 12 is week 11
- **THEN** the page says `Not saved for week 12` and that the slots start from week 11

#### Scenario: After a save

- **WHEN** a save of bojjaes for week 12 succeeds and the browser follows the redirect
- **THEN** the page says `Saved for week 12` and the slots show the saved lineup

#### Scenario: An edit not yet saved

- **WHEN** the page says `Saved for week 12` and the reader changes any slot select
- **THEN** the line reads `Unsaved changes`, with no reload, until the page is next loaded

### Requirement: Save leads the action row

With a `Save` button, the action row SHALL hold, left to right, `Save` at half the row's width,
then `Copy` and `Download` at a quarter each. `Save` SHALL be highlighted, filled muted green
(#4f8a5b) with white text, only while there is something to save: when this week's lineup is not
saved, or once a slot has changed. Otherwise it SHALL look like `Copy` and `Download`, and its
width SHALL not change either way. Without `Save`, `Copy` SHALL take three
quarters of the row and `Download` one quarter. `Download` SHALL show only a download glyph (a
down arrow over a line) and SHALL have the accessible name `Download`.

#### Scenario: With a store

- **WHEN** a store is configured and `GET /lineup` is requested
- **THEN** the action row reads `Save`, `Copy`, then the `Download` glyph, and `Save` is twice as
  wide as `Copy`, which is as wide as `Download`

#### Scenario: Save is highlighted for an unsaved week

- **WHEN** a store is configured and this week's lineup is not saved
- **THEN** `Save` is filled muted green with white text

#### Scenario: Save is plain for a saved week

- **WHEN** a store is configured and this week's lineup is saved
- **THEN** `Save` looks like `Copy` and `Download` and is still twice as wide as `Copy`

#### Scenario: An edit highlights Save

- **WHEN** this week's lineup is saved and a slot select is changed
- **THEN** `Save` is highlighted as the status line reads `Unsaved changes`

#### Scenario: With no store

- **WHEN** no store is configured
- **THEN** `Copy` is three times as wide as `Download`

#### Scenario: Download is named for screen readers

- **WHEN** the page is rendered
- **THEN** the `Download` button holds an inline SVG glyph and no visible text, and its accessible
  name is `Download`

## MODIFIED Requirements

### Requirement: Copy checks the lineup and copies its thread text

The slots form's `Copy` button SHALL check the proposal by reloading the page for it, as submitting
the form would, and, when the proposal is a valid lineup, SHALL first copy the lineup's thread text
to the clipboard and show the button as `Copied` on the reloaded page. Changing any slot select
SHALL set the button back to `Copy`. Without a clipboard (or without script) the button SHALL still
submit the form.

`GET /lineup.txt` SHALL answer the thread text as `text/plain`: a first line naming the team,
season, and week, then one line per slot of slot, name, and NFL team, in slot order. It SHALL apply
the same checks and statuses as `GET /lineup.csv`.

#### Scenario: Copying a valid lineup

- **WHEN** Copy is pressed on wood's valid proposal for 2026 week 9
- **THEN** the clipboard holds text whose first line is `wood · 2026 week 9` and whose next nine
  lines start `QB `, `RB `, `WR `, `TE `, `FLEX `, `FLEX `, `K `, `D `, `D `, and the reloaded
  page's button reads `Copied`

#### Scenario: Copying an invalid lineup

- **WHEN** Copy is pressed on a proposal with a duplicate player
- **THEN** nothing is copied, and the reloaded page lists the problem

### Requirement: A valid lineup downloads as a roster file

The slots form SHALL always show an enabled `Download` button, whatever the slots hold; it SHALL
never be greyed out, and editing a slot SHALL NOT disable it. Pressing it SHALL check the slots as
`Copy` does: when they form a valid lineup, the browser SHALL download `GET /lineup.csv` with the
form's fields; otherwise the page SHALL reload at `/lineup` with the form's fields, listing the
problems. Without script, `Download` SHALL submit the form to `GET /lineup.csv`, which is still the
check: a valid lineup downloads, and an invalid one answers 400 naming its problems.

`GET /lineup.csv` SHALL answer the lineup as a roster file in slot order (as the lineup renders
itself), with content type `text/csv` and an attachment file name `<team>.csv`. It SHALL apply the
page's team, current-week, and proposal checks, answering 400 for an unknown team or an invalid or
incomplete proposal, 404 when there is no week to set, and 502 when the current week cannot be read.

#### Scenario: Download is offered for any slots

- **WHEN** the slots hold a duplicate player, or a starting lineup with an empty slot
- **THEN** `Download` is shown enabled, not greyed out

#### Scenario: Downloading a valid lineup

- **WHEN** Download is pressed on wood's valid slots
- **THEN** the browser requests `/lineup.csv` with the form's fields and the page is not reloaded

#### Scenario: Downloading an invalid lineup

- **WHEN** Download is pressed on slots with a duplicate player
- **THEN** no file is downloaded, and the page reloads at `/lineup` with those slots and lists the
  problem

#### Scenario: Download

- **WHEN** `GET /lineup.csv` carries a valid proposal for wood
- **THEN** it answers 200, `Content-Disposition: attachment; filename="wood.csv"`, and a body of the
  header row and nine records in slot order

#### Scenario: Invalid download

- **WHEN** `GET /lineup.csv` carries a duplicate player
- **THEN** it answers 400 and the body names the duplicate

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
