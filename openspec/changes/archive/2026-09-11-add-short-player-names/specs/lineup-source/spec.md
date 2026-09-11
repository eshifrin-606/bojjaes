## MODIFIED Requirements

### Requirement: The id is authoritative and the name is a label

The id SHALL be the only field used to identify a player to anything outside the roster. The name
SHALL be carried as display text only and SHALL NOT be used to look up, match, or validate a
player. The short name derived from it SHALL be display text on the same terms, and nothing SHALL
identify a player by it.

A record MAY have an empty name. Reading SHALL NOT reject it, invent a placeholder, or fall back to
the id, so that a roster written without labels stays readable as written. A record with an empty
name SHALL have an empty short name.

#### Scenario: A record with no name is read

- **WHEN** a record is written as `4984,` or as `4984`
- **THEN** the record is read with id `4984`, an empty name, and an empty short name

#### Scenario: The name is not checked against anything

- **WHEN** a record pairs a valid id with a name that belongs to a different player
- **THEN** reading succeeds and the record carries the name as written

## ADDED Requirements

### Requirement: Every record read from a roster carries a short name

Every record in a lineup read from a roster SHALL carry a short name, starters and bench alike. The
short name SHALL be derived from the record's name when the roster is read. It SHALL NOT be read
from the file, and the roster file format SHALL NOT change to carry it. The short name SHALL be
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

#### Scenario: Bench records carry a short name

- **WHEN** a roster holds ten records and the tenth is named `Dak Prescott`
- **THEN** the bench record's short name is `D Prescott`

### Requirement: Colliding short names fall back to the full name

Collisions SHALL be resolved within each group of a lineup separately: the starters against the
other starters, and the bench against the other bench records. When two or more records in one
group would derive the same non-empty short name, each of those records SHALL carry its full name
as its short name. Records whose derived short name is unique in their group SHALL keep it.

A record in one group SHALL NOT cause a record in the other group to fall back. Because the groups
are positional, moving a record across the starters/bench split MAY change the short name of another
record. Collisions SHALL NOT be considered across different rosters: each team's names are shown
under that team.

A collision SHALL NOT be an error. It is a repeated label, and a repeated name is not an error
either.

#### Scenario: Two starters who share an initial and surname show their full names

- **WHEN** a roster's starters include `Jameson Williams`, `Javonte Williams`, and `Caleb Williams`
- **THEN** their short names are `Jameson Williams`, `Javonte Williams`, and `C Williams`

#### Scenario: A bench record does not force a starter to its full name

- **WHEN** a roster's second record is `Josh Allen` and its tenth record is `Jaylen Allen`
- **THEN** both records carry `J Allen` as their short name

#### Scenario: Two bench records who collide show their full names

- **WHEN** a roster's tenth record is `Jaylen Allen` and its eleventh is `Jordan Allen`
- **THEN** both records carry their full names as short names

#### Scenario: Moving a record into the starters can change another's short name

- **WHEN** a roster's second record is `Josh Allen`, and `Jaylen Allen` is moved from the tenth
  record to the ninth
- **THEN** both records carry their full names as short names

#### Scenario: A repeated name is read with its full name as its short name

- **WHEN** two starters carry different ids and the same name `Josh Allen`
- **THEN** reading succeeds and both records carry `Josh Allen` as their short name

#### Scenario: Another roster's names do not cause a collision

- **WHEN** one roster holds `Josh Allen` and a different roster holds `Jaylen Allen`
- **THEN** each record's short name is `J Allen`
