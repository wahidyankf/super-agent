package behaviour

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/catalog"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/envfile"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/out/markdownfiles"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/tests/support/memfs"
)

// backend is where one level keeps a scenario: its files, its working directory, the process environment, and the
// production adapters that read them. Every path it takes is absolute. The unit level keeps all of it in memory, so its
// scenarios read and write no disk and change no process state. The integration and end-to-end levels keep it on disk,
// in the files of backend_disk_test.go, which the build tags select.
type backend interface {
	// start creates the scenario folder and returns its absolute path.
	start() (string, error)
	// stop undoes start: it restores the working directory and the process environment of an in-process level.
	stop()
	// inProcess reports whether commands run in this process, rather than as the built binary.
	inProcess() bool
	// adoptEnv makes the process environment hold exactly env. Only the disk level reads the process environment.
	adoptEnv(env map[string]string)
	// getwd returns the working directory of the in-process level.
	getwd() (string, error)
	// writeFile creates or replaces the file at path, with its folder.
	writeFile(path string, data []byte) error
	// mkdir creates the folder at path, with its parents.
	mkdir(path string) error
	// readFile returns the bytes of the file at path.
	readFile(path string) ([]byte, error)
	// regularFile reports whether a regular file exists at path.
	regularFile(path string) bool
	// walk returns the regular files under root, keyed by slash path relative to root.
	walk(root string) (map[string]string, error)
	// dirFiles returns the regular files directly inside the folder at path. A missing folder holds none.
	dirFiles(path string) (map[string]string, error)
	// credentials returns the env-file reader of the file at path, which reads the environment through getenv.
	credentials(path string, getenv func(string) string) (*envfile.Credentials, error)
	// loadCatalog reads the sources catalog at path.
	loadCatalog(path string) (domain.Catalog, error)
	// sink returns the export sink that writes into dir.
	sink(dir string) ports.ExportSink
	// spawnServe starts retriever serve as the built binary. Only the end-to-end level has a binary.
	spawnServe(w *world, argv []string) (*serveSession, error)
	// spawnMCP starts retriever mcp as the built binary.
	spawnMCP(w *world, argv []string) (*mcpSession, error)
	// runBinary runs the built binary once and records its result.
	runBinary(w *world, argv []string) error
}

// errNoProcess is the error of a process at a level that runs commands in this process.
var errNoProcess = errors.New("this level runs commands in process and starts no binary")

// memRoot is the scenario folder of the in-memory level. It is absolute, as the configuration rules require.
const memRoot = "/scenario"

// memoryBackend is the in-memory level's backend. Its files live in one memfs tree, and the working directory is the
// scenario folder. No disk, environment, or child process is touched.
type memoryBackend struct {
	files *memfs.FS
}

// newMemoryBackend returns an empty in-memory backend for one scenario.
func newMemoryBackend(*testing.T) backend {
	return &memoryBackend{files: memfs.New()}
}

// start creates the scenario folder in the tree and returns it.
func (m *memoryBackend) start() (string, error) {
	if err := m.files.MkdirAll(memRoot); err != nil {
		return "", err
	}
	return memRoot, nil
}

// stop does nothing, because the in-memory level changed no process state.
func (m *memoryBackend) stop() {}

// inProcess reports true: the in-memory level runs every command in this process.
func (m *memoryBackend) inProcess() bool { return true }

// adoptEnv does nothing. The in-memory level reads the scenario environment through the getenv it is given.
func (m *memoryBackend) adoptEnv(map[string]string) {}

// getwd returns the scenario folder, which is the working directory of the in-memory level.
func (m *memoryBackend) getwd() (string, error) { return memRoot, nil }

// writeFile stores data at path, creating its folder first.
func (m *memoryBackend) writeFile(name string, data []byte) error {
	if err := m.files.MkdirAll(path.Dir(name)); err != nil {
		return err
	}
	return m.files.WriteFile(name, data)
}

// mkdir creates the folder at name.
func (m *memoryBackend) mkdir(name string) error { return m.files.MkdirAll(name) }

// readFile returns the bytes stored at name.
func (m *memoryBackend) readFile(name string) ([]byte, error) { return m.files.ReadFile(name) }

