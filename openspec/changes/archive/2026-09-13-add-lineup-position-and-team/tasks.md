## 1. Backfill the embedded lineup files

The old parser reads a header line as one more record and still passes every existing test, so the
data can move first and the parser can follow it.

- [x] 1.1 Fetch `/v1/players/nfl` into the scratchpad and, for every record in the eight files under
  `internal/lineup/data/`, look up `position` and `team` by id (a null team becomes `FA`). Keep each
  id and name exactly as written; don't take `full_name`, which drops suffixes. Don't commit the
  lookup script.
- [x] 1.2 Rewrite each file as `id,name,position,team` under a header row. Update its leading
  comment to describe the new format, and keep the `# bench` comments where they are.
- [x] 1.3 Check every value against the closed sets in `design.md` decision 4. Stop and ask if any
  record's position or team falls outside them.
- [x] 1.4 `go test ./internal/lineup/...` passes with the old parser.

## 2. Header row and columns by name

Add `Position` and `Team` to `Record` before any test reads them, so the first red is a value
mismatch rather than a compile error.

- [x] 2.1 Add empty `Position` and `Team` fields to `Record`, with label-only comments matching
  `Name`'s.
- [x] 2.2 Red: a test that `team,position,name,id` with `BUF,QB,Josh Allen,4984` reads id, name,
  position, and team by column name. Green: treat the first non-blank, non-comment line as the
  header, split lines on every comma, trim, and look fields up by name.
- [x] 2.3 Migrate every existing test input in `parse_test.go`, `read_test.go`, and `lineup_test.go`, the fixtures
  in `internal/lineup/testdata/`, and the inline lineup fixtures in `internal/web/matchup_test.go`,
  to a header plus four valid fields. Delete
  `TestParseSplitsOnFirstCommaOnly`, since its behavior is removed. Rewrite
  `TestParseMissingNameIsEmpty` as `4984,,QB,BUF`. The suite is green.
- [x] 2.4 Red → green: surrounding spaces in header names and in position and team values are
  trimmed.
- [x] 2.5 Red → green: comments and blank lines before the header are neither the header nor
  records, including an indented `#` line.

## 3. Header validation

- [x] 3.1 Red → green: a header missing a column is refused, naming the file and the missing column.
- [x] 3.2 Red → green: an unknown column (`postion`) is refused, naming the column.
- [x] 3.3 Red → green: a repeated column is refused, naming the column.
- [x] 3.4 Confirm, with a test, that an old-format file (`4984,Josh Allen` as the first non-comment
  line) is refused. If 3.2 already makes it pass, say so rather than forcing a separate red.
- [x] 3.5 Red → green: a header followed only by comments has no records and is refused, naming the
  file.

## 4. Field count

- [x] 4.1 Red → green: a record with fewer fields than the header is refused, naming the file and
  line number.
- [x] 4.2 Red → green: a name containing a comma (`1234,Smith, Jr.,WR,BUF`) is refused for having
  too many fields.
- [x] 4.3 Keep `,Josh Allen,QB,BUF` and `  , , ,  ` refused for having no id, and update the existing
  no-id tests to four-field inputs.

## 5. Closed sets

- [x] 5.1 Red → green: position `W R` is refused, naming the file, line number, and value. Introduce
  the position set from design decision 4 here.
- [x] 5.2 Red → green: `wr` and an empty position are refused (exact case, no empty value).
- [x] 5.3 Add a test that `DE` and `DL` are both accepted and carried as written.
- [x] 5.4 Add a test that `OL` and `P` are refused.
- [x] 5.5 Red → green: team `BUFF` is refused, naming the file, line number, and value. Introduce
  the 32-code set plus `FA`.
- [x] 5.6 Red → green: `buf`, `OAK`, and an empty team are refused. Add a test that `FA` is
  accepted.

## 6. Labels stay labels

- [x] 6.1 Add a test that a record whose position is not a legal lineup slot for its row doesn't
  change the split: nine `QB` starters and a `K` in tenth place gives nine starters and a benched
  kicker.
- [x] 6.2 Add a test that a record whose name, position, and team belong to a different player, with
  all values in their sets, reads successfully as written.
- [x] 6.3 Confirm that `TestEveryEmbeddedWeekIsAReadableMatchup` fails on bad data: temporarily set
  one embedded record's team to `BUFF`, see the walk go red naming that file, then revert.

## 7. Documentation

- [x] 7.1 Amend ADR 0004 decision 9 in place:
  - the header row and the four column names
  - the closed position set as Sleeper spells it (with the defensive inconsistency noted) and the
    team set with `FA`
  - values looked up in the players index at write time
  - the future-column rule: backfill in the same change, or explicitly optional
  
  Remove the field-order line from its open items, and any stale reference to `scripts/scores.sh`.
- [x] 7.2 Remove the `backlog.md` line about growing the lineup CSV and `scores.sh`.
- [x] 7.3 Update the comment on `Lineup` in `internal/lineup/lineup.go`, which says a lineup "carries
  no position column".

## 8. Verify

- [x] 8.1 `go vet ./...` and `go test ./...` pass.
- [x] 8.2 `openspec validate add-lineup-position-and-team --strict` passes.
