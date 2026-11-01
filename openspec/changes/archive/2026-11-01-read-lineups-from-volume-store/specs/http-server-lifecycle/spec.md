## ADDED Requirements

### Requirement: Lineup seeding completes before the server listens

When a lineup volume is configured, the process SHALL complete boot seeding (per `lineup-source`)
before it opens its listener, so no request is ever answered from a partly seeded store.

If seeding fails — the volume or its manifest cannot be read or written — the process SHALL log the
error and exit with a non-zero status without opening the listener. A volume the app cannot write
to therefore blocks reads as well as writes (ADR 0005 consequences), which is preferred to serving
a store of unknown state.

With no volume configured, nothing is seeded and startup proceeds as before.

#### Scenario: The tree handed to the server is already seeded

- **WHEN** the server's lineup tree is built with a volume configured over an empty directory
- **THEN** by the time the tree is returned, it already serves every embedded week

#### Scenario: A seeding failure stops startup

- **WHEN** `LINEUP_VOLUME` names a directory the process cannot create `lineups/` or the manifest
  in
- **THEN** building the lineup tree returns an error, and the process exits non-zero without
  listening

#### Scenario: No volume, no seeding

- **WHEN** `LINEUP_VOLUME` is unset
- **THEN** the lineup tree is the embedded tree and no file is written