// regularFile reports whether a regular file is stored at name.
func (m *memoryBackend) regularFile(name string) bool { return m.files.IsRegular(name) }

// walk returns every file under root.
func (m *memoryBackend) walk(root string) (map[string]string, error) {
	files, err := m.files.Walk(root)
	if err != nil {
		return nil, err
	}
	return toText(files), nil
}

// dirFiles returns the files directly inside the folder at name.
func (m *memoryBackend) dirFiles(name string) (map[string]string, error) {
	files, err := m.files.Files(name)
	if err != nil {
		return nil, err
	}
	return toText(files), nil
}

// toText turns the bytes of each file into text.
func toText(files map[string][]byte) map[string]string {
	text := make(map[string]string, len(files))
	for name, data := range files {
		text[name] = string(data)
	}
	return text
}

// credentials reads the env file through the given lookup and the in-memory files, as envfile.NewWithFiles does.
func (m *memoryBackend) credentials(name string, getenv func(string) string) (*envfile.Credentials, error) {
	return envfile.NewWithFiles(name, getenv, m.envFiles())
}

// envFiles are the file calls of the in-memory tree, in the form the envfile adapter takes.
func (m *memoryBackend) envFiles() envfile.Files {
	return envfile.Files{Lstat: m.files.Lstat, Open: m.files.Open, SameFile: m.files.SameFile}
}

// loadCatalog reads the catalog from the in-memory tree, as catalog.NewWithFiles does.
func (m *memoryBackend) loadCatalog(name string) (domain.Catalog, error) {
	loader, err := catalog.NewWithFiles(name, catalog.Files{
		Lstat:    m.files.Lstat,
		Open:     m.files.Open,
		SameFile: m.files.SameFile,
	})
	if err != nil {
		return domain.Catalog{}, err
	}
	return loader.Load()
}

// sink returns an export sink that stores each exported file in the in-memory tree. A relative folder is taken from the
// working directory, which is the scenario folder, as the Markdown sink takes it from the process working directory.
func (m *memoryBackend) sink(dir string) ports.ExportSink {
	if dir != "" && !path.IsAbs(dir) {
		dir = path.Join(memRoot, dir)
	}
	return &memorySink{files: m.files, dir: dir}
}

// spawnServe fails: the in-memory level has no binary to start.
func (m *memoryBackend) spawnServe(*world, []string) (*serveSession, error) { return nil, errNoProcess }

// spawnMCP fails: the in-memory level has no binary to start.
func (m *memoryBackend) spawnMCP(*world, []string) (*mcpSession, error) { return nil, errNoProcess }

// runBinary fails: the in-memory level has no binary to run.
func (m *memoryBackend) runBinary(*world, []string) error { return errNoProcess }

// memorySink is the export sink of the in-memory level. It keeps the contract of the Markdown sink: a file name must be
// one plain name that ends in .md, a write replaces a file of the same name, and the folder is created on the first
// write. The integration level runs the real sink on the same scenarios.
type memorySink struct {
	files *memfs.FS
	dir   string
}

// Write stores data as the named file in the sink's folder, and returns the path it stored.
func (s *memorySink) Write(_ context.Context, name string, data []byte) (string, error) {
	if s.dir == "" {
		return "", markdownfiles.ErrNoFolder
	}
	if err := checkExportName(name); err != nil {
		return "", err
	}
	if err := s.files.MkdirAll(s.dir); err != nil {
		return "", fmt.Errorf("creating the export folder: %w", err)
	}
	target := path.Join(s.dir, name)
	if err := s.files.WriteFile(target, data); err != nil {
		return "", fmt.Errorf("placing the export: %w", err)
	}
	return target, nil
}

// checkExportName accepts one plain file name ending in .md, as the Markdown sink does.
func checkExportName(name string) error {
	switch {
	case name == "", name == ".", name == "..":
		return markdownfiles.ErrBadName
	case strings.ContainsAny(name, "/\\\x00"):
		return markdownfiles.ErrBadName
	case !strings.HasSuffix(name, ".md") || len(name) <= len(".md"):
		return markdownfiles.ErrBadName
	}
	return nil
}
