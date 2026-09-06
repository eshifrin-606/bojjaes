## ADDED Requirements

### Requirement: Weekly stat snapshot

One fetch of a season and week SHALL yield a provider-neutral snapshot from which any player's stat
line can be read. The snapshot SHALL carry the season and week it was fetched for, so a stat line
read from it cannot be attributed to a different week.

The snapshot SHALL NOT expose the provider's payload shape or its stat keys. A consumer holding a
snapshot SHALL be able to read a player without knowing which provider produced it.

Reading a player from a snapshot SHALL report absence as a value rather than an error, on the same
terms as the transform: the payload cannot say whether a missing player has not kicked off, was
inactive, or does not exist.

A snapshot SHALL be complete when it is returned — no part of its content is computed later — so
that reading from it cannot fail, cannot vary between reads, and is safe to share between concurrent
requests.

#### Scenario: A player is read from a snapshot

- **WHEN** a snapshot has been fetched for a season and week and a player is present in it
- **THEN** reading that player yields the stat line for that season and week

#### Scenario: An absent player is a value, not an error

- **WHEN** a player has no entry in the snapshot
- **THEN** reading that player reports absence without an error and without a zeroed stat line

#### Scenario: The snapshot carries no provider shape

- **WHEN** a snapshot is held by a consumer
- **THEN** that consumer can read any player without referring to a provider stat key or payload type

#### Scenario: Repeated reads agree

- **WHEN** the same player is read from the same snapshot twice, including from two concurrent
  requests
- **THEN** both reads yield the same result

## REMOVED Requirements

### Requirement: Score endpoint

**Reason**: It was a walking-skeleton smoke test wearing an endpoint's clothes — a hardcoded player
and week (Puka Nacua, 2025 week 14) that also printed to standard output. Its value was proving the
fetch-and-score path end to end, which the multi-player endpoint now does for any player, and which
the adapter's own tests do against the same settled week without a server.

**Migration**: Use the multi-player score endpoint with `player_ids: ["9493"]`, season 2025, week 14
for the same answer. The absent-target check the endpoint performed becomes what it always was — a
test — and moves into the provider adapter's test suite. Nothing in `scripts/**` calls `GET /score`.
