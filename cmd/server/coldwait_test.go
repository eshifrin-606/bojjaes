package main

import (
	"strings"
	"testing"
	"time"
)

func TestResolvePlaysColdWaitParsesDuration(t *testing.T) {
	got, err := resolvePlaysColdWait(fakeEnv(map[string]string{"PLAYS_COLD_WAIT": "10s"}))

	if err != nil || got != 10*time.Second {
		t.Errorf("resolvePlaysColdWait = %v, %v; want 10s, nil", got, err)
	}
}

func TestResolvePlaysColdWaitDefaultsWhenUnset(t *testing.T) {
	got, err := resolvePlaysColdWait(fakeEnv(nil))

	if err != nil || got != 30*time.Second {
		t.Errorf("resolvePlaysColdWait = %v, %v; want 30s, nil", got, err)
	}
}

func TestResolvePlaysColdWaitDefaultsWhenEmpty(t *testing.T) {
	got, err := resolvePlaysColdWait(fakeEnv(map[string]string{"PLAYS_COLD_WAIT": ""}))

	if err != nil || got != 30*time.Second {
		t.Errorf("resolvePlaysColdWait = %v, %v; want 30s, nil", got, err)
	}
}

func TestResolvePlaysColdWaitAcceptsZero(t *testing.T) {
	got, err := resolvePlaysColdWait(fakeEnv(map[string]string{"PLAYS_COLD_WAIT": "0"}))

	if err != nil || got != 0 {
		t.Errorf("resolvePlaysColdWait = %v, %v; want 0, nil", got, err)
	}
}

func TestResolvePlaysColdWaitRejectsMalformed(t *testing.T) {
	_, err := resolvePlaysColdWait(fakeEnv(map[string]string{"PLAYS_COLD_WAIT": "ten"}))

	if err == nil || !strings.Contains(err.Error(), "PLAYS_COLD_WAIT") {
		t.Errorf("err = %v, want an error naming PLAYS_COLD_WAIT", err)
	}
}

func TestResolvePlaysColdWaitRejectsNegative(t *testing.T) {
	_, err := resolvePlaysColdWait(fakeEnv(map[string]string{"PLAYS_COLD_WAIT": "-5s"}))

	if err == nil || !strings.Contains(err.Error(), "PLAYS_COLD_WAIT") {
		t.Errorf("err = %v, want an error naming PLAYS_COLD_WAIT", err)
	}
}
