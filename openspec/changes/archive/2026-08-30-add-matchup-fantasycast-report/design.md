## Context

`scripts/scores.sh` already turns one roster file into a full report: it resolves a team name to a
file under `scripts/lineups/<season>/<week>/`, parses `id,name` records, posts every id to
`POST /scores` in one request, and prints starters, a starter total, and bench. The matchup report
needs exactly that, twice, arranged in columns.

Two facts constrain how the two tools can share work.

**The shell is bash 3.2.** macOS ships `/bin/bash` 3.2.57, which has no namerefs (`declare -n`) and
no associative arrays. A sourced library therefore cannot hand two rosters' parallel `ids`/`names`
arrays back to a caller without `eval` gymnastics. Any factoring that wants "parse both files into
`ids_a` and `ids_b`" fights the shell.

**The two lineup files are not position-aligned.** `bojjaes.csv` runs WR WR WR TE RB QB K IDP IDP;
`wood.csv` runs WR TE RB RB RB QB K IDP IDP. The roster file carries no position column — a
deliberate decision recorded in the `split-roster-starters-and-bench` design — so row *n* on the
left and row *n* on the right are not a positional matchup, and nothing here should imply they are.

## Goals / Non-Goals

**Goals:**

- Answer "how am I doing against them" in one screen, without running two commands and holding both
  lineups in your head.
- Add zero duplicated roster, scoring, or report logic. Whatever `scores.sh` does today, it stays
  the only place that does it.
- Keep the dependency set as it is: `curl` and `jq`.

**Non-Goals:**

- No positional matchup. Columns are two independent lists that happen to sit side by side.
- No shared bash library. See the first decision.
- No scoreboard, margin, or winner.
- No server change, and no schedule data — the caller names the opponent.
- No test harness for either script, unchanged from `split-roster-starters-and-bench`. See Open
  Questions.

## Decisions

### `fantasycast.sh` invokes `scores.sh` as a subprocess, twice

The shared unit is the *process*, not a sourced function. `fantasycast.sh` resolves its arguments,
runs `scores.sh <season> <week> <team>` once per column, captures each report as a string, and lays
the two strings out in columns.

*Alternative considered:* a sourced `scripts/lib/roster.sh` holding `roster_path`, `roster_ids`,
`fetch_scores`, and `report_roster`. Rejected for this change. Under bash 3.2 the only clean
factoring is a self-contained "print one team's block" function using `local -a` internals — which
is precisely what a `scores.sh` invocation already is, minus a fork. The library would buy a saved
process and cost a new file, a new sourcing path, and a refactor of a working script.

*Consequence accepted:* two `POST /scores` requests instead of one, and therefore two upstream
Sleeper fetches of roughly half a megabyte each. This is the right side of the trade anyway: the
26-id cap in `internal/score/batch.go` then applies per roster rather than per matchup, so two full
26-player rosters work, where a merged request would fail at 52.

### The season/week line moves out of `scores.sh` entirely

`scores.sh` drops the line and begins at `STARTERS`. `fantasycast.sh` prints it once above both
columns.

*Alternative considered:* keep the line in `scores.sh` and have `fantasycast.sh` strip the first two
lines of each capture. Rejected as brittle — it couples the matchup tool to the exact line count of
another script's preamble, and a later change to that preamble would silently eat a player.

*Alternative considered:* an env var or flag suppressing the header. Rejected as a configuration
knob for a one-line difference; the header was never part of the `roster-score-report` spec, and the
season and week are the caller's own arguments, which the caller can print.

This is a visible behavior change for anyone running `scores.sh` directly, and the README's example
output changes with it.

### Columns are laid out in bash, not by `paste`

`fantasycast.sh` reads each captured report into an array, computes the widest line of the left
report, and prints `printf '%-*s%s\n'` per line with a two-space gutter, iterating to the longer of
the two line counts and substituting an empty string past the end of the shorter.

*Alternative considered:* `paste -d '\t'`. Rejected: `paste` delimits with a tab, and tab stops do
not fall where a 24-character name field ends, so the right column would jitter. Padding every left
line to a fixed width first — which is what makes the layout work — leaves `paste` doing nothing a
`printf` loop was not already doing.

