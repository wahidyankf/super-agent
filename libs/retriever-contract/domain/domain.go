// Package domain holds the plain types the retriever use cases share. It depends on
// the standard library only, so no adapter detail leaks into the core.
package domain

// OutputFormat is the rendering a command was asked for, selected by the global --output flag.
type OutputFormat int

const (
	// FormatText is the default human-readable rendering.
	FormatText OutputFormat = iota
	// FormatJSON renders the result as one JSON document with a schemaVersion field.
	FormatJSON
)
