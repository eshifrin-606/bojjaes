## Context

`scoreColumn` in `internal/web/matchup.go` asks `score.WeekStats.Player` for each starter. When the
second return is `false` it sets the starter's `Points` to the package constant `noStats`
(`"no stats"`) and adds nothing to the total. The template prints `Points` verbatim into
`<span class="points">`. The choice between a number and a placeholder is already made in Go, so
this change is a one-constant change behind one test.

A comment on that branch says the wording matches `scripts/scores.sh` "while both UIs exist". The
script is interim UI and keeps its wording, so that comment is about to become false.

## Goals / Non-Goals

**Goals:**

- An absent starter's points slot on the page reads `--`.
- The `matchup-page` spec names `--`, so the placeholder is part of the contract rather than an
  implementation detail a test happens to pin.
- The absent-versus-scoreless distinction, and the total that ignores absent starters, are
  untouched.

**Non-Goals:**

- `scripts/scores.sh`, `lineup-score-report`, `matchup-report`, and the README's examples.
- Styling the placeholder differently from a number (dimmed, italic, tooltip). The page stays CSS
  neutral here, and "no styling that distinguishes" rules elsewhere in the spec make any new styling
  worth its own decision.
- Marking the column total when a starter is absent.

## Decisions

### Two ASCII hyphens, exactly as asked

The placeholder is the two-byte string `--`, not an em dash (`—`) or en dash (`–`).

**Alternatives considered:**

- **Em dash `—`.** Typographically the conventional "no value" mark, and one glyph wide. Rejected
  for now because it is not what was asked for, and swapping it later is the same one-constant
  change.
- **Empty slot.** Rejected: an empty right-hand span collapses the flex row's visual rhythm and
  reads as a rendering bug rather than a stated absence.

### Keep the constant, rename nothing

`noStats` still names the fact (the provider has no stats), not the glyph, so the name stays and
only its value changes. The comment explaining it stands. The script-parity sentence in
`scoreColumn` is removed; the sentence about absence and a scoreless week being different facts
stays, because that is the reason the branch exists.

### The total stays a number even when every starter is absent

Absent starters contribute nothing, so a column whose nine starters are all absent totals `0`, and
that is what renders — not `--`. This is today's behaviour, and the change pins it with a scenario
so the new placeholder cannot drift into the total line.

## Risks / Trade-offs

**`--` is terser than `no stats`, so a first-time reader may not know what it means** → The page is
read by three people who know the league. The two values that could be confused, `--` and `0`,
are the ones the spec already requires to differ, and they still do.

**The page and the script now word the same fact differently** → Deliberate, per the proposal. The
script is interim UI slated for replacement by the page, and the coupling comment is removed so
nothing claims otherwise.

**A screen reader may announce `--` as "dash dash" or skip it** → Accepted for now; no part of the
page has accessibility labelling yet. If that changes, a visually hidden "no stats" beside the
glyph is the natural follow-up.
