// Package synthetic holds the synthetic Outline replies that the behaviour harness serves. The files are embedded at
// build time, so a test reads no file at run time, and any test package may import them.
package synthetic

import (
	"embed"
	"io/fs"
)

// replies holds one JSON reply per Outline method that the write and admin commands call, named <method>.json.
//
//go:embed *.json
var replies embed.FS

// Replies returns the reply files at the root of the returned file system, one <method>.json for each method.
func Replies() fs.FS {
	return replies
}
