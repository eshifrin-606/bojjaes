## ADDED Requirements

### Requirement: A slot states which positions it accepts

Each slot SHALL answer whether it accepts a position, by the same rule that fills slots on read:
RB accepts RB and FB; FLEX accepts anything RB, WR, or TE accepts; D accepts every defensive
position; QB, WR, TE, and K accept only themselves. The nine slots SHALL be available in slot
order.

#### Scenario: Slot acceptance

- **WHEN** RB is asked about FB, FLEX about TE, D about CB, and QB about WR
- **THEN** the answers are yes, yes, yes, and no

### Requirement: A team's roster is read by team name

Roster files SHALL be located as `<team>.csv` within a caller-supplied roster directory, and SHALL
be read with the same record rules as a lineup file (header, closed sets, unique IDs), in file
order, but with no limit on the number of records and no slot assignment. The team names SHALL be
listable, in name order, as the `.csv` files in that directory, ignoring dotfiles. A team name that
is not a single plain path segment SHALL be refused.

#### Scenario: Teams are listed

- **WHEN** the directory holds `wood.csv`, `aroma.csv`, `.DS_Store`, and `notes.txt`
- **THEN** the teams are `aroma`, `wood`

#### Scenario: A roster of more than nine

- **WHEN** `wood.csv` holds 25 records
- **THEN** reading wood's roster yields all 25 in file order

#### Scenario: A path as a team

- **WHEN** the roster for `../wood` is read
- **THEN** the read fails

#### Scenario: Every embedded roster reads

- **WHEN** each team listed in the embedded rosters is read
- **THEN** every read succeeds, and the teams are the league's ten

### Requirement: A lineup can be built from chosen slots

A lineup SHALL be buildable from nine records given in slot order, keeping each record in the slot
it was given. It SHALL be refused when a record has no ID, a slot does not accept its record's
position, or two records share an ID; the error SHALL name the slot. Rendered back to a roster file
and read again, it SHALL yield the same starters in the same slots.

#### Scenario: Chosen order is kept

- **WHEN** a lineup is built with an RB in RB and a second RB in the first FLEX
- **THEN** its starters are in the slots given

#### Scenario: Ineligible record

- **WHEN** a WR is given for the QB slot
- **THEN** building fails, naming QB

#### Scenario: Duplicate record

- **WHEN** the same ID is given for RB and FLEX
- **THEN** building fails

#### Scenario: Round trip

- **WHEN** a built lineup is written as a roster file and read back from the tree
- **THEN** the starters equal the ones it was built from

### Requirement: A team's latest lineup at or before a week can be found

The tree SHALL find the newest season and week, at or before a given season and week, that holds a
lineup file for a team, without opening it. Weeks after the bound, in its season or a later one,
SHALL NOT count. When there is none, it SHALL report that the team has no lineup. A team name that
is not a single plain path segment SHALL be refused.

#### Scenario: The bound week itself

- **WHEN** wood has lineups for 2026 weeks 8 and 9, and the bound is 2026 week 9
- **THEN** the answer is 2026 week 9

#### Scenario: A later week is skipped

- **WHEN** wood has lineups for 2026 weeks 7 and 10, and the bound is 2026 week 9
- **THEN** the answer is 2026 week 7

#### Scenario: Back into an earlier season

- **WHEN** wood has a lineup for 2025 week 16 and 2026 week 3, and the bound is 2026 week 2
- **THEN** the answer is 2025 week 16

#### Scenario: Nothing at or before

- **WHEN** wood's only lineup is 2026 week 10 and the bound is 2026 week 9
- **THEN** it reports no lineup
