package main

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/eshifrin/bojjaes/internal/lineup"
)

func resolveLineupVolume(getenv func(string) string) string {
	return getenv("LINEUP_VOLUME")
}

// The volume names the mount, not the lineups directory, so the app owns the
// subdirectory and nothing else on the mount (lost+found) reads as a season.
func lineupTree(volume string) fs.FS {
	if volume == "" {
		return lineup.Embedded
	}
	return lineup.Layered(lineup.Embedded, os.DirFS(filepath.Join(volume, "lineups")), log.Printf)
}
