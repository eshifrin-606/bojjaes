package lineup

import (
	"io"
	"io/fs"
	"path"
	"slices"
	"strconv"
	"strings"
)

// Layered composes the embedded archive with a volume holding the in-flight
// week, so Tree answers over both without knowing there are two.
func Layered(archive, volume fs.FS, logf func(string, ...any)) fs.FS {
	l := &layered{archive: archive, volume: volume}
	for _, d := range weekDirs(archive) {
		if !l.hasArchive || l.archiveLatest.before(d.weekDir) {
			l.archiveLatest, l.hasArchive = d.weekDir, true
		}
	}
	// Logged here rather than per request, which would repeat it on every page
	// view. An ignored week is left on the volume: it may hold edits that never
	// reached git.
	scan := l.scanVolume()
	for _, d := range scan.stale {
		logf("lineup: ignoring volume week %s/%s, older than the archive's latest %s/%s",
			d.season, d.week, l.archiveLatest.season, l.archiveLatest.week)
	}
	for _, d := range scan.superseded {
		logf("lineup: ignoring volume week %s/%s, older than the volume's latest %s/%s",
			d.season, d.week, scan.active.season, scan.active.week)
	}
	return l
}

type layered struct {
	archive, volume fs.FS
	// The archive is embedded in the binary and never changes, so its latest
	// week is found once.
	archiveLatest weekDir
	hasArchive    bool
}

// weekDir is a <season>/<week> directory, keeping its names as listed so it
// can be routed by name and ordered by number.
type weekDir struct {
	season, week string
}

func (d weekDir) numbers() (season, week int) {
	season, _ = strconv.Atoi(d.season)
	week, _ = strconv.Atoi(d.week)
	return season, week
}

func (d weekDir) before(other weekDir) bool {
	s1, w1 := d.numbers()
	s2, w2 := other.numbers()
	return s1 < s2 || s1 == s2 && w1 < w2
}

func (d weekDir) contains(name string) bool {
	dir := path.Join(d.season, d.week)
	return name == dir || strings.HasPrefix(name, dir+"/")
}

// listedWeek is a week directory with the entry its season listed it under,
// so a merged season listing need not list the volume a second time.
type listedWeek struct {
	weekDir
	seasonEntry, entry fs.DirEntry
}

func weekDirs(fsys fs.FS) []listedWeek {
	seasons, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil
	}
	var dirs []listedWeek
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
				dirs = append(dirs, listedWeek{weekDir{s.Name(), w.Name()}, s, w})
			}
		}
	}
	return dirs
}

func isNumberedDir(e fs.DirEntry) bool {
	_, err := strconv.Atoi(e.Name())
	return e.IsDir() && err == nil
}

type volumeScan struct {
	active    listedWeek
	hasActive bool
	stale     []weekDir
	// superseded are weeks at or after the archive latest that lost to a
	// greater volume week.
	superseded []weekDir
}

// scanVolume is run per call rather than cached, so a week written to the
// volume while the server runs is served without a restart.
func (l *layered) scanVolume() volumeScan {
	var scan volumeScan
	for _, d := range weekDirs(l.volume) {
		if l.hasArchive && d.before(l.archiveLatest) {
			scan.stale = append(scan.stale, d.weekDir)
			continue
		}
		switch {
		case !scan.hasActive:
			scan.active, scan.hasActive = d, true
		case scan.active.before(d.weekDir):
			scan.superseded = append(scan.superseded, scan.active.weekDir)
			scan.active = d
		default:
			scan.superseded = append(scan.superseded, d.weekDir)
		}
	}
	return scan
}

func (l *layered) Open(name string) (fs.File, error) {
	scan := l.scanVolume()
	if scan.hasActive && scan.active.contains(name) {
		return l.volume.Open(name)
	}
	if !scan.hasActive || name != "." && name != scan.active.season {
		return l.archive.Open(name)
	}
	info, err := fs.Stat(l.archive, name)
	if err != nil {
		info, err = fs.Stat(l.volume, name)
	}
	if err != nil {
		return nil, err
	}
	entries, err := l.ReadDir(name)
	if err != nil {
		return nil, err
	}
	return &mergedDir{info: info, entries: entries}, nil
}

// mergedDir is a directory whose listing spans both layers, so opening it
// lists what ReadDir lists.
type mergedDir struct {
	info    fs.FileInfo
	entries []fs.DirEntry
}

func (d *mergedDir) Stat() (fs.FileInfo, error) { return d.info, nil }
func (d *mergedDir) Close() error               { return nil }

func (d *mergedDir) Read([]byte) (int, error) {
	return 0, &fs.PathError{Op: "read", Path: d.info.Name(), Err: fs.ErrInvalid}
}

func (d *mergedDir) ReadDir(n int) ([]fs.DirEntry, error) {
	if n <= 0 {
		entries := d.entries
		d.entries = nil
		return entries, nil
	}
	if len(d.entries) == 0 {
		return nil, io.EOF
	}
	n = min(n, len(d.entries))
	entries := d.entries[:n]
	d.entries = d.entries[n:]
	return entries, nil
}

func (l *layered) ReadDir(name string) ([]fs.DirEntry, error) {
	scan := l.scanVolume()
	if scan.hasActive && scan.active.contains(name) {
		return fs.ReadDir(l.volume, name)
	}
	entries, archiveErr := fs.ReadDir(l.archive, name)
	if !scan.hasActive {
		return entries, archiveErr
	}
	switch name {
	case ".":
		if !slices.ContainsFunc(entries, func(e fs.DirEntry) bool { return e.Name() == scan.active.season }) {
			entries = append(entries, scan.active.seasonEntry)
		}
	case scan.active.season:
		week := scan.active.week
		entries = slices.DeleteFunc(entries, func(e fs.DirEntry) bool { return e.Name() == week })
		entries = append(entries, scan.active.entry)
	default:
		return entries, archiveErr
	}
	slices.SortFunc(entries, func(a, b fs.DirEntry) int { return strings.Compare(a.Name(), b.Name()) })
	return entries, nil
}
