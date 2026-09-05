## Context

`internal/roster` resolves a roster when the caller already knows the team: `Tree.Path` builds
`<root>/<season>/<week>/<team>.csv`, `Tree.Read` opens and parses it. Nothing in the package looks
at a directory. The team name arrives from an argument — `scores.sh` takes one, `fantasycast.sh`
takes one or two.

The backlog's `GET /{season}/{week}` page has no such argument: the URL carries two path segments
and the page must render two columns. The information is already on disk. Each of the three weeks
in `scripts/lineups/2025/` holds exactly two files, one of them `bojjaes.csv`.

Constraints that shape the design:

- **The lineup tree's layout is stated in exactly one place.** `roster-source` says callers do not
  construct paths themselves. A handler that ran `os.ReadDir` on a path it built would put a second
  copy of the layout outside the package.
- **The week directory is hand-maintained, like the files in it.** Nobody generates it. A week can
  be staged early, a file can be left behind from an edit, and the scoring server is the first
  thing that would ever notice.
- **A wrong matchup is invisible in the output.** The page renders two columns of real players with
  real points either way. Unlike a bad roster line, there is no downstream check that would catch
  a matchup resolved against the wrong opponent.
- **The tree will move under a package for `//go:embed`.** Directory listing must work the same
  against whatever it moves to, so nothing here may assume an OS path beyond what `Tree` already
  holds.

## Goals / Non-Goals

**Goals:**

- Resolve a season and week to two team names, using only the week directory.
- Make every non-matchup directory a distinguishable, named error rather than a guess.
- Keep the layout knowledge inside `internal/roster` alongside `Path`.
- Behaviour driven out test-first, one behaviour at a time, per the repo's TDD convention.

**Non-Goals:**

- Reading or parsing rosters during resolution. `Read` already does that, and its errors are about
  files, not directories.
- Any HTTP surface, handler, template, or route. The page is the next change.
- Changing `scripts/scores.sh` or `scripts/fantasycast.sh`. Neither takes its opponent from the
  directory, and neither needs to before the page replaces them.
- Discovering which *weeks* exist, or listing seasons. A different question about a different
  directory level, with no caller yet.
- Anything about the position and team columns the CSV is going to grow.

## Decisions

### Resolution returns two team names, not two rosters

`Matchup(season, week int) (ours, theirs string, err error)` — or a small named pair — hands back
names. The caller then calls `Read` twice, which it already knows how to do.

The two questions have different subjects and different failure modes. "Is this week a matchup" is
about a directory: the wrong number of files, or no Bojjaes among them. "Is this roster usable" is
about a file: a line with no id, a duplicate id, no records at all. Folding them into one call
gives the caller a single error value covering both, so it cannot tell a week that is not staged
yet from a week whose opponent has a typo on line 7 — and those are fixed in different ways by
different people.

Keeping them apart also keeps resolution cheap and its tests honest. `Matchup` never opens a file,
so its tests seed empty files and assert on directory shape alone; nothing in a matchup test can
fail because a fixture's records changed.

*Alternative considered:* returning `Matchup{Ours, Theirs}` where each side carries a name and an
already-parsed `Roster`. Rejected for the reasons above. It is also strictly additive later: the
names-only call is the thing a rosters-returning wrapper would be built on, so choosing names now
does not foreclose it.

*Alternative considered:* returning just the opponent's name, since the caller knows ours is
`bojjaes`. Rejected — it puts the string `"bojjaes"` back at the call site, which is the knowledge
this change is trying to keep in one place, and it loses the ordering guarantee that gives the page
its left column for free.

### `bojjaes` is a package constant

An unexported constant in `internal/roster`, commented as our team, is what `Matchup` compares
against. Not a `Tree` field, not a parameter.

The repo is a tool for one team; `CONTEXT.md` names it, and the matchup-report spec already treats
the Bojjaes as a distinguished team rather than one of N. A parameter would be one every call site
fills in identically, which is a worse place for the name than a constant — a caller that passed
the wrong string would resolve a plausible matchup with our column missing.

