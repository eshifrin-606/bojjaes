// Package data holds the league files carried in the binary. go:embed cannot
// reach outside its own package directory, so the files and the directives
// that embed them live together here.
package data

import (
	"embed"
	"io/fs"
)

//go:embed lineups
var lineups embed.FS

// Lineups is rooted at the season directories so it is interchangeable with
// any other tree a lineup.Tree is built over. Adding a season is adding a
// directory: the directive names lineups, not the seasons under it.
var Lineups = mustSub(lineups, "lineups")

//go:embed rosters
var rosters embed.FS

// Rosters holds one <team>.csv per team.
var Rosters = mustSub(rosters, "rosters")

// A failure here is a broken directive, not a broken request, so it fails the
// process at initialisation the way internal/web fails an unparsable template.
func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic("data: rooting the embedded tree at " + dir + ": " + err.Error())
	}
	return sub
}
