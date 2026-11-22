package behaviour

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/wahidyankf/super-agent/libs/retriever-plugin"
)

// memPrivateDir is the private folder of the unit level's plugin calls. The in-memory tree keeps no removal, so the copy
// stays in the tree, and no step reads it.
const memPrivateDir = "/private/plugin"

// errPluginCannotStart is the fake start's failure of a plugin the scenario made unable to start.
var errPluginCannotStart = errors.New("the fake plugin cannot start")

// memPluginFiles serves the plugin file calls from the in-memory tree of the scenario, so the pin is taken over the bytes
// the scenario wrote and no real file is read.
type memPluginFiles struct {
	be *memoryBackend
}

// MakePrivateDir returns the private folder of the in-memory level.
func (f memPluginFiles) MakePrivateDir() (string, error) { return memPrivateDir, nil }

// CopyFile copies the bytes of src to dst in the in-memory tree.
func (f memPluginFiles) CopyFile(src, dst string) error {
	data, err := f.be.readFile(src)
	if err != nil {
		return err
	}
	return f.be.writeFile(dst, data)
}

// HashFile returns the SHA-256 of the bytes stored at name, in lower-case hexadecimal.
func (f memPluginFiles) HashFile(name string) (string, error) {
	data, err := f.be.readFile(name)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// RemoveAll does nothing, because the in-memory tree has no removal.
func (f memPluginFiles) RemoveAll(string) error { return nil }

// memPluginProc is the fake process of the unit level. It takes no input and runs nothing, and its one reply is the
// scenario's reply.
type memPluginProc struct {
	reply []byte
}

// Write accepts the request frame and ignores it.
func (memPluginProc) Write([]byte) error { return nil }

// Read returns the scenario's reply frame.
func (p memPluginProc) Read() ([]byte, error) { return p.reply, nil }

// Stderr returns no diagnostics.
func (memPluginProc) Stderr() []byte { return nil }

// Kill does nothing, because the fake runs no process.
func (memPluginProc) Kill() error { return nil }

// Wait returns at once, because the fake runs no process.
func (memPluginProc) Wait() error { return nil }

// The compile-time check that the fake process is a plugin process.
var _ plugin.Proc = memPluginProc{}
