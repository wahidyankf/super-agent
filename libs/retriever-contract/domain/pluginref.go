package domain

import (
	"errors"
	"regexp"
	"strings"
)

// pluginSHAPattern is the shape of a SHA-256 pin in lower-case hexadecimal.
var pluginSHAPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Plugin pin validation errors. Each text states the requirement and never a value.
var (
	// ErrInvalidPluginPath reports a plugin path that is not absolute.
	ErrInvalidPluginPath = errors.New("invalid plugin path: use an absolute path")
	// ErrInvalidPluginPin reports a plugin pin that is not 64 lower-case hex characters.
	ErrInvalidPluginPin = errors.New("invalid plugin pin: use 64 lower-case hex characters")
)

// PluginRef is the pinned path of one source's out-of-process plugin: the absolute path of the program and the
// SHA-256 of the bytes the user approved. It is data, never a value from the environment, and it holds no secret.
// The host copies the file, hashes the copy, and runs the copy only when the hash equals the pin. Build it with
// NewPluginRef.
type PluginRef struct {
	path   string
	sha256 string
}

// NewPluginRef accepts only an absolute path and a pin of 64 lower-case hex characters. A relative path, an empty
// path, and a misshaped pin are refused, so a typo cannot silently point at another program.
func NewPluginRef(path, sha256 string) (PluginRef, error) {
	if !strings.HasPrefix(path, "/") {
		return PluginRef{}, ErrInvalidPluginPath
	}
	if !pluginSHAPattern.MatchString(sha256) {
		return PluginRef{}, ErrInvalidPluginPin
	}
	return PluginRef{path: path, sha256: sha256}, nil
}

// Path returns the absolute path of the plugin program.
func (r PluginRef) Path() string { return r.path }

// SHA256 returns the approved SHA-256 of the plugin bytes, in lower-case hexadecimal.
func (r PluginRef) SHA256() string { return r.sha256 }
