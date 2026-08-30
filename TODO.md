## To Do

This is my scrum board

## In Progress

| task | priority | description of value | risk if not done |
| -- | -- | -- | -- |

## Backlog

| task | priority | description of value | risk if not done |
| -- | -- | -- | -- |
| find docs on sleeper api | medium |  improve trust in api | investigation solely based on hitting api could be wrong or incomplete


## Recently Completed

- Single player-week stats (puka nacua, week 14). Running a go server that makes a network call to sleeper, transforms the data, calculates the fantasy score, and returns nfl and hmffl stats for the given player week.
    - next up: improve flexibility. Allow calls for any wide reciever, any week, any year
- Flexible player-week scoring. Any player, any week, any season, scored in batches from a saved roster file.
- Matchup report. `scripts/fantasycast.sh <season> <week> <team> [team2]` prints two rosters side by side, each column a full `scripts/scores.sh` report, with no margin or winner line.