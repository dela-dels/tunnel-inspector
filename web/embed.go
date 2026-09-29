package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// Dist returns the sub-filesystem containing the built web assets.
func Dist() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}
