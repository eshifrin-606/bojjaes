package main

import (
	"strings"
	"testing"
	"time"
)

func fakeEnv(vars map[string]string) func(string) string {
	return func(key string) string { return vars[key] }
}

func TestLoadConfigAddr(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"uses PORT", map[string]string{"PORT": "3000"}, ":3000"},
		{"defaults when PORT unset", nil, ":8080"},
		{"defaults when PORT empty", map[string]string{"PORT": ""}, ":8080"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := loadConfig(fakeEnv(tt.env))

			if err != nil || cfg.addr != tt.want {
				t.Errorf("loadConfig addr = %q, %v; want %q, nil", cfg.addr, err, tt.want)
			}
		})
	}
}

func TestLoadConfigPlaysColdWait(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want time.Duration
	}{
		{"parses the duration", map[string]string{"PLAYS_COLD_WAIT": "10s"}, 10 * time.Second},
		{"defaults when unset", nil, 30 * time.Second},
		{"defaults when empty", map[string]string{"PLAYS_COLD_WAIT": ""}, 30 * time.Second},
		{"accepts zero", map[string]string{"PLAYS_COLD_WAIT": "0"}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := loadConfig(fakeEnv(tt.env))

			if err != nil || cfg.playsColdWait != tt.want {
				t.Errorf("loadConfig playsColdWait = %v, %v; want %v, nil", cfg.playsColdWait, err, tt.want)
			}
		})
	}
}

func TestLoadConfigRejectsABadPlaysColdWait(t *testing.T) {
	for _, v := range []string{"ten", "-5s"} {
		t.Run(v, func(t *testing.T) {
			_, err := loadConfig(fakeEnv(map[string]string{"PLAYS_COLD_WAIT": v}))

			if err == nil || !strings.Contains(err.Error(), "PLAYS_COLD_WAIT") {
				t.Errorf("err = %v, want an error naming PLAYS_COLD_WAIT", err)
			}
		})
	}
}

func TestLoadConfigLineupVolume(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"uses LINEUP_VOLUME", map[string]string{"LINEUP_VOLUME": "/data"}, "/data"},
		{"is empty when unset", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := loadConfig(fakeEnv(tt.env))

			if err != nil || cfg.lineupVolume != tt.want {
				t.Errorf("loadConfig lineupVolume = %q, %v; want %q, nil", cfg.lineupVolume, err, tt.want)
			}
		})
	}
}
