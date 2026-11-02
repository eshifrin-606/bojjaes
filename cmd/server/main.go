// Command server serves the Bojjaes' HMFFL matchups as HTML pages, scored from
// Sleeper's weekly stats. The routes are listed in newMux.
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

	"github.com/eshifrin/bojjaes/internal/data"
	"github.com/eshifrin/bojjaes/internal/lineup"
	"github.com/eshifrin/bojjaes/internal/sleeper"
	"github.com/eshifrin/bojjaes/internal/sleeper/plays"
	"github.com/eshifrin/bojjaes/internal/volume"
	"github.com/eshifrin/bojjaes/internal/weekcache"
)

// Comfortably above WriteTimeout minus a typical request, and below Fly's own
// kill timeout, so we exit on our terms rather than being SIGKILLed.
const shutdownGrace = 15 * time.Second

func main() {
	cfg, err := loadConfig(os.Getenv)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// Before any disk or network work, so none of it runs as root.
	if err := claimVolumeAndDropRoot(cfg.lineupVolume); err != nil {
		log.Fatalf("privileges: %v", err)
	}

	log.Printf("plays cold-week wait: %v", cfg.playsColdWait)

	// Wrapped once, here: the cache bounds upstream volume only if everything
	// that reads a week reads through the same one.
	playStore := plays.NewStore(sleeper.BaseURL, cfg.playsColdWait, log.Printf)
	client := sleeper.Client{
		BaseURL: sleeper.BaseURL,
		Plays:   playStore,
		Logf:    log.Printf,
	}
	stats := weekcache.New("statscache", client.WeekStats, weekcache.TTL)
	// A whole-week fetch that outlasts the cold wait lands after its week was
	// cached from the poll alone; dropping the entry lets the next read see it.
	playStore.OnWholeWeekChanged(stats.Invalidate)
	games := weekcache.New("gamescache", client.WeekGames, weekcache.TTL)

	lineups, err := volume.Lineups(data.Lineups, cfg.lineupVolume, log.Printf)
	if err != nil {
		log.Fatalf("lineups: %v", err)
	}

	srv := newServer(cfg.addr, newMux(stats, games, lineup.New(lineups)))

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	log.Printf("listening on %s", cfg.addr)
	listen := func() (net.Listener, error) { return net.Listen("tcp", cfg.addr) }
	if err := run(srv, listen, stop, shutdownGrace); err != nil {
		log.Printf("server stopped: %v", err)
		os.Exit(1)
	}
}
