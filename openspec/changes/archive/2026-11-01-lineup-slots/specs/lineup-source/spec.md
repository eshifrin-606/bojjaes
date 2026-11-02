## ADDED Requirements

### Requirement: A lineup is nine slots in a fixed order

A lineup SHALL be exactly nine slots, in this canonical order: QB, RB, WR, TE, FLEX, FLEX, K, D, D.
Each slot SHALL hold exactly one record, and the starters SHALL be those nine records in slot order.
There SHALL be no bench: every record in a roster file fills a slot.

A slot SHALL accept a record by its position:

- QB accepts `QB`; K accepts `K`.
- RB accepts `RB` or `FB`; WR accepts `WR`; TE accepts `TE`.
- FLEX accepts any position RB, WR, or TE accepts.
- D accepts `DL DE DT NT LB OLB ILB DB CB S SS FS`.

A record's position SHALL remain the player's listed position as written in the file. Filling a
slot SHALL NOT rewrite it, so a fullback in the RB slot is still shown and written as `FB`.

#### Scenario: A file already in slot order keeps its order

- **WHEN** a roster file lists a QB, RB, WR, TE, WR, RB, K, LB, and CB in that order
- **THEN** the starters are those nine records in file order

#### Scenario: A fullback fills the RB slot

- **WHEN** a roster file holds a QB, an `FB`, two WRs, a TE, another WR, a K, and two defenders,
  and no `RB`
- **THEN** the `FB` record fills the RB slot and its position is still `FB`

### Requirement: Records fill the slots by position, not by file order

Reading a roster SHALL assign its records to slots deterministically, whatever order the file lists
them in:

1. QB takes the `QB` record; K takes the `K` record.
2. RB, WR, and TE each take the first record in file order that it accepts.
3. The two FLEX slots take the remaining RB, WR, and TE records, in file order.
4. The two D slots take the defensive records, in file order.

Reordering lines of a roster file SHALL change a slot's occupant only where that order breaks a
tie: which of several same-position records fills the RB, WR, or TE slot and which a FLEX, the
order of the two FLEX slots, and the order of the two D slots.

#### Scenario: A file in another order is read into slot order

- **WHEN** a roster file lists a K, LB, WR, WR, TE, RB, RB, QB, and DE in that order
- **THEN** the starters are the QB, the first RB, the first WR, the TE, the second WR, the second
  RB, the K, the LB, and the DE

#### Scenario: Extra players at one position fill the FLEX slots in file order

- **WHEN** a roster file lists three RBs before its only WR and its only TE
- **THEN** the first RB fills the RB slot, and the second and third fill the two FLEX slots in
  file order

#### Scenario: A FLEX may hold a tight end

- **WHEN** a roster file holds one RB, one WR, and three TEs
- **THEN** the first TE fills the TE slot and the second and third fill the FLEX slots

#### Scenario: Defenders keep file order

- **WHEN** a roster file lists a CB above a DE
- **THEN** the CB fills the first D slot and the DE the second

#### Scenario: Swapping two RB lines swaps the RB and the FLEX

- **WHEN** a roster file's two RBs, filling the RB slot and a FLEX slot, swap lines
- **THEN** each now fills the slot the other filled, and no other slot changes

### Requirement: A roster that cannot fill the slots is refused

Reading SHALL fail with an error naming the file when the roster does not hold exactly nine
records, or when its records cannot fill every slot by the rules in *Records fill the slots by
position, not by file order*. A roster is never partly slotted, and no record is dropped or moved
into a slot that does not accept it.

When the count is wrong the error SHALL state the count. When nine records cannot fill the slots
the error SHALL name a slot left empty.

#### Scenario: A tenth player row is refused

- **WHEN** a roster file holds ten records
- **THEN** reading fails with an error naming the file and stating ten records

#### Scenario: Eight player rows are refused

- **WHEN** a roster file holds eight records
- **THEN** reading fails with an error naming the file and stating eight records

#### Scenario: Comments do not count as players

- **WHEN** a roster file holds nine records and comment and blank lines between them
- **THEN** reading succeeds

#### Scenario: A roster with no tight end is refused

- **WHEN** a roster file holds nine records: a QB, two RBs, three WRs, a K, and two defenders
- **THEN** reading fails with an error naming the TE slot

#### Scenario: A second quarterback is refused

- **WHEN** a roster file holds two QBs and otherwise one each of RB, WR, TE, a fourth skill player,
  K, and two defenders
- **THEN** reading fails, because the second QB fits no slot and a FLEX slot is left empty

