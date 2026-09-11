## Why

A starter the provider has no stats for renders on the matchup page as the words `no stats`, in the
slot every other line fills with a number. In a half-width column on a phone that phrase crowds the
player's name and reads as prose among numerals. `--` says the same thing — there is no value here —
in a shape that sits in a points column the way a number does.

## What Changes

- An absent starter's points slot renders `--` instead of `no stats`.
- The `matchup-page` spec names the placeholder. Today it says only "a placeholder"; it will say
  `--`, so the rendered string is pinned by requirement rather than by a test alone.
- Unchanged: an absent starter is still not `0`, still contributes nothing to the column total, and
  a starter present with no scoring production still renders `0`.
- `scripts/scores.sh` keeps printing `no stats`. The page's code comment that ties its wording to
  the script is removed, since the two UIs deliberately diverge from here on.

Out of scope:

- The terminal report (`lineup-score-report`, `matchup-report`) and the README's examples of it.
  The script is interim UI and its `no stats` wording is sized into its column alignment.
- Any marker on the column total for absent starters. The `--` lines sit directly above it, as the
  `no stats` lines do today.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `matchup-page`: the requirement "A starter the provider has no stats for shows no number" names
  `--` as the placeholder. The absent-versus-scoreless distinction and the total's arithmetic are
  unchanged.

## Impact

- `internal/web/matchup.go`: the placeholder constant's value, and the comment in `scoreColumn` that
  says the wording matches `scripts/scores.sh`.
- `internal/web/matchup_test.go`: the absent-starter assertion expects `--`.
- No change to `internal/web/matchup.html`, the CSS, `internal/score`, `internal/api`,
  `POST /scores`, `scripts/**`, or the README.
- `refresh-page-while-visible` is also an open change against `matchup-page`. It adds a different
  requirement and does not touch this one, so the two deltas do not conflict.
