package behaviour

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/cucumber/godog"
)

// registerConnectorSteps binds the steps of the connector spec behaviours. Each one writes a spec file into the scenario
// folder, or checks how many requests a spec source received. The writes are the same at every level.
func registerConnectorSteps(ctx *godog.ScenarioContext, cur func() *world) {
	ctx.Step(`^the connectors folder "([^"]*)" holds the example wiki spec as "([^"]*)"$`, func(dir, name string) error {
		return cur().writeSpec(dir, name, wikiSpec)
	})
	ctx.Step(`^the connectors folder "([^"]*)" holds the example wiki spec with the CLI off and MCP on as "([^"]*)"$`,
		func(dir, name string) error {
			return cur().writeSpec(dir, name, specCLIOffMCPOn())
		})
	ctx.Step(`^the connectors folder "([^"]*)" holds the example wiki spec with the CLI on and MCP off as "([^"]*)"$`,
		func(dir, name string) error {
			return cur().writeSpec(dir, name, specCLIOnMCPOff())
		})
	ctx.Step(`^the connectors folder "([^"]*)" holds the example wiki spec with an Authorization header as "([^"]*)"$`,
		func(dir, name string) error {
			return cur().writeSpec(dir, name, specWithAuthHeader())
		})
	ctx.Step(`^the connectors folder "([^"]*)" holds the example wiki spec with a Markdown converter as "([^"]*)"$`,
		func(dir, name string) error {
			return cur().writeSpec(dir, name, specWithConverter())
		})
	ctx.Step(`^the connectors folder "([^"]*)" holds the example wiki spec as "([^"]*)" with the kind "([^"]*)"$`,
		func(dir, name, kind string) error {
			return cur().writeSpec(dir, name, specWithKind(kind))
		})
	ctx.Step(`^the spec source received no requests$`, func() error {
		return cur().checkSpecRequests(0)
	})
	ctx.Step(`^the spec source received (\d+) requests?$`, func(want string) error {
		var n int
		if _, err := fmt.Sscanf(want, "%d", &n); err != nil {
			return err
		}
		return cur().checkSpecRequests(n)
	})
}

// writeSpec writes one spec file into the connectors folder dir, which is relative to the scenario folder. The spec is an
// input that the user wrote, as the catalog is, so the secret scan skips it: the scan looks for values that retriever
// writes, and a spec may hold the header text a scenario checks for.
func (w *world) writeSpec(dir, name, text string) error {
	rel := path.Join(dir, name)
	w.inputs[rel] = true
	return w.be.writeFile(w.absPath(rel), []byte(text))
}

// checkSpecRequests checks how many requests the spec source of the scenario received. The unit and integration levels count
// them at the source: the replay stub or the loopback stub. The end-to-end level has no stub to count at, so a check that
// expects no request holds only when the last run's standard error names no network failure, which a request that left
// would have raised. A check that expects a request cannot be made there.
func (w *world) checkSpecRequests(want int) error {
	if w.level == levelE2E {
		if want != 0 {
			return errors.New("the built binary has no spec stub, so a count of requests cannot be checked at the end-to-end level")
		}
		last, err := w.lastRun()
		if err != nil {
			return err
		}
		for _, failure := range []string{".unreachable\"", "network_refused"} {
			if strings.Contains(last.stderr, failure) {
				return fmt.Errorf("the run reached the network: its standard error names %s", failure)
			}
		}
		return nil
	}
	got := 0
	if w.spec != nil {
		got = w.spec.requests()
	}
	if got != want {
		return fmt.Errorf("the spec source received %d requests, want %d", got, want)
	}
	return nil
}