#### Scenario: A third defender is refused

- **WHEN** a roster file holds three defenders, one QB, one K, and four RB, WR, and TE records
- **THEN** reading fails with an error naming a slot left empty

### Requirement: A lineup renders back to roster records in slot order

A lineup SHALL render to the text of a roster file: the header `id,name,position,team`, then one
record per slot, in canonical slot order, each carrying its id, name, position, and team as read.
The rendering SHALL NOT include a short name, comments, or blank lines.

Reading the rendered text SHALL yield a lineup equal to the one rendered.

#### Scenario: A lineup read from any order renders in slot order

- **WHEN** a roster file listing its K first and its QB last is read and rendered
- **THEN** the rendered text is the header followed by the QB, RB, WR, TE, FLEX, FLEX, K, D, D
  records, in that order

#### Scenario: Rendering keeps a fullback's position

- **WHEN** a lineup whose RB slot holds an `FB` is rendered
- **THEN** that record's line carries the position `FB`

#### Scenario: A rendered lineup reads back unchanged

- **WHEN** a lineup is rendered and the text is read as a roster file
- **THEN** the starters, in slot order, have the same ids, names, positions, and teams

## MODIFIED Requirements

### Requirement: A roster file is an ordered list of records

A roster file SHALL be read as a header row followed by records, one record per line, in file order.
Parsing SHALL yield the records in that order and SHALL NOT sort, group, or otherwise rearrange
them. File order does not decide a record's slot; it only breaks ties when the records are assigned
to slots, per *Records fill the slots by position, not by file order*.

A line SHALL be split on every comma, and each field SHALL be taken by the column its position in
the header names. Surrounding spaces SHALL be trimmed from every field and every header name, so
alignment spacing in a hand-edited file does not become part of a value.

Blank lines and lines whose first non-space character is `#` SHALL NOT be records and SHALL NOT be
the header. A final line with no trailing newline SHALL still yield its record.

#### Scenario: Records are returned in file order

- **WHEN** a roster file holds a header and three records
- **THEN** reading it yields those three records in the order they appear in the file

#### Scenario: Comments and blank lines are not records

- **WHEN** a roster file opens with a `#` comment line, then the header, and holds a blank line
  between two records
- **THEN** reading it yields only the records, and the comment and blank line occupy no position

#### Scenario: An indented comment is still a comment

- **WHEN** a line's first non-space character is `#`
- **THEN** that line is neither the header nor a record

#### Scenario: Surrounding spaces are not part of a field

- **WHEN** under the header `id, name, position, team` a record is written as
  `4984 , Josh Allen , QB , BUF`
- **THEN** its id is `4984`, its name `Josh Allen`, its position `QB`, and its team `BUF`

#### Scenario: A file with no final newline yields its last record

- **WHEN** a roster file's last line has no trailing newline
- **THEN** that line is read as a record

### Requirement: A line that is neither a comment nor a record is an error

A non-blank, non-comment line after the header SHALL make reading the roster fail with an error
naming the file and the line number when it yields no id, or when its number of fields differs
from the number of columns in the header. Reading SHALL NOT skip such a line.

A silently skipped line is worse than a refused file: it drops a player from the lineup, or changes
which player fills a FLEX or D slot, without anything visible in the report saying so. A line
with the wrong number of fields is refused rather than padded or truncated, because a missing or
extra comma moves every later value into the wrong column.

#### Scenario: A line with no id is refused

- **WHEN** a roster file holds the line `,Josh Allen,QB,BUF`
- **THEN** reading fails with an error naming the file and that line's number

#### Scenario: A line that is only spaces and commas is refused

- **WHEN** a roster file holds the line `  , , ,  `
- **THEN** reading fails rather than treating the line as blank

#### Scenario: A line with too few fields is refused

- **WHEN** under a four-column header a roster file holds the line `4984,Josh Allen`
- **THEN** reading fails with an error naming the file and that line's number

#### Scenario: A name containing a comma is refused

- **WHEN** under a four-column header a roster file holds the line `1234,Smith, Jr.,WR,BUF`
- **THEN** reading fails, because the line has five fields

### Requirement: Every record read from a roster carries a short name

Every record in a lineup read from a roster SHALL carry a short name. The
short name SHALL be derived from the record's name when the roster is read. It SHALL NOT be read
from the file, and the roster file SHALL NOT carry a column for it. The short name SHALL be
non-empty whenever the name is non-empty.

