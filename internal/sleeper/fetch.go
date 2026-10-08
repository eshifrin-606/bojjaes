package sleeper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// fetchRecentPlays reads the limit newest plays of a week, newest first.
func fetchRecentPlays(ctx context.Context, baseURL string, season, week, limit int) ([]play, error) {
	url := fmt.Sprintf("%s/plays/nfl/recent?season_type=regular&season=%d&week=%d&limit=%d", baseURL, season, week, limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("building sleeper plays request for season %d week %d: %w", season, week, err)
	}

	resp, err := sleeperClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching sleeper plays for season %d week %d: %w", season, week, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching sleeper plays for season %d week %d: status %s", season, week, resp.Status)
	}

	var plays []play
	if err := json.NewDecoder(resp.Body).Decode(&plays); err != nil {
		return nil, fmt.Errorf("decoding sleeper plays for season %d week %d: %w", season, week, err)
	}
	return plays, nil
}
