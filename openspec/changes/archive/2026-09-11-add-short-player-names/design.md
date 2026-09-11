## Context

`internal/lineup` turns a roster CSV into `[]Record{ID, Name}`. `parse` handles the file syntax
(splitting, trimming, and refusing bad lines and duplicate ids). `Tree.Read` locates the file,
calls `parse`, and wraps the result in a `Lineup`. `Starters()` and `Bench()` are slices of that one
record list.

`fit-matchup-page-to-phone-width`, being proposed in parallel, will render `Record.ShortName` on
narrow cards. The contract between the two changes is fixed: the field is named `ShortName`, every
record from `Tree.Read` has it filled, it is non-empty whenever `Name` is, and it is display text
only.

In the embedded tree today, every name has two or three words. Two carry a suffix (`Luther Burden
III`, `Will McDonald IV`), one first name is hyphenated (`Amon-Ra St. Brown`), and one is written
as initials (`KC Concepcion`, on the fuego bench). Exactly one roster has a short-name collision:
`2026/1/fuego.csv`, where `Jameson Williams` and `Javonte Williams` both derive `J Williams`. Both
are starters. No bench has a collision.

## Goals / Non-Goals

**Goals:**

- Every `Record` returned through `Tree.Read` carries a `ShortName` built from the rules in the
  `lineup-source` delta.
- Among the starters, no short name is ambiguous; likewise among the bench.
- `Name`, the CSV format, and `parse` behave exactly as they do today.

**Non-Goals:**

- Anything in `internal/web`, the template, CSS, or the `matchup-page` spec, including deciding
  when to show `ShortName`.
- `scripts/scores.sh`, which keeps printing `Name`.
- An override column or file for hand-picked short names.
- Particles and compound surnames (`Van Noy`, `St. Brown`): the rest of the name is kept as
  written, so these survive intact rather than being cut down further.
- Suffixes beyond `Jr`/`Sr`/`II`–`V`, lowercase suffixes (`jr.`), and suffixes after a comma
  (`Smith, Jr.`). These are not in any roster, and a suffix that isn't recognized is simply kept.
- Keeping internal whitespace as written. Runs of spaces collapse to one in the short form only.
- Recognizing every way initials can be written. Only the two forms under "Initials" below count;
  any other first word gets a single initial.
- Stripping punctuation from a first name that is not initials (`Ja'Marr` gives `J`).
- Guaranteeing the short name is shorter. For a one-word name, an initials-style first name, and
  after a collision fallback, it can be the same length as `Name`.

## Decisions

### Collision rule: every colliding record falls back to full `Name` (confirmed)

Two records in one group derive the same short name (fuego: `J Williams` ×2).

- **(a) Both fall back to full `Name`** — *chosen*
  - \+ One rule, no new abbreviation style; the output is always either the standard short form or
    the real name.
  - \+ Never ambiguous, including with identical first names (two `Josh Allen`s).
  - \+ Three-way collisions need no extra logic.
  - − The colliding rows are the long rows the change exists to avoid, so on a phone they may still
    wrap. That happens in 1 of 8 rosters today, for 2 of 9 starters.
  - − Looks inconsistent next to short names in the same card.
- **(b) Extend the first-name prefix until the names differ** (`Jam Williams` / `Jav Williams`)
  - \+ Stays short, so no wrapping.
  - − Nonstandard abbreviations that readers have to decode.
  - − More rules, and identical or prefix-equal first names (`Josh` and `Joshua`) still need a
    fallback to (a).
- If the fallback rows wrap in practice, (b) can be layered on later without changing the field or
  the scope.

### Collision scope: starters among starters, bench among bench (confirmed: starters only)

The starters (the first nine records, `Lineup.Starters()`) are checked for collisions only against
each other. A bench player can't push a starter back to the long name.

The bench still has to carry a non-empty `ShortName` under the fixed contract. The choices were to
derive bench short names with no collision check, or to run the same collision pass over the bench
on its own.

