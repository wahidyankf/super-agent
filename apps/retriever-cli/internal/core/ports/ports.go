// Package ports declares the outbound contracts the retriever core depends on.
// Adapters implement these interfaces; tests replace them with fakes.
package ports

import "context"

// FileSystem is the read-only view of files the configuration loader uses. Relative paths resolve against the working
// directory.
type FileSystem interface {
	// ReadFile returns the bytes of one file.
	ReadFile(path string) ([]byte, error)
	// Exists reports whether a file or a directory is present at path.
	Exists(path string) bool
}

// ExportSink stores one exported document. The core chooses the file name from the document's title and ID, and the sink
// chooses where the bytes go. The default sink writes Markdown files into the folder the --out flag names.
type ExportSink interface {
	// Write stores data under name and returns the location it wrote, which the caller reports.
	Write(ctx context.Context, name string, data []byte) (location string, err error)
}

// Ports bundles the outbound ports the composition root supplies to the CLI.
type Ports struct {
	FS FileSystem
}
