package main

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/eshifrin/bojjaes/internal/data"
	"github.com/eshifrin/bojjaes/internal/volume"
)

func resolveLineupVolume(getenv func(string) string) string {
	return getenv("LINEUP_VOLUME")
}

// The volume names the mount, not the lineups directory, so the app owns the
// subdirectory and nothing else on the mount (lost+found) reads as a season.
func lineupTree(mount string, logf func(string, ...any)) (fs.FS, error) {
	if mount == "" {
		return data.Lineups, nil
	}
	if err := volume.Seed(data.Lineups, mount, logf); err != nil {
		return nil, err
	}
	return os.DirFS(filepath.Join(mount, "lineups")), nil
}
