// Command server scores NFL players against the rushing/receiving slice of the
// HMFFL rules, from Sleeper's weekly stats.
//
// GET /{season}/{week} serves that week's matchup as an HTML page: the two
// teams of that week's lineup directory, their starters, and each lineup's
// total.
//
// POST /scores takes a season, a week, and up to a lineup's worth of player IDs,
// and answers with each player's stat line and point total.
//
// This is the composition root: it is the only package that names a provider.
package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eshifrin/bojjaes/internal/lineup"
	"github.com/eshifrin/bojjaes/internal/sleeper"
	"github.com/eshifrin/bojjaes/internal/statscache"
)

// Comfortably above WriteTimeout minus a typical request, and below Fly's own
// kill timeout, so we exit on our terms rather than being SIGKILLed.
const shutdownGrace = 15 * time.Second

func main() {
	// Wrapped once, here, and handed to both handlers: the cache bounds
	// upstream volume only if everything that reads a week reads through the
	// same one. A cache per handler would be two budgets for one league.
	stats := statscache.New(sleeper.Client{BaseURL: sleeper.BaseURL}, statscache.TTL)

	addr := resolveAddr(os.Getenv)
	srv := newServer(addr, newMux(stats, lineup.New(lineup.Embedded)))

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	log.Printf("listening on %s; GET http://localhost%s/2025/15 and POST http://localhost%s/scores", addr, addr, addr)
	listen := func() (net.Listener, error) { return net.Listen("tcp", addr) }
	if err := run(srv, listen, stop, shutdownGrace); err != nil {
		log.Printf("server stopped: %v", err)
		os.Exit(1)
	}
}
