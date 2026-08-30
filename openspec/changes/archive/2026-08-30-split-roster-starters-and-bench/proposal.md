## Why

`scripts/scores.sh` prints a roster as one flat list, which was right while every roster file held
exactly a starting lineup. Rosters are up to 26 players — the server already caps requests there —
and the moment a file holds more than nine, the flat list stops answering the question the tool
exists to answer: *what did my lineup score this week?* Today that means adding nine numbers by
hand off the terminal, which is the manual spreadsheet arithmetic this repo is meant to remove.

## What Changes

- **Split the printed roster into `STARTERS` and `BENCH`.** The first nine non-comment rows of the
  players file are the starters; every remaining row is bench. Both sections print in file order,
  the same order the flat list uses today.
- **Aggregate the starters into a `TOTAL`.** The total sums the points of starters the server
  returned stats for, and nothing else.
- **A starter with no stats contributes nothing and is not marked in the total.** It still prints
  `no stats` on its own line, directly above the total, where it is visible while reading the total.
  The alternative — an asterisk or an "8 of 9" count on the total — was considered and rejected as
  redundant with the lines above it.
- **Bench players show their points but get no total.** Bench points do not count for anything in
  the league; they are printed because "should I have started him" is most of why a bench is worth
  looking at, and the server already returns every requested player in the one existing request.
- **The `BENCH` header always prints, even when the bench is empty.** Every roster file in the repo
  is currently nine rows or fewer, so this is the common case today. An empty section is deliberate:
  it keeps the feature visible rather than letting it disappear until a roster grows.
- **Remove `scripts/players/quarterbacks.csv`.** It is a nine-quarterback comparison list, not a
  roster, and under a positional starters rule it would render as nine bogus starters with a
  meaningless total. It can be re-added when the script grows a way to distinguish a comparison list
  from a lineup.

Nine is a hardcoded constant, not a flag. The lineup is a positional convention and nothing
validates it: the players file carries no position column, so the script cannot know that rows 1–9
form a legal lineup, and reordering two lines silently changes the total. That is accepted as
temporary — the file *is* the lineup card, maintained by hand.

No change to the server, the scoring rules, or the request the script sends.

## Capabilities

### New Capabilities

- `roster-score-report`: How a saved roster file is turned into a readable weekly report — the
  starters/bench split, the starter total, how a player the server has no stats for is reported, and
  what the output does for rosters shorter than a full lineup. The script's behavior has never been
  specified; this change is the first to give it requirements worth writing down.

### Modified Capabilities

None. `player-week-score` covers the server's scoring of a player-week and is untouched: no new
stats, no new rules, no change to `POST /scores` or its response.

## Impact

- `scripts/scores.sh` — the print loop becomes two loops with a summing pass. The parse, request,
  and error paths are unchanged.
- `scripts/players/quarterbacks.csv` — deleted, along with the now-empty `scripts/players/`
  directory.
- `README.md` — the "Scoring a roster" section shows the flat output and describes the players file;
  both need updating, including the note that output is "in file order," which is now true within
  each section.
- No new dependencies. `curl` and `jq` still cover it, and the summation is one `jq` expression over
  the response already in hand.
- No floating-point rounding is needed. Every HMFFL award lands on a half — 0.5-yard increments,
  1.5 for a half sack — and halves are exact in binary floating point, so summing them yields no
  representation error.
- `scripts/scores.sh` has no test harness and this change does not add one. It is a local, manual
  tool run against a live server; the repo's TDD requirement governs the Go packages, which this
  change does not touch.
