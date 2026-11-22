package behaviour

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/cucumber/godog"
)

// pluginMarker is the file a plugin script writes when it runs, so a disk level sees that a process started. The unit
// level has no script, and it counts the starts of its fake instead.
const pluginMarker = "plugin-started"

// The replies a plugin script writes. The collection list answers the call. The other reply carries id 2, which the host
// refuses, because it sends id 1.
const (
	pluginCollectionReply = `{"jsonrpc":"2.0","id":1,"result":{"collections":[{"id":"c1","name":"One"}]}}`
	pluginWrongIDReply    = `{"jsonrpc":"2.0","id":2,"result":{}}`
	// pluginNotAProgram is the body of a plugin that cannot start: no kernel runs it, so the start fails.
	pluginNotAProgram = "not a program\n"
)

// pluginFile is the plugin file a scenario wrote: its path relative to the scenario folder, the reply it gives to its one
// request, and whether it cannot start.
type pluginFile struct {
	rel    string
	reply  string
	broken bool
}

// content returns the bytes of the plugin file. A plugin that can start is a POSIX shell script: it writes the marker
// when it runs, reads its one request line, and writes its reply. A plugin that cannot start holds no program.
func (p *pluginFile) content(marker string) string {
	if p.broken {
		return pluginNotAProgram
	}
	return "#!/bin/sh\n: > '" + marker + "'\nread -r request\nprintf '%s\\n' '" + p.reply + "'\n"
}

// pluginCatalogSpec is a sources catalog that names a plugin file. It is written once the plugin file holds the bytes its
// pin is taken from, so the steps may come in either order.
type pluginCatalogSpec struct {
	name    string
	rel     string
	pinFile bool
}

// registerPluginSteps binds the steps of the plugin scenarios: a sources catalog that names a plugin, the plugin file and
// its reply or its start, and the check that no plugin process started.
func registerPluginSteps(ctx *godog.ScenarioContext, cur func() *world) {
	ctx.Step(`^the sources catalog file "([^"]*)" holds one outline source served by the plugin "([^"]*)" pinned to (a digest of 64 zeros|the digest of its file)$`,
		func(name, rel, pin string) error {
			return cur().setPluginCatalog(name, rel, pin == "the digest of its file")
		})
	ctx.Step(`^the plugin file "([^"]*)" answers every request with one collection list$`, func(rel string) error {
		return cur().writePluginFile(rel, pluginCollectionReply, false)
	})
	ctx.Step(`^the plugin file "([^"]*)" answers every request with a reply whose id is not the request id$`, func(rel string) error {
		return cur().writePluginFile(rel, pluginWrongIDReply, false)
	})
	ctx.Step(`^the plugin file "([^"]*)" cannot start$`, func(rel string) error {
		return cur().writePluginFile(rel, "", true)
	})
	ctx.Step(`^no plugin process has started$`, func() error {
		return cur().checkNoPluginStart()
	})
}

// setPluginCatalog records the catalog that names the plugin file, and writes it when the file's bytes are known.
func (w *world) setPluginCatalog(name, rel string, pinFile bool) error {
	w.pluginCatalog = &pluginCatalogSpec{name: name, rel: rel, pinFile: pinFile}
	return w.flushPluginCatalog()
}

// writePluginFile writes the plugin file of the scenario and makes it the plugin the fake or the real start runs.
func (w *world) writePluginFile(rel, reply string, broken bool) error {
	w.pluginFile = &pluginFile{rel: rel, reply: reply, broken: broken}
	if _, err := w.writeInput(rel, w.pluginFile.content(w.absPath(pluginMarker))); err != nil {
		return err
	}
	return w.flushPluginCatalog()
}

// flushPluginCatalog writes the plugin catalog when it can be written: at once for a literal pin, and once the plugin file
// exists for a pin taken from the file's digest. It writes nothing while the file is missing.
func (w *world) flushPluginCatalog() error {
	spec := w.pluginCatalog
	if spec == nil {
		return nil
	}
	pin := strings.Repeat("0", 64)
	if spec.pinFile {
		if !w.be.regularFile(w.absPath(spec.rel)) {
			return nil
		}
		data, err := w.be.readFile(w.absPath(spec.rel))
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		pin = hex.EncodeToString(sum[:])
	}
	_, err := w.writeInput(spec.name, renderPluginCatalog(w.absPath(spec.rel), pin))
	return err
}

// renderPluginCatalog renders the catalog of one outline source served by the plugin at the absolute path plugin, pinned
// to the digest pin.
func renderPluginCatalog(plugin, pin string) string {
	return "version: 1\nsources:\n" +
		"  - id: example-outline\n" +
		"    kind: outline\n" +
		"    tags: [wiki, docs]\n" +
		"    endpoint: https://docs.example.test\n" +
		"    credential_env: EXAMPLE_OUTLINE_API_KEY\n" +
		"    plugin:\n" +
		"      path: " + plugin + "\n" +
		"      sha256: " + pin + "\n"
}

// checkNoPluginStart checks that no plugin process started in the scenario: the fake starts of the unit level, or the
// marker file of a disk level.
func (w *world) checkNoPluginStart() error {
	if n := w.be.pluginStarts(); n != 0 {
		return fmt.Errorf("%d plugin process(es) started, and none may start", n)
	}
	return nil
}
