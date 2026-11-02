// Package volume decides which lineup tree the server reads, and seeds the
// store on a mounted volume from the copy embedded in the binary. It is the
// only package that writes lineup files.
package volume

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"
)

// Seed copies each numbered week of src into mount/lineups, replacing a store
// week only when git's copy changed since it was last seeded.
func Seed(src fs.FS, mount string, logf func(string, ...any)) error {
	m, err := loadManifest(mount, logf)
	if err != nil {
		return err
	}
	for _, d := range weekDirs(src) {
		hash, err := weekHash(src, d.season, d.week)
		if err != nil {
			return err
		}
		key := path.Join(d.season, d.week)
		if m.Weeks[key] == hash {
			continue
		}
		if err := logOverwrite(mount, d, m.Weeks[key], hash, logf); err != nil {
			return err
		}
		if err := copyWeek(src, mount, d); err != nil {
			return err
		}
		// Recorded only once the copy is complete, so a crash mid-copy leaves
		// no entry and the next boot copies the week again.
		m.Weeks[key] = hash
		if err := m.write(mount); err != nil {
			return err
		}
	}
	return nil
}

// logOverwrite logs every file of a store week that holds content git never
// seeded, such as a phone edit, so the lineup can be recovered from the log
// once the week is replaced.
func logOverwrite(mount string, d weekDir, seeded, incoming string, logf func(string, ...any)) error {
	store := os.DirFS(filepath.Join(mount, lineupsDir))
	dir := path.Join(d.season, d.week)
	if _, err := fs.Stat(store, dir); errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	held, err := weekHash(store, d.season, d.week)
	if err != nil {
		return err
	}
	if held == seeded || held == incoming {
		return nil
	}
	entries, err := fs.ReadDir(store, dir)
	if err != nil {
		return err
	}
	logf("seed: overwriting store week %s, which git never seeded", dir)
	for _, e := range entries {
		if !e.Type().IsRegular() {
			continue
		}
		data, err := fs.ReadFile(store, path.Join(dir, e.Name()))
		if err != nil {
			return err
		}
		logf("seed: overwritten %s/%s: %q", dir, e.Name(), data)
	}
	return nil
}

type manifest struct {
	Weeks map[string]string `json:"weeks"`
}

// loadManifest treats an unparseable manifest as empty: every week then
// re-seeds, and any phone edit it replaces is logged, so nothing is lost
// silently and boot is not blocked on a file it can rebuild.
func loadManifest(mount string, logf func(string, ...any)) (manifest, error) {
	m := manifest{Weeks: map[string]string{}}
	data, err := os.ReadFile(filepath.Join(mount, "seed-manifest.json"))
	if errors.Is(err, fs.ErrNotExist) {
		return m, nil
	}
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(data, &m); err != nil {
		logf("seed: ignoring unreadable seed-manifest.json, re-seeding every week: %v", err)
		return manifest{Weeks: map[string]string{}}, nil
	}
	return m, nil
}

// write goes through a temp file and rename so a crash never leaves a
// half-written manifest.
func (m manifest) write(mount string) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	tmp := filepath.Join(mount, "seed-manifest.json.tmp")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(mount, "seed-manifest.json"))
}

func copyWeek(src fs.FS, mount string, d weekDir) error {
	from := path.Join(d.season, d.week)
	entries, err := fs.ReadDir(src, from)
	if err != nil {
		return err
	}
	// Removing first drops files git no longer has, such as last week's
	// opponent, so the store week matches git's exactly.
	to := filepath.Join(mount, lineupsDir, d.season, d.week)
	if err := os.RemoveAll(to); err != nil {
		return err
	}
	if err := os.MkdirAll(to, 0o755); err != nil {
		return err
	}
	for _, e := range entries {
		if !e.Type().IsRegular() {
			continue
		}
		data, err := fs.ReadFile(src, path.Join(from, e.Name()))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(to, e.Name()), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// weekHash covers the week's file names and contents but not mtimes, so the
// same week embedded in two builds hashes the same. Each file is
// length-prefixed so bytes cannot shift between adjacent files unnoticed.
func weekHash(fsys fs.FS, season, week string) (string, error) {
	dir := path.Join(season, week)
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	for _, e := range entries {
		if !e.Type().IsRegular() {
			continue
		}
		data, err := fs.ReadFile(fsys, path.Join(dir, e.Name()))
		if err != nil {
			return "", err
		}
		h.Write([]byte(e.Name()))
		h.Write([]byte{0})
		h.Write([]byte(strconv.Itoa(len(data))))
		h.Write([]byte{0})
		h.Write(data)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// weekDir keeps a week's names as listed, so "02" is copied and recorded as
// "02" rather than renumbered.
type weekDir struct {
	season, week string
}

func weekDirs(fsys fs.FS) []weekDir {
	seasons, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil
	}
	var dirs []weekDir
	for _, s := range seasons {
		if !isNumberedDir(s) {
			continue
		}
		weeks, err := fs.ReadDir(fsys, s.Name())
		if err != nil {
			continue
		}
		for _, w := range weeks {
			if isNumberedDir(w) {
				dirs = append(dirs, weekDir{s.Name(), w.Name()})
			}
		}
	}
	return dirs
}

func isNumberedDir(e fs.DirEntry) bool {
	_, err := strconv.Atoi(e.Name())
	return e.IsDir() && err == nil
}
