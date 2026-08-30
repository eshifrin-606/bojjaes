## Context

`scripts/scores.sh` parses a players file into parallel `ids`/`names` arrays, posts every id in one
request to `POST /scores`, and then walks the arrays printing one padded line per player. Points are
pulled per player with a `jq` query against the response held in a shell variable; a player the
server put in `no_stats` falls through `//` to the string `no stats`.

Two properties of the existing script make this change small. The response is fetched once and kept
in a variable, so a summing pass costs no extra request. And the arrays preserve file order, so
"first nine records" is already expressible as an index comparison — no re-parse, no sort.

The constraint that shapes the decisions below is the server's `no_stats` contract, documented at
length in `README.md`: an absent stat line is genuinely ambiguous between a game that has not
kicked off, an inactive player, a typo'd id, and an unplayed week. The script's job is to keep that
ambiguity visible, and a total is the one place it could quietly disappear.

## Goals / Non-Goals

**Goals:**

- Answer "what did my lineup score" without the reader adding nine numbers off the terminal.
- Keep the `no_stats` distinction legible once a total exists above it.
- Stay within `curl` and `jq`, one request, no new dependency.

**Non-Goals:**

- No lineup validation. The file has no position column and this change does not add one.
- No configurability. Nine is a constant; there is no flag, no env var, no per-file override.
- No test harness for the script. It is a manual tool run against a live server.
- No server change. `POST /scores` and its response are untouched.

## Decisions

### The starter total sums only scored starters, and carries no marker

A starter in `no_stats` prints `no stats` and adds nothing. The total below it is a plain number.

*Alternative considered:* annotate the total — `72.5*`, or `72.5 (8 of 9)`. Rejected as redundant.
The report is nine lines tall and the absent starter's own line sits directly above the total, in
the same glance. An annotation would restate what is already on screen, and would need its own
explanation in the README for a case the reader can already see.

*Alternative considered:* refuse to total when a starter is missing. Rejected because absence is
routine — a player whose game has not kicked off produces it every Sunday afternoon — and a report
that withholds the number for most of the day is worse than one that shows a running number beside
a visible gap.

This is a real trade-off, not a free choice; see Risks.

### Nine is a hardcoded constant

The lineup size lives as one named constant near the top of the script.

*Alternative considered:* a `--starters N` flag. Rejected: it invites the question of what `N` means
for a file that is not a lineup, which is exactly the confusion this change removes by deleting
`scripts/players/quarterbacks.csv`. A flag can be added the day a second lineup size exists.

### The bench heading prints unconditionally

Every players file in the repo today is nine rows or fewer, so suppressing an empty bench would mean
the sections effectively never appear, and the feature would sit unexercised until a roster grows.
Printing the empty heading keeps it visible. This is a deliberate reversal of the usual instinct to
hide empty sections, and it is expected to be revisited once rosters routinely exceed nine.

### Summation happens in `jq`, over the response already in hand

The starter ids are passed to one `jq` expression that selects their scores from the response and
sums the points. Doing it in `jq` rather than accumulating in the shell avoids `bash`'s lack of
floating-point arithmetic, which would otherwise force `bc` or `awk` — a new dependency for a script
whose documented requirements are exactly `curl` and `jq`.

Rounding is not a concern. Every HMFFL award lands on a half: the rushing/receiving increment is
0.5 and a half sack pays 1.5. Halves are exactly representable in binary floating point, so a sum of
them is exact and prints cleanly. This holds only as long as the rules pay in halves; a future
third-point rule would reopen it.

### The per-player print stays as it is

Both sections use the existing `printf '%-24s %s\n'` line. Bench players are scored and printed the
same way starters are — the server returns them in the same response, so showing their points costs
nothing and answers "should I have started him."

## Risks / Trade-offs

- **An unmarked total is a lower number than the lineup actually scored, and nothing in the total
  itself says so.** → The absent starter's `no stats` line is directly above it, and the README
  already explains why absence is not zero. Accepted knowingly; revisit if the report ever grows
  long enough that the total and the player lines stop fitting in one glance.
- **The starters are whichever nine rows come first, with nothing checking the lineup is legal.**
  Reordering two lines silently changes the total. → Accepted as temporary. The file is a hand-kept
  lineup card; positions are not modelled anywhere in the repo yet.
- **Deleting `scripts/players/quarterbacks.csv` loses a comparison list.** → It is nine ids and
  names, trivially reconstructed, and the git history keeps it. It is removed because under a
  positional starters rule it renders as nine bogus starters with a meaningless total.
- **The script remains untested.** → Unchanged from today, and scoped: the logic added here is
  presentation and one sum, while every scoring rule stays in the tested Go packages.
