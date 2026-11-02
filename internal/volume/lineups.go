package volume

import (
	"io/fs"
	"os"
	"path/filepath"
)

// The mount holds more than lineups (the seed manifest, lost+found), so the
// store lives in its own subdirectory and nothing else reads as a season.
const lineupsDir = "lineups"

// Lineups is the lineup tree to serve: embedded alone when there is no mount,
// otherwise the store on the mount, seeded from embedded first.
func Lineups(embedded fs.FS, mount string, logf func(string, ...any)) (fs.FS, error) {
	if mount == "" {
		return embedded, nil
	}
	if err := Seed(embedded, mount, logf); err != nil {
		return nil, err
	}
	return os.DirFS(filepath.Join(mount, lineupsDir)), nil
}
