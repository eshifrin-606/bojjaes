## ADDED Requirements

### Requirement: A checked lineup is saved by POST /lineup

When a lineup store is configured, the system SHALL serve `POST /lineup`, taking the form's fields
(`team`, `season`, `week`, and the nine slot fields as the GET form names them) from the request
body. It SHALL apply the same team, current-week, and proposal checks as `GET /lineup.csv`, with the
same statuses: 400 for an unknown team or an invalid or incomplete proposal, 404 when there is no
week to set, 502 when the current week cannot be read. Only a complete, valid lineup SHALL be
written. On success it SHALL answer `303 See Other` to `/lineup?team=<team>`.

A refused save SHALL write nothing, and SHALL answer with the form for the posted team and slots,
the reason listed with the page's other problems.

#### Scenario: Saving a valid lineup

- **WHEN** `POST /lineup` carries `team=bojjaes`, the current season and week, and nine eligible,
  distinct roster players
- **THEN** it answers 303 to `/lineup?team=bojjaes`, and the store's file for bojjaes that week
  holds those nine players in slot order

#### Scenario: An invalid lineup is not saved

- **WHEN** `POST /lineup` carries a duplicate player
- **THEN** it answers 400 listing the duplicate, and no file is written or changed

#### Scenario: An unknown team

- **WHEN** `POST /lineup` carries `team=../wood`
- **THEN** it answers 400 before the current week is read, and nothing is written

### Requirement: Only the week the form shows is saved

A save SHALL be filed under the current week as the form resolves it — Sleeper's settable week, at
most week 17 — and under no other. No field SHALL choose the week written. The posted `season` and
`week` SHALL be compared with the current week, and a save whose posted week differs SHALL be
refused with `409 Conflict`, saying the week changed and the form must be reloaded, rather than
filed under the new week.

#### Scenario: The week the form showed

- **WHEN** the form showed 2026 week 12, the current week is still 2026 week 12, and the save is
  posted
- **THEN** the lineup is written under `2026/12`

#### Scenario: The week turned over while the form was open

- **WHEN** the form showed 2026 week 12, the current week is now 2026 week 13, and the save is
  posted
- **THEN** it answers 409, and nothing is written under week 12 or week 13

#### Scenario: No week to set

- **WHEN** the current week is NFL week 18
- **THEN** `POST /lineup` answers 404 and nothing is written

### Requirement: A save fills the Bojjaes' side or their opponent's

A week directory SHALL hold at most the Bojjaes' lineup and one opponent's. Saving `bojjaes` SHALL
create or replace `bojjaes.csv`. Saving any other team SHALL create that team's file when the week
holds no opponent, and SHALL replace it when that team is already the week's opponent. Saving a
team other than `bojjaes` when the week already holds a different opponent SHALL be refused with
`409 Conflict` and the message `bojjaes' opponent is already <team>`, and nothing SHALL be written.
A save SHALL NOT delete any file. Files whose names begin with a dot, and files not ending in
`.csv`, SHALL NOT count as an opponent.

#### Scenario: The first lineup of a new week

- **WHEN** the store has no `2026/12` directory and bojjaes is saved for 2026 week 12
- **THEN** `2026/12/` is created holding only `bojjaes.csv`

#### Scenario: The opponent joins

- **WHEN** `2026/12/` holds only `bojjaes.csv` and wood is saved
- **THEN** `2026/12/` holds `bojjaes.csv` and `wood.csv`

#### Scenario: The opponent is saved first

- **WHEN** the store has no `2026/12` directory and wood is saved for 2026 week 12
- **THEN** `2026/12/` holds only `wood.csv`

#### Scenario: Re-saving a side replaces it

- **WHEN** `2026/12/` holds `bojjaes.csv` and `wood.csv`, and bojjaes or wood is saved again
- **THEN** that team's file holds the new lineup and the directory still holds exactly those two
  files

#### Scenario: A different opponent is refused

- **WHEN** `2026/12/` holds `bojjaes.csv` and `wood.csv`, and fuego is saved
- **THEN** it answers 409 with `bojjaes' opponent is already wood`, `fuego.csv` is not created, and
  `wood.csv` is unchanged

#### Scenario: A different opponent is refused before the Bojjaes are saved

- **WHEN** `2026/12/` holds only `wood.csv`, and fuego is saved
- **THEN** it answers 409 with `bojjaes' opponent is already wood`

### Requirement: A saved lineup is written whole or not at all

A save SHALL write the lineup as a roster file in slot order, as `lineup-source` renders a lineup,
first to a temporary file in the same week directory whose name begins with a dot, then rename it
over the target. A reader SHALL see either the previous file or the new one, never a partial file.
A failed write SHALL leave the previous file in place, SHALL remove its temporary file, and SHALL
answer 500. Saves SHALL be serialized within the process, so two concurrent saves to one week
cannot together leave a third file.

#### Scenario: The written file reads back

- **WHEN** a lineup is saved
- **THEN** reading that season, week, and team from the lineup tree yields the same nine starters
  in the same slots

#### Scenario: Concurrent opponents

- **WHEN** `2026/12/` holds only `bojjaes.csv`, and saves of wood and fuego arrive at once
- **THEN** exactly one of them is written and the other is refused with 409

### Requirement: A saved lineup is served at once and survives restarts

A saved lineup SHALL be what the lineup tree serves for its week on the next request, with no
restart. It SHALL remain in the store across restarts and deploys, and SHALL be replaced only as
`lineup-source`'s seeding replaces a store week: when git's copy of that week changed since it was
last seeded, which logs the replaced files.

#### Scenario: The scoreboard shows the save

- **WHEN** bojjaes is saved for 2026 week 12 and `GET /2026/12` is requested
- **THEN** the Bojjaes' column shows the saved starters

#### Scenario: A restart keeps the save

- **WHEN** bojjaes is saved for a week git also holds, and the server restarts with git's copy of
  that week unchanged
- **THEN** the saved lineup is still served

### Requirement: With no store there is no save

When no lineup store is configured, `POST /lineup` SHALL answer 405, as before this capability.

#### Scenario: Local development

- **WHEN** the server runs with no volume configured and `POST /lineup` is requested
- **THEN** it answers 405
