## ADDED Requirements

### Requirement: A week's existence can be asked without resolving it

The lineup tree SHALL answer whether a season and week exist in it. A week exists when the tree
holds a directory at `<season>/<week>`, the same directory that matchup resolution reads.

The answer SHALL depend only on that directory being present. It SHALL NOT open, read, or parse any
roster file, and SHALL NOT decide whether the directory holds a matchup. A week that exists but
would fail resolution (too many, too few, or no Bojjaes roster) SHALL still be reported as
existing: it is a mistake in the tree, not a missing week.

A file at `<season>/<week>`, rather than a directory, SHALL NOT be reported as an existing week.

#### Scenario: A week with a directory exists

- **WHEN** the tree holds `2026/2/bojjaes.csv` and `2026/2/renegades.csv`
- **THEN** season 2026 week 2 exists

#### Scenario: A week with no directory does not exist

- **WHEN** the tree holds `2026/1/` and `2026/2/` and nothing else under 2026
- **THEN** season 2026 week 3 does not exist

#### Scenario: A week that is not a matchup still exists

- **WHEN** the tree's `2026/3/` directory holds `bojjaes.csv`, `wood.csv`, and `aroma.csv`
- **THEN** season 2026 week 3 exists

#### Scenario: A malformed roster does not affect existence

- **WHEN** the tree's `2026/3/` directory holds `bojjaes.csv` and a `wood.csv` that would fail to
  parse
- **THEN** season 2026 week 3 exists and neither file is read

#### Scenario: A file where a week should be is not a week

- **WHEN** the tree holds a regular file named `2026/3`
- **THEN** season 2026 week 3 does not exist
