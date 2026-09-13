## ADDED Requirements

### Requirement: Weekly stats are regular-season stats

A season and week SHALL be interpreted as the regular season wherever stats are fetched for them.
Preseason and postseason stats SHALL NOT be fetched. Nothing that asks for a week carries a season
type.

The league scores regular-season play. This rule used to be stated only on the removed multi-player
endpoint, but the matchup page reads through the same fetch, so the rule belongs to the snapshot.

#### Scenario: A week is fetched from the regular-season aggregate

- **WHEN** stats are fetched for a season and week
- **THEN** the provider request names the regular season, never preseason or postseason

## REMOVED Requirements

### Requirement: Multi-player score endpoint

**Reason**: `POST /scores` existed only so `scripts/scores.sh` and `scripts/fantasycast.sh` could run, and it is removed with them.
**Migration**: Read scores on the matchup page at `GET /{season}/{week}`. Scoring itself (`score.Points` over a `score.WeekStats`) is unchanged and still specified by this capability's scoring requirements.

### Requirement: Multi-player request validation

**Reason**: This validated the removed `POST /scores` request body.
**Migration**: None needed. The page validates its season and week from the URL through `score.ValidateSeasonWeek`, as `matchup-page` specifies. The 26-player cap has no remaining caller.
