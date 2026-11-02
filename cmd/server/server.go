package main

import (
	"net/http"
	"time"
)

// A slowloris bound: headers arrive in one packet or the client is not serious.
const readHeaderTimeout = 5 * time.Second

// Every route is a GET, so there is no body worth waiting for.
const readTimeout = 10 * time.Second

// Covers a typical cache miss, but not the worst case: a matchup page fetches
// stats then games, each bounded only by the cache's 30s fetch timeout. A
// response cut off here is not wasted, since the cache's fetches outlive the
// request and the page's next load is served from cache.
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
