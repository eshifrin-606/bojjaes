## ADDED Requirements

### Requirement: A season's latest existing week can be found without resolving it

The lineup tree SHALL answer, for a season, the greatest week number it holds a directory for. The
answer is the largest `<week>` such that `<season>/<week>` exists per the existence rule above; it
SHALL NOT depend on whether that week's directory is a well-formed matchup.

The answer SHALL depend only on which week directories are present under the season. It SHALL NOT
open, read, or parse any roster file. A gap in a season's weeks SHALL NOT change the answer: the
latest week is the greatest number present, not one more than the previous answer or the count of
directories.

A season that holds no week directories SHALL report that it has no latest week, distinguishably
from reporting week 0 or any other number.

#### Scenario: The latest week is the greatest present

- **WHEN** the tree holds `2026/1/` and `2026/2/` and nothing else under 2026
- **THEN** season 2026's latest week is 2

#### Scenario: A gap does not lower the answer

- **WHEN** the tree holds `2026/1/` and `2026/3/` but not `2026/2/`
- **THEN** season 2026's latest week is 3

#### Scenario: A week that is not a matchup still counts

- **WHEN** the tree's `2026/3/` directory holds `bojjaes.csv`, `wood.csv`, and `aroma.csv`, and no
  higher week exists under 2026
- **THEN** season 2026's latest week is 3

#### Scenario: A season with no weeks has no latest week

- **WHEN** the tree holds no directories under `2027`
- **THEN** season 2027 has no latest week

#### Scenario: A malformed roster does not affect the answer

- **WHEN** the tree's `2026/2/` directory holds a `wood.csv` that would fail to parse, and no higher
  week exists under 2026
- **THEN** season 2026's latest week is 2 and no roster file is read
