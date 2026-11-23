# lineup-form Specification

## Purpose

Serve a phone form for building a legal lineup for the current week: a team picker, one select per
slot limited to the players that slot accepts, a proposal carried in the URL and checked on the
page, and, for a valid lineup, its thread text copied to the clipboard, a roster-file download, and,
when a store is configured, a `Save` button. Its GET routes write nothing; saving belongs to
`lineup-save`.

## Requirements

### Requirement: The form is served for the current week

The system SHALL serve the lineup form at `GET /lineup`. Its season and week SHALL be the settable
week from the current-week source, shown as labels; no query parameter SHALL choose a season or a
week. The page SHALL show no points, totals, or margin.

#### Scenario: A mid-season week

- **WHEN** the current week is season 2026, regular season, week 9, and `GET /lineup` is requested
- **THEN** the page answers 200 and names 2026 week 9

#### Scenario: A season or week in the query is ignored

- **WHEN** the current week is 2026 week 9 and `GET /lineup?season=2025&week=3` is requested
- **THEN** the page names 2026 week 9

### Requirement: Outside the league's weeks there is no week to set

When the current-week source has no settable week, or its settable week is after week 17 (the
HMFFL's last week), the page SHALL answer 200 saying there is no week to set, SHALL show no slot
selects, and SHALL NOT read the lineup tree or the games source.

#### Scenario: The offseason

- **WHEN** the current week is the offseason
- **THEN** the page says there is no week to set, and the games source is not called

#### Scenario: NFL week 18

- **WHEN** the current week is regular season, week 18
- **THEN** the page says there is no week to set

#### Scenario: The league's last week

- **WHEN** the current week is regular season, week 17
- **THEN** the page shows the slot selects for week 17

### Requirement: A failed current-week read is not a page

When the current-week source fails, the page SHALL answer 502 and SHALL NOT render a form for a
guessed week.

#### Scenario: The source fails

- **WHEN** the current-week source returns an error
- **THEN** the page answers 502

### Requirement: The team is picked from the teams that have rosters

The page SHALL offer a team picker listing every team with a roster file, in name order. The
`team` query parameter SHALL select the team; when absent, the team SHALL be `bojjaes`. The picker
SHALL be a GET form to `/lineup` that submits itself when the team changes, so changing the team
reloads the page for that team with no proposed slots; it SHALL have no submit button. A `team` naming no roster SHALL answer 400 before the current-week source or the
games source is called.

#### Scenario: The default team

- **WHEN** `GET /lineup` is requested
- **THEN** the picker lists all ten teams and `bojjaes` is selected

#### Scenario: Another team

- **WHEN** `GET /lineup?team=wood` is requested
- **THEN** `wood` is selected and the slots offer wood's roster

#### Scenario: An unknown team

- **WHEN** `GET /lineup?team=nobody` or `GET /lineup?team=../wood` is requested
- **THEN** the page answers 400, and neither the current-week source nor the games source is called

### Requirement: Each slot is a select of the roster players it accepts

The page SHALL show nine selects in slot order — QB, RB, WR, TE, FLEX, FLEX, K, D, D — named `qb`,
`rb`, `wr`, `te`, `flex`, `flex`, `k`, `d`, `d`. Each select's options SHALL be the players of the
team's roster file (roster transactions are not consulted) whose position that slot accepts, in
roster file order, preceded by an empty option. A player in the starting lineup (see *Slots start
from the team's lineup*) who is not on the roster SHALL also be offered in the slots that accept
his position. Each option's value SHALL be the player's Sleeper ID.

#### Scenario: A QB slot

- **WHEN** the roster holds a QB, two RBs, an FB, a WR, and a CB
- **THEN** the QB select offers only the QB, and the empty option

#### Scenario: A FLEX slot

- **WHEN** the roster holds a QB, an RB, an FB, a WR, a TE, and a CB
- **THEN** each FLEX select offers the RB, FB, WR, and TE, and not the QB or CB

#### Scenario: A D slot

- **WHEN** the roster holds a DE, a CB, and a LB
- **THEN** each D select offers all three

### Requirement: Each option states the player's position, NFL team, and game

Each option's text SHALL be the player's name, position, and NFL team, then the player's game in
the current week as the matchup page states it: the kickoff in the matchup page's compact Central
form for an upcoming game, otherwise the game's state text (for example `Q3`, `Final`, `Bye`). A
free agent, or a week whose games are unavailable, SHALL show no game, and a games failure SHALL
NOT fail the page.

#### Scenario: An upcoming game

- **WHEN** Amon-Ra St. Brown (WR, DET) plays a game kicking off Sunday 12:00 Central
- **THEN** his option reads `Amon-Ra St. Brown · WR · DET · Sun 12p`

#### Scenario: A bye

- **WHEN** a player's team has no game in a week that has games
- **THEN** his option ends `· Bye`

#### Scenario: The games are unavailable

- **WHEN** the games source fails
- **THEN** the page answers 200 and each option ends at the NFL team

### Requirement: Each slot shows its chosen player as a card

Each slot SHALL show its selected player as the matchup page shows a starter: the name on one line,
then a meta line of position, NFL team, and game, the game stated as the option states it. The
slot's select SHALL lie over the card, unseen, so a tap opens the browser's own picker. A slot on
the empty option SHALL show an empty card. Changing a slot's select SHALL redraw its card from the
newly selected option without a reload.

#### Scenario: A chosen player

- **WHEN** the WR slot holds Amon-Ra St. Brown (WR, DET), kicking off Sunday 12:00 Central
- **THEN** the card's name reads `Amon-Ra St. Brown` and its meta line `WR · DET · Sun 12p`

#### Scenario: An empty slot

- **WHEN** a slot is on the empty option
- **THEN** its card shows no name and no meta line

#### Scenario: A changed slot

- **WHEN** the reader picks Derrick Henry in an RB slot that held Bijan Robinson
- **THEN** that slot's card names Derrick Henry, with no reload

### Requirement: Slots start from the team's lineup

With no proposed slots in the URL, each select SHALL start on the player in that slot of the team's
lineup for the current week; failing that, of the team's latest lineup in an earlier week; failing
that, on the empty option. A later week's lineup SHALL NOT be used. A starting lineup file that
cannot be read SHALL leave every slot empty and SHALL say so on the page, not fail it.

#### Scenario: This week's file exists

- **WHEN** the team has lineups for 2026 weeks 8 and 9 and the current week is 2026 week 9
- **THEN** the slots start on the week 9 lineup

#### Scenario: Only an earlier week exists

- **WHEN** the team's newest lineup is 2026 week 7, and it also has one for week 10, and the
  current week is 2026 week 9
- **THEN** the slots start on the week 7 lineup

#### Scenario: No lineup at all

- **WHEN** the team has no lineup at or before the current week
- **THEN** every select starts on the empty option

#### Scenario: A starter who left the roster

- **WHEN** the starting lineup's TE is not in the team's roster file
- **THEN** the TE select shows him selected

### Requirement: A proposed lineup is encoded in the URL

The slot selects and a "Copy" button SHALL form a GET form to `/lineup` carrying `team` as a
hidden field, so the submitted URL is `/lineup?team=<team>&qb=<id>&rb=<id>&wr=<id>&te=<id>&flex=<id>&flex=<id>&k=<id>&d=<id>&d=<id>`.
Repeated `flex` and `d` values SHALL fill those slots in URL order. When any slot parameter is
present, the URL SHALL be a proposal: every select SHALL start on the proposed player, not the
starting lineup, and a slot with no value SHALL be empty.

#### Scenario: Loading a shared URL

- **WHEN** a URL proposes nine roster players, each in a slot that accepts him
- **THEN** each select shows its proposed player selected and no error is shown

#### Scenario: FLEX order follows the URL

- **WHEN** a URL has `flex=A&flex=B`
- **THEN** the first FLEX select shows A and the second shows B

### Requirement: A proposed lineup is checked on the page

A proposal SHALL be checked and every problem listed in red below the slots with a 200 answer, never a 5xx:

- an ID that is neither on the roster nor in the starting lineup is unknown, and its slot is empty;
- a player in a slot that does not accept his position is ineligible, and stays selected there;
- a player in two slots is a duplicate, and stays selected in both;
- a slot with no player is missing;
- more `flex` or `d` values than there are slots of that kind is an error.

#### Scenario: Unknown ID

- **WHEN** `qb=99999` and no such player is on the roster or starting lineup
- **THEN** the page answers 200, names `99999` as unknown, and the QB select is empty

#### Scenario: Ineligible player

- **WHEN** a WR's ID is given as `qb`
- **THEN** the page says that player cannot play QB, and the QB select shows him selected

#### Scenario: Duplicate player

- **WHEN** the same RB's ID is given as `rb` and as `flex`
- **THEN** the page says he is in two slots

#### Scenario: Missing slot

- **WHEN** a proposal has no `k`
- **THEN** the page says the K slot is empty

#### Scenario: Three FLEX values

- **WHEN** a proposal has three `flex` values
- **THEN** the page says there are too many FLEX players

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

### Requirement: The lineup routes and the season routes do not capture each other

`/lineup` and `/lineup.csv` SHALL reach the lineup handlers, and `/{season}` and `/{season}/{week}`
SHALL keep reaching the matchup handlers.

#### Scenario: Literal wins

- **WHEN** `GET /lineup` is requested
- **THEN** the lineup form answers, not the season handler's 400

#### Scenario: A season still resolves

- **WHEN** `GET /2025` is requested
- **THEN** the season page answers

### Requirement: Roster text on the form is escaped as HTML

Names and labels from roster and lineup files SHALL be escaped wherever the form renders them.

#### Scenario: A name with markup

- **WHEN** a roster name is `<b>X</b>`
- **THEN** the page contains `&lt;b&gt;X&lt;/b&gt;` and no `<b>` element

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
