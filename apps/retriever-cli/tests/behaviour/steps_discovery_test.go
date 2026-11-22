package behaviour

import (
	"errors"
	"fmt"
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
	ctx.Step(`^the profile "([^"]*)" is a symbolic link to a folder that holds these sources:$`, func(name string, tbl *godog.Table) error {
		return cur().linkProfile(name, tbl)
	})
	ctx.Step(`^the standard error does not name the profile folder$`, func() error {
		return cur().checkProfileNotNamed()
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

// linkProfile makes the profile folder a symbolic link to a real folder that holds the sources. The profiles folder stays
// real, as it is when a user links one profile, and the link's target is a folder inside the scenario.
func (w *world) linkProfile(name string, tbl *godog.Table) error {
	rows, err := tableRows(tbl)
	if err != nil {
		return err
	}
	target := path.Join("links", name)
	if err := w.be.mkdir(w.absPath(target)); err != nil {
		return err
	}
	if _, err := w.writeInput(path.Join(target, "sources.yml"), renderCatalog(rows)); err != nil {
		return err
	}
	if err := w.be.mkdir(w.absPath(path.Dir(profileFolder(name)))); err != nil {
		return err
	}
	if err := w.be.symlink(w.absPath(target), w.absPath(profileFolder(name))); err != nil {
		return err
	}
	w.linkedProfile = name
	return nil
}

// checkProfileNotNamed checks that the standard error of the last command names neither the linked profile nor its folder.
func (w *world) checkProfileNotNamed() error {
	if w.linkedProfile == "" {
		return errors.New("no profile was made a symbolic link in this scenario")
	}
	text, err := w.streamText("stderr")
	if err != nil {
		return err
	}
	for _, named := range []string{w.linkedProfile, w.absPath(profileFolder(w.linkedProfile))} {
		if strings.Contains(text, named) {
			return fmt.Errorf("the standard error names the profile folder %q: %q", named, text)
		}
	}
	return nil
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
