package behaviour

import (
	"net"
	"path"
	"strconv"
	"strings"

	"github.com/cucumber/godog"
)

// heldPorts holds, per scenario, the loopback port that another process holds. A command line names it as {held-port}.
var heldPorts = map[*world]string{}

// registerDiscoverySteps binds the steps of the profile, home-folder, env-file, and oversize-spec scenarios. Each level
// uses them the same way: a profile is a folder under the scenario's configuration folder, and a held port is a real
// loopback listener that lasts until the scenario's outer test ends.
func registerDiscoverySteps(ctx *godog.ScenarioContext, cur func() *world) {
	ctx.Step(`^the profile "([^"]*)" holds these sources:$`, func(name string, tbl *godog.Table) error {
		return cur().writeProfileCatalog(name, tbl)
	})
	ctx.Step(`^the profile "([^"]*)" holds these lines in its env file:$`, func(name string, doc *godog.DocString) error {
		return cur().writeEnvFileIn(profileFolder(name), ".env.local", doc.Content)
	})
	ctx.Step(`^the sources catalog in the home configuration folder holds these sources:$`, func(tbl *godog.Table) error {
		return cur().writeHomeCatalog(tbl)
	})
	ctx.Step(`^the env file "([^"]*)" holds the line (.+)$`, func(name, line string) error {
		return cur().writeEnvFile(name, line+"\n")
	})
	ctx.Step(`^another process holds a loopback port$`, func() error {
		return holdLoopbackPort(cur())
	})
	ctx.Step(`^the spec file "([^"]*)" is (\d+) KiB large$`, func(name, kib string) error {
		return cur().writeOversizeSpec(name, kib)
	})
}

// profileFolder is the folder of a named profile, inside the scenario's configuration folder.
func profileFolder(name string) string {
	return path.Join("config", "retriever", "profiles", name)
}

// writeProfileCatalog writes the sources catalog of a profile.
func (w *world) writeProfileCatalog(name string, tbl *godog.Table) error {
	rows, err := tableRows(tbl)
	if err != nil {
		return err
	}
	folder := profileFolder(name)
	if err := w.be.mkdir(w.absPath(folder)); err != nil {
		return err
	}
	_, err = w.writeInput(path.Join(folder, "sources.yml"), renderCatalog(rows))
	return err
}

// writeHomeCatalog writes a sources catalog into the folder of the home directory's configuration, which the scenario
// reaches when XDG_CONFIG_HOME is empty.
func (w *world) writeHomeCatalog(tbl *godog.Table) error {
	rows, err := tableRows(tbl)
	if err != nil {
		return err
	}
	folder := path.Join("home", ".config", "retriever")
	if err := w.be.mkdir(w.absPath(folder)); err != nil {
		return err
	}
	_, err = w.writeInput(path.Join(folder, "sources.yml"), renderCatalog(rows))
	return err
}

// holdLoopbackPort listens on a free loopback port of the scenario's network for the rest of the scenario's outer test,
// so a command that asks for the same port finds it busy. The in-memory network and the real one both refuse a busy port.
func holdLoopbackPort(w *world) error {
	ln, err := w.be.listenPort(0)
	if err != nil {
		return err
	}
	w.outer.Cleanup(func() { _ = ln.Close() })
	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		return err
	}
	heldPorts[w] = port
	return nil
}

// substituteHeldPort replaces {held-port} in a command line with the port that another process holds.
func substituteHeldPort(w *world, line string) string {
	return strings.ReplaceAll(line, "{held-port}", heldPorts[w])
}

// writeOversizeSpec writes a connector spec file of the given size in KiB. The bytes are comment lines, so only the size
// rule refuses the file.
func (w *world) writeOversizeSpec(name, kib string) error {
	n, err := strconv.Atoi(kib)
	if err != nil {
		return err
	}
	var b strings.Builder
	line := "# " + strings.Repeat("x", 61) + "\n"
	for b.Len() < n*1024 {
		b.WriteString(line)
	}
	_, err = w.writeInput(name, b.String())
	return err
}