*Alternative considered:* `awk` merging two files. Rejected only to hold the dependency line at
`curl` and `jq`; it would otherwise be a fine implementation.

Reading a captured report into an array under bash 3.2 uses `while IFS= read -r line` over a
here-string. `mapfile`/`readarray` are bash 4 and must not appear. `IFS=` and `-r` matter: the
report contains blank separator lines and padded fields, and both must survive intact.

The left width is computed rather than fixed, because `scores.sh` pads names to a *minimum* of 24
characters and does not truncate — a longer name widens its own line, and a fixed gutter offset
would let that one line push into the right column.

### `scores.sh` gains a fixed-width points field

`printf '%-24s %s\n'` becomes a right-aligned fixed-width points field. This is what makes the two
totals and every pair of player lines read as a column rather than as ragged text, and it is
required by the matchup report's alignment requirement.

It also mildly improves `scores.sh` on its own: points line up under each other instead of starting
wherever a name happens to end.

### No scoreboard line

The two `TOTAL` lines land on the same output line, facing each other. Nothing computes the
difference.

*Alternative considered:* `BOJJAES 72.5 — WOOD 88`, or a margin. Rejected on the same reasoning that
kept an annotation off the single-roster total: a starter in `no_stats` is indistinguishable from one
whose game has not kicked off, so a margin printed at 1pm Sunday reads as a settled deficit when it
is nothing of the kind. Under option C the totals are also not directly available — `fantasycast.sh`
would have to grep them back out of another script's stdout, re-coupling the two tools the
subprocess boundary just decoupled.

### The one-argument form defaults to `bojjaes` on the left

`fantasycast.sh 2025 14 wood` is Bojjaes versus Wood. This matches `scores.sh`, whose third argument
already defaults to `bojjaes.csv` in the same week's directory, and it means the common case — my
team against this week's opponent — is the shortest command.

Team-name resolution is `scores.sh`'s, unchanged: `fantasycast.sh` passes the name straight through,
so a name means the same file in both tools and no resolution logic is duplicated. A name with no
file under that week fails inside `scores.sh` with its existing message.

## Risks / Trade-offs

- **`scores.sh` output becomes an interface between two scripts, without being one.** A future change
  to its preamble or spacing shifts the matchup layout. → Mitigated by consuming it opaquely: the
  layout code never counts lines from the top or matches on heading text, it only pads and pairs
  whatever lines it is given. The header removal exists specifically so nothing has to be stripped.
- **Two Sleeper fetches per matchup, ~1MB where one request would have moved 0.5MB.** → Accepted.
  It buys independent failure per team and permanent freedom from the 26-id cap, and the server
  already treats the upstream fetch as fixed-cost regardless of roster size.
- **Unequal roster lengths leave the two `BENCH` headings on different lines.** A reader may expect
  the sections to align horizontally. → Accepted and specified: columns are independent, and forcing
  them to align would mean inserting blank lines inside one team's report, which is worse.
- **A partial failure prints half a matchup.** If the second `scores.sh` fails, the first has already
  run. → `fantasycast.sh` captures both reports before printing anything, so a failure of either
  exits non-zero with the server's message and no columns.
- **Removing the header breaks anyone parsing `scores.sh` output.** → No such consumer exists in the
  repo; the README example is updated in this change.
- **Both scripts remain untested**, and this change roughly doubles the shell surface. → See below.

## Open Questions

None outstanding. The TDD question is settled below.

### The shell scripts stay untested

`CLAUDE.md` mandates a strict TDD red-green loop, and `split-roster-starters-and-bench` carved the
shell scripts out of it as manual tools. That carve-out holds here, deliberately, even though this
change roughly doubles the shell surface with argument branching, a defaulting rule, and a layout
algorithm.

The reason is that these scripts are an *intermediate* user interface. The intended destination is a
hosted one — Cloudflare, or a full website — and the terminal report exists to make the league's
scoring usable before that exists. Building a shell test harness would invest in a layer with a known
expiry date, while every scoring rule that matters stays in the tested Go packages under
`internal/score/`. The tasks in this change verify behavior by running the scripts against a live
server rather than by automated test.

This is a scoped exemption, not a precedent: logic that outlives the scripts belongs in Go, under the
normal loop.
