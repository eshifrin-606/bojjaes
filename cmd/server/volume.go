package main

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/eshifrin/bojjaes/internal/lineup"
)

func resolveLineupVolume(getenv func(string) string) string {
	return getenv("LINEUP_VOLUME")
}

// The volume names the mount, not the lineups directory, so the app owns the
// subdirectory and nothing else on the mount (lost+found) reads as a season.
func lineupTree(volume string, logf func(string, ...any)) (fs.FS, error) {
	if volume == "" {
		return lineup.Embedded, nil
	}
	if err := lineup.Seed(lineup.Embedded, volume, logf); err != nil {
		return nil, err
	}
	return os.DirFS(filepath.Join(volume, "lineups")), nil
}
