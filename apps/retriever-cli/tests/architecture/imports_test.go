package architecture

// module is the import path of this module.
const module = "github.com/wahidyankf/super-agent/apps/retriever-cli"

// coreRoot is the core source tree, relative to this package at apps/retriever-cli/tests/architecture.
const coreRoot = "../../internal/core"

// shellPrefixes are the import prefixes the core must never reach: the adapters, the composition root, and the tests.
var shellPrefixes = []string{
	module + "/internal/adapters",
	module + "/cmd",
	module + "/tests",
}

// effectfulStd are the standard-library packages that reach the world outside the process: the network, the file system,
// and processes. The core reaches them only through a port. The log/slog package is not listed: the core imports it only
// to return a redacted value from LogValue, which is pure, and the logger that writes it belongs to the shell.
var effectfulStd = map[string]bool{
	"net":           true,
	"net/http":      true,
	"os":            true,
	"os/exec":       true,
	"path/filepath": true,
	"io/fs":         true,
	"log":           true,
	"syscall":       true,
}
