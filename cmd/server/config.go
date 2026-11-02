package main

import (
	"fmt"
	"time"
)

// The platform decides where we listen; 8080 is only the local convenience.
const defaultPort = "8080"

// The fallback when nothing sets PLAYS_COLD_WAIT; config/local.env and
// fly.toml each set their own.
const defaultPlaysColdWait = 30 * time.Second

type config struct {
	addr          string
	playsColdWait time.Duration
	lineupVolume  string
}

// loadConfig takes the environment lookup as an argument rather than calling
// os.Getenv, so a test can supply a fake without touching the process
// environment.
func loadConfig(getenv func(string) string) (config, error) {
	port := getenv("PORT")
	if port == "" {
		port = defaultPort
	}
	coldWait, err := playsColdWait(getenv("PLAYS_COLD_WAIT"))
	if err != nil {
		return config{}, err
	}
	return config{
		addr:          ":" + port,
		playsColdWait: coldWait,
		lineupVolume:  getenv("LINEUP_VOLUME"),
	}, nil
}

func playsColdWait(v string) (time.Duration, error) {
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