The name SHALL be taken as words separated by whitespace. The short name SHALL be the first word's
lead, a single space, and the remaining words joined by single spaces. The lead SHALL be the first
character of the first word, with no period after it, unless the first word is written as initials,
in which case the lead SHALL be the first word as written.

The first word SHALL count as written as initials when it contains a period, or when it is exactly
two letters and both are uppercase. No other first word SHALL count as initials. A first word with
lowercase letters, or of three or more letters with no period, or joined by a hyphen or apostrophe,
SHALL contribute only its first character.

When the name has three or more words and its last word is a generational suffix (`Jr.`, `Jr`,
`Sr.`, `Sr`, `II`, `III`, `IV`, or `V`), that word SHALL be left out of the short name, whether or
not the first word is initials. The name itself SHALL keep the suffix.

A name of one word SHALL be its own short name.

A short name is a narrower label, not a normalized one. Deriving it SHALL NOT change the record's
name, and SHALL NOT fail a read for any name the roster accepts.

#### Scenario: A two-word name becomes an initial and a surname

- **WHEN** a record's name is `Josh Allen`
- **THEN** its short name is `J Allen`

#### Scenario: A hyphenated first name contributes one letter

- **WHEN** a record's name is `Amon-Ra St. Brown`
- **THEN** its short name is `A St. Brown`

#### Scenario: A first name written with periods is kept as written

- **WHEN** a record's name is `A.J. Brown`
- **THEN** its short name is `A.J. Brown`

#### Scenario: A first name of two capital letters is kept as written

- **WHEN** a record's name is `KC Concepcion`, or `DJ Moore`
- **THEN** its short name is `KC Concepcion`, or `DJ Moore`

#### Scenario: A short or mixed-case first name is not initials

- **WHEN** a record's name is `Al Smith`, or `CeeDee Lamb`
- **THEN** its short name is `A Smith`, or `C Lamb`

#### Scenario: Three capitals with no period are not initials

- **WHEN** a record's name is `JOE Smith`
- **THEN** its short name is `J Smith`

#### Scenario: A generational suffix is dropped from the short name only

- **WHEN** a record's name is `Will McDonald IV`
- **THEN** its short name is `W McDonald` and its name is still `Will McDonald IV`

#### Scenario: A suffix written with or without a period is dropped

- **WHEN** a record's name is `Marvin Harrison Jr.`, or `Marvin Harrison Jr`
- **THEN** its short name is `M Harrison`

#### Scenario: A suffix is dropped after an initials first name

- **WHEN** a record's name is `A.J. Brown Jr.`
- **THEN** its short name is `A.J. Brown`

#### Scenario: A two-word name keeps a word that looks like a suffix

- **WHEN** a record's name is `Tony V`
- **THEN** its short name is `T V`

#### Scenario: A one-word name is unchanged

- **WHEN** a record's name is `Mahomes`
- **THEN** its short name is `Mahomes`

#### Scenario: A record in a late slot carries a short name

- **WHEN** a roster's second defender, filling the last D slot, is named `Fred Warner`
- **THEN** that record's short name is `F Warner`

### Requirement: A short name depends only on its own record's name

A record's short name SHALL be derived from that record's name alone, by the rules in *Every record
read from a roster carries a short name*. It SHALL NOT depend on any other record in the roster or in
any other roster, and it SHALL NOT depend on which slot the record fills.

Two or more records MAY carry the same short name. That SHALL NOT be an error, and neither record
SHALL fall back to its full name or to any longer form.

#### Scenario: Two starters who share an initial and surname keep the short form

- **WHEN** a roster's starters include `Jameson Williams`, `Javonte Williams`, and `Caleb Williams`
- **THEN** their short names are `J Williams`, `J Williams`, and `C Williams`

#### Scenario: A repeated name keeps the short form

- **WHEN** two starters carry different ids and the same name `Josh Allen`
- **THEN** reading succeeds and both records carry `J Allen` as their short name

#### Scenario: Moving a record between slots does not change another's short name

- **WHEN** a roster holds wide receivers `Josh Allen` and `Jaylen Allen`, and their two lines are
  swapped so each fills the other's slot
- **THEN** both records carry `J Allen` as their short name before and after the swap

## REMOVED Requirements

### Requirement: The first nine records are the starting lineup

**Reason**: Slots are assigned by position, not by file order, and there is no bench. A roster file
holds exactly nine records.

**Migration**: Use the starters in slot order per *A lineup is nine slots in a fixed order*. Trim any
roster file over nine records to its nine starters.
