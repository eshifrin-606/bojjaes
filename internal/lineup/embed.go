package lineup

import (
	"embed"
	"io/fs"
)

//go:embed data
var embedded embed.FS

// Embedded is the lineup tree carried in the binary, rooted at the season
// directories so it is interchangeable with any other tree a Tree is built
// over. Adding a season is adding a directory: the directive names data, not
// the seasons under it.
var Embedded = mustRoot()

// A failure here is a broken directive, not a broken request, so it fails the
// process at initialisation the way internal/web fails an unparsable template.
func mustRoot() fs.FS {
	sub, err := fs.Sub(embedded, "data")
	if err != nil {
		panic("lineup: rooting the embedded tree at data: " + err.Error())
	}
	return sub
}
