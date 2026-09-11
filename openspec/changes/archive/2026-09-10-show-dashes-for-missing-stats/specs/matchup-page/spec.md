## MODIFIED Requirements

### Requirement: A starter the provider has no stats for shows no number

A starter absent from the provider's weekly payload SHALL be rendered with the placeholder `--` in
place of a point value, not with `0`, and SHALL contribute nothing to the column total. A starter
present in the payload with no scoring production SHALL be rendered as `0` and is a real scoreless
line.

Absence and a scoreless week are distinguishable on the page because they are different facts: the
payload cannot say whether a missing player has not kicked off, is inactive, or does not exist.

The column total SHALL always be rendered as a number, never as the placeholder, including when
every starter in the column is absent.

#### Scenario: A missing starter is not zero

- **WHEN** one of nine starters has no entry in the weekly payload and the other eight total 40
- **THEN** that starter's line shows `--` rather than `0` and the column total is 40

#### Scenario: A scoreless starter is zero

- **WHEN** a starter has an entry in the weekly payload but scores nothing under the rules
- **THEN** that starter's line shows `0`

#### Scenario: A column with no stats at all totals zero

- **WHEN** none of a column's nine starters has an entry in the weekly payload
- **THEN** every starter line in that column shows `--` and the column total shows `0`