- **Same pass, run once per group** — *chosen*
  - \+ One function, `fillShortNames(group)`, called twice; no second code path for the bench.
  - \+ No ambiguous short names within whatever group is shown together, if the bench is ever
    rendered.
  - − The bench pass does work that no page uses today.
- **No collision check on the bench**
  - \+ Marginally less work.
  - − A second rule to state and test, and it would leave two `J Williams` on a rendered bench.

Consequence, accepted: the groups are positional, so moving a record across line 9 can change
*another* record's `ShortName`. With `Josh Allen` starting and `Jaylen Allen` tenth, both are
`J Allen`; move `Jaylen Allen` up to ninth and both become their full names. That matches how the
split already behaves: reordering the card changes who starts.

Cross-roster collisions are out of scope, because the team header separates them.

### Initials: a first name written as initials is kept as written (confirmed)

`A.J. Brown` stays `A.J. Brown` and `KC Concepcion` stays `KC Concepcion`. Cutting them to `A Brown`
and `K Concepcion` produces names that nobody uses.

A first word counts as initials when either:

- it contains a period (`A.J.`, `T.J.`, `J.`), or
- it is exactly two letters, and both are uppercase (`KC`, `DJ`, `DK`, `TJ`, `JK`).

That rule is deliberately narrow:

- `Al` is two letters but not both uppercase, so it's a name (`Al Smith` → `A Smith`).
- `CeeDee` has capitals but isn't all capitals (`CeeDee Lamb` → `C Lamb`).
- Three or more capitals with no period (`JOE`) are not treated as initials. The two-letter limit
  covers every real no-period initials name we know of, and a three-capital word is as likely to be
  a name typed in capitals as initials.
- Hyphens and apostrophes don't count (`Amon-Ra`, `Ja'Marr` contribute one letter).

The suffix rule still applies to an initials-style name: `A.J. Brown Jr.` → `A.J. Brown`. The two
rules look at different ends of the name, and keeping them independent means neither has a special
case for the other.

### Suffix: dropped only when the name has three or more words (confirmed)

`Tony V` stays `T V`. Dropping the suffix from a two-word name would leave only the initial.

### Derive in `Tree.Read`, not in `parse`

- `parse` stays about file syntax, and its tests keep comparing `Record{ID, Name}` literals
  unchanged.
- The collision pass needs whole groups, which `Read` has once it builds the `Lineup`.
- − A `Lineup` built directly from `[]Record` (as `lineup_test.go` does) has no short names. That's
  acceptable, because the contract is scoped to reading.

### Shape: two unexported functions

- `shortName(name string) string` derives the short name for a single name.
- `fillShortNames(group []Record)` sets every record's `ShortName` in the group, counts duplicates,
  and applies the fallback. It knows nothing about starters or bench.
- `Read` builds the `Lineup`, then calls `fillShortNames(l.Starters())` and
  `fillShortNames(l.Bench())`. Both are slices of the lineup's one record list, so filling them in
  place fills the lineup, and the starters/bench boundary stays defined in exactly one place.
- The splitting uses `strings.Fields`, which handles surrounding and repeated whitespace. The
  initial is the first rune of the first field, not its first byte.
- Neither function is exported. The contract is the field, not the derivation.

## Risks / Trade-offs

- **[A suffix rule drops a real surname word ending in `V`/`II`]** → The rule applies only to names
  of three or more words and to an exact token match. No roster has such a name. If one appears,
  its short name loses a word but its `Name` is intact.
- **[The collision fallback still wraps on phones]** → Accepted under (a). It's visible, rare, and
  option (b) is the known follow-up.
- **[A two-capital first name that isn't initials]** → It would be kept whole, which is only a
  longer label, never a wrong one.
- **[Reordering the card changes another record's short name]** → Accepted; see collision scope.
- **[Existing `Read` tests compare whole `Record` values]** → Their expected values gain
  `ShortName`. This is the red step that wires the derivation into `Read` (see tasks).
- **[The parallel change depends on this one]** → The field name and population rule are fixed by
  contract. None of the decisions above changes the field's type or where it's filled.
