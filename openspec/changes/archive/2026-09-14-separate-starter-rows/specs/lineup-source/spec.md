## REMOVED Requirements

### Requirement: Colliding short names fall back to the full name

**Reason**: A short name exists to keep a name on one line in a narrow column. Falling back to the
full name on a collision brings back the wrapping the short name was meant to avoid, and it only
happens in the rare case where two starters share an initial and surname. The matchup page now
shows position and team under every name, and that usually tells such players apart.

**Migration**: None needed for data. Short names are display text and nothing identifies a player
by them. Two records that used to carry their full names as short names now carry the same derived
short name.

## ADDED Requirements

### Requirement: A short name depends only on its own record's name

A record's short name SHALL be derived from that record's name alone, by the rules in *Every record
read from a roster carries a short name*. It SHALL NOT depend on any other record in the roster or in
any other roster, and it SHALL NOT depend on whether the record is a starter or on the bench.

Two or more records MAY carry the same short name. That SHALL NOT be an error, and neither record
SHALL fall back to its full name or to any longer form.

#### Scenario: Two starters who share an initial and surname keep the short form

- **WHEN** a roster's starters include `Jameson Williams`, `Javonte Williams`, and `Caleb Williams`
- **THEN** their short names are `J Williams`, `J Williams`, and `C Williams`

#### Scenario: A repeated name keeps the short form

- **WHEN** two starters carry different ids and the same name `Josh Allen`
- **THEN** reading succeeds and both records carry `J Allen` as their short name

#### Scenario: Moving a record across the starter split does not change another's short name

- **WHEN** a roster's second record is `Josh Allen`, and `Jaylen Allen` is moved from the tenth
  record to the ninth
- **THEN** both records carry `J Allen` as their short name before and after the move
