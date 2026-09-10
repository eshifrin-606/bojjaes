package main

import (
	"net/http"
	"time"
)

// A slowloris bound: headers arrive in one packet or the client is not serious.
const readHeaderTimeout = 5 * time.Second

// Bodies here are a small JSON array of player IDs.
const readTimeout = 10 * time.Second

// Must cover a matchup page whose stat fetch is a cache miss: an upstream
// Sleeper call plus render. The cache's own fetch timeout is the real bound;
// this sits above it.
const writeTimeout = 30 * time.Second

// Keep-alive reuse for the page's self-refresh, without holding connections
// indefinitely on a 256mb machine.
const idleTimeout = 60 * time.Second

func newServer(addr string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
}
