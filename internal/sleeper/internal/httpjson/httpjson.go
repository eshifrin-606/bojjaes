// Package httpjson is the GET-and-decode helper the Sleeper packages share, so
// they share one timed client too.
package httpjson

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// http.DefaultClient has no timeout, so a stalled upstream would hang the
// request forever. The budget covers the whole exchange, body included.
var client = &http.Client{Timeout: 15 * time.Second}

// Get decodes the 200 response body of a GET into out. what names the
// lookup in every error, so a failed call says which season and week it was.
func Get(ctx context.Context, url, what string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("building request for %s: %w", what, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("fetching %s: %w", what, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetching %s: status %s", what, resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding %s: %w", what, err)
	}
	return nil
}