*Alternative considered:* `New(root, ourTeam string)`. Rejected as league-generality nothing asks
for. If a second team's tooling ever appears, promoting a constant to a field is a small, obvious
edit with a compiler-enforced blast radius.

### Only `.csv` regular files count, and dotfiles never do

The count that decides "exactly two" is over the directory's regular files whose name ends `.csv`
and does not begin with `.`. Subdirectories and anything else are skipped.

This is not tidiness — it is what keeps the rule usable on a hand-maintained tree edited on macOS.
A `.DS_Store` appears in any directory opened in Finder, and would silently make every week a
three-file error. The dotfile exclusion also covers an editor's `.wood.csv.swp`.

The filter deliberately does not extend to non-dot junk: a `notes.md` is skipped because it is not
a `.csv`, but a `wood.csv.bak` is not a `.csv` either and is likewise skipped, while a genuine
third roster is counted and refused. The line is drawn at the extension, so the rule stays "the
roster files in this directory", which is a sentence someone editing the tree can hold in their
head.

### Three files is an error, and the error names all three

There is no rule by which two of three roster files are the matchup. Picking `bojjaes.csv` plus
whichever of the other two sorts first would succeed loudly and be wrong silently: the page would
render two columns of real players with real points, for a game nobody is playing.

The error text lists the files found, because the fix is always "which of these does not belong",
and the person fixing it is looking at a browser, not the directory.

### Resolution reports its four refusals distinguishably

Too many rosters, too few, no directory, and two rosters with no `bojjaes.csv` are four different
mistakes with four different fixes. Each error names the directory and says which case it hit.
Whether these are sentinel errors or distinct messages is an implementation choice for the tasks;
what the spec pins is that they are distinguishable and that the directory is named.

A missing directory is reported as "this week is not staged" rather than surfacing a bare
`os.ReadDir` `ENOENT`, because the caller asked about a week, not a path.

### The directory path comes from the same place `Path` gets it

`Matchup` derives `<root>/<season>/<week>` through the package's own layout knowledge, so the week
directory and the roster files inside it can never disagree about where the tree is. `Path` gains,
or is refactored onto, a week-directory step; season and week are integers, so the team-name
segment check `Path` performs has no equivalent here.

## Risks / Trade-offs

- **Every week must now hold exactly two rosters** → True of all three weeks in the tree today, and
  the change turns a coincidence into a checked rule. A bye week or a partially staged week will
  fail loudly. That is the intended trade: the failure is a named error at resolve time instead of
  a wrong opponent rendered as fact.
- **The `.csv` filter could hide a real roster** → A roster file saved as `wood.CSV` or `wood.txt`
  would be skipped, and the week would fail as one-roster rather than resolving. The failure is
  loud and names the directory, so the fix is visible; the alternative — counting every file —
  breaks on `.DS_Store`, which is far more likely.
- **`bojjaes` hardcoded** → Named constant, one comparison, one place. Cheap to promote to a field
  if a second team's tooling ever exists.
- **Names-only means the caller can still read the wrong file** → It cannot, in practice: the names
  come from `filepath` entries in the tree and go straight back through `Path`. But the round trip
  through a string is real, and it is why the returned names keep the Bojjaes first rather than
  relying on the caller to re-identify them.
- **Case sensitivity differs between macOS and the deploy target** → `BOJJAES.csv` matches on a
  case-insensitive filesystem's lookup but not a byte comparison against the constant. Comparison
  is exact and lowercase; every file in the tree is lowercase, and a mixed-case one fails as
  no-bojjaes rather than resolving on one machine and not another.

## Migration Plan

Additive: a new function on an existing type, no existing caller, no deployed behaviour. Nothing to
roll back. The first consumer is the `GET /{season}/{week}` handler in a later change.

## Open Questions

- Does a bye week ever exist in this league? If it does, "exactly two" needs a companion rule for
  the week directory that holds one roster on purpose. Nothing in the tree suggests it yet, and
  guessing at it now would weaken the check that catches a half-staged week.
- When the tree moves under a package for `//go:embed`, directory listing goes through `fs.FS`
  rather than `os`. That is a change to how `Tree` reaches the tree, not to this rule, and it is
  deferred with the move.
