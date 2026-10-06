package main

import (
	"fmt"
	"time"
)

// The default is the local value because local runs set no environment;
// fly.toml sets its own.
const defaultPlaysColdWait = 30 * time.Second

func resolvePlaysColdWait(getenv func(string) string) (time.Duration, error) {
	v := getenv("PLAYS_COLD_WAIT")
	if v == "" {
		return defaultPlaysColdWait, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("PLAYS_COLD_WAIT: %w", err)
	}
	if d < 0 {
		return 0, fmt.Errorf("PLAYS_COLD_WAIT: %v is negative", d)
	}
	return d, nil
}
