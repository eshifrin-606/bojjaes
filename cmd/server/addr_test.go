package main

import "testing"

func fakeEnv(vars map[string]string) func(string) string {
	return func(key string) string { return vars[key] }
}

func TestResolveAddrUsesPort(t *testing.T) {
	got := resolveAddr(fakeEnv(map[string]string{"PORT": "3000"}))

	if got != ":3000" {
		t.Errorf("resolveAddr = %q, want %q", got, ":3000")
	}
}

func TestResolveAddrDefaultsWhenPortUnset(t *testing.T) {
	got := resolveAddr(fakeEnv(nil))

	if got != ":8080" {
		t.Errorf("resolveAddr = %q, want %q", got, ":8080")
	}
}

func TestResolveAddrDefaultsWhenPortEmpty(t *testing.T) {
	got := resolveAddr(fakeEnv(map[string]string{"PORT": ""}))

	if got != ":8080" {
		t.Errorf("resolveAddr = %q, want %q", got, ":8080")
	}
}
