// Package web holds the built frontend, embedded in the program. Build it
// with `make web` (or `npm run build` here) before building MCM Lexicon.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:build
var build embed.FS

// Files is the built frontend, with index.html at its root.
func Files() fs.FS {
	files, err := fs.Sub(build, "build")
	if err != nil {
		panic(err) // "build" is a valid path, so this can't happen
	}
	return files
}
