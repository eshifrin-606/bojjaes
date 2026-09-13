## Context

A roster file is a hand-maintained `id,name` list. `parse` in `internal/lineup/parse.go` splits each
line on its first comma, and `Tree.Read` is the only reader now that `scores.sh` and
`scripts/teams/` are gone. There are eight files under `internal/lineup/data/`, all in the old
format. `TestEveryEmbeddedWeekIsAReadableMatchup` reads every one of them through `Tree.Read`.

In practice, files are written by Claude from pasted player names. The likeliest error is not a
misspelling but a plausible, valid, stale value remembered instead of looked up.

## Goals / Non-Goals

**Goals:**

- Position and team are in every roster file, frozen at the time it was written.
- A typo in a position, a team, or the column layout fails the file rather than reaching the page.
- A future column can be added without rewriting how every old file is read.

**Non-Goals:**

- Showing position or team on the page. That is a separate change; `Record` gains the fields and
  nothing reads them yet.
- Checking labels against Sleeper at read time. The id stays the only identity, and old weeks must
  not change when Sleeper does.
- A generator script. The backfill is done once by hand-run lookup; a committed tool can come later.
- Correct historical 2025 teams.

## Decisions

### 1. Header row, fields read by column name

**Chosen:** the first non-blank, non-comment line names the columns, and each record's fields are
looked up by name.

**Alternatives:**

- *Positional four fields* (`id,name,position,team` with no header). Less to parse, but every future
  column is breaking for every file, and a missing comma silently moves values into the wrong
  column.
- *A format version marker* (`# format: 2`). The parser would have to keep every past format
  forever. The header already describes the file, and backfilling is our policy.
- *JSON or YAML.* Sturdy to change, but worse to write by hand, which is how these files are made.

The header costs one line per file, and adding it now is free because every file is being rewritten
anyway.

### 2. Unknown and repeated columns are refused

Ignoring an unknown column would turn a misspelled `postion` into a file whose records all lack a
position. This matches the parser's existing stance: a refused file is loud, a skipped line is not.
When a future column is added, the known-column set grows in the same change (see decision 6).

### 3. Keep the hand-rolled splitter; don't use `encoding/csv`

`encoding/csv` supports quoting, but its `Comment` only matches `#` in column one, so indented
comments would break. Trimming would also need a second pass, and `FieldsPerRecord` errors don't
fit our "file and line number" messages. Quoting is only needed for names containing commas, and
Sleeper names don't contain them. So lines split on every comma, and a comma in a name is refused
by the field-count check. This drops the old `Smith, Jr.` allowance.

### 4. Closed sets live in `internal/lineup` as Go literals

**Position set:** `QB RB FB WR TE K DL DE DT NT LB OLB ILB DB CB S SS FS`. This is every Sleeper
`position` value a scoring player could carry, taken from the 2026-09-13 players index. It excludes
`OL`/`G`/`T`/`C`/`OT`/`OG`, `P`, `LS`, `DEF`, `ATH`, and `K/P`, which can't score in this league.
It is a subset of Sleeper's vocabulary rather than the whole thing so that a line-shift or typo
landing a non-position in the column is caught.

**Team set:** the 32 current codes plus `FA`. Sleeper's index still carries `OAK`, which is left out.

**Why in `lineup` and not `sleeper`:** `internal/sleeper` must not be imported by serving code, and
the lineup files are provider-neutral data that happen to use Sleeper's spelling. The sets are a
copy of that spelling, which is stated once where they're defined.

**Accepted consequence:** Sleeper's defensive positions are inconsistent (`DL` for Harmon and Pearce,
`DE` for Garrett and Crosby, `LB` for Oweh, `DB` for Watts). The file carries these as written, per
the user's choice. Whether the page groups them is a question for the layout change.

### 5. Backfill from the players index, done once, not committed

For each existing file, keep the id and the name as already written. The name isn't taken from
Sleeper, because Sleeper's `full_name` drops suffixes (`Luther Burden`, `Will McDonald`) that the
short-name rules rely on. `position` and `team` come from `/v1/players/nfl`. A null team becomes
`FA`. The lookup script lives in the scratchpad; the output is the commit.

The same procedure applies to future files: look values up in the index when writing, never from
memory.

### 6. A rule for future columns

Recorded in ADR 0004 decision 9: a change that adds a column either backfills it in every existing
file in the same change, or makes it explicitly optional, with an absent value that has a stated
meaning. An optional column must not become required later for the week it was introduced, since
that would put a date-based branch in the format.

## Risks / Trade-offs

- [A valid but wrong value, e.g. `BAL` for a `BUF` player] → Not catchable by a closed set. The
  mitigation is procedural: values are looked up in the index at write time (decision 5).
- [Sleeper introduces a new position spelling for a rostered player] → That file is refused at test
  time, not at request time, because every embedded file is parsed in CI. Fix: add the value to the
  set.
- [A team relocates or changes its code] → Same failure and fix. Old files keep the old code, which
  must then stay in the set.
- [2025 teams reflect later moves] → Accepted; the app didn't exist then.
- [Required columns make hand-writing a file harder] → The same lookup that finds ids gives position
  and team.

## Migration Plan

One commit containing the parser, tests, test fixtures, all eight data files, the ADR amendment, and
the backlog edit. The embedded tree ships with the binary, so there is no window where the deployed
reader and deployed files disagree. Rollback is a revert.
