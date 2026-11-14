package behaviour

import (
	"context"
	"testing"

	"github.com/cucumber/godog"
)

// featureDir is the behaviour specs, relative to this package at apps/retriever-cli/tests/behaviour.
const featureDir = "../../../../specs/behaviours"

// registerSteps binds every step to the running scenario's world. Each step is defined once, and every level uses it.
// A step that a level cannot satisfy fails that scenario, so no level silently skips a step.
func registerSteps(ctx *godog.ScenarioContext, cur func() *world) {
	registerCLISteps(ctx, cur)
	registerStubSteps(ctx, cur)
	registerProcessSteps(ctx, cur)
	registerReadSteps(ctx, cur)
	registerConnectorSteps(ctx, cur)
}

// runSuite runs the scenarios of one level, the ones that carry its tag. Strict mode fails the run on any undefined or
// pending step. Scenarios run one at a time, because an in-process level changes the working directory and the environment of this process at the disk levels.
func runSuite(t *testing.T, lvl level, tag string, newBackend func(*testing.T) backend) {
	t.Helper()
	var w *world
	cur := func() *world { return w }
	suite := godog.TestSuite{
		Name: "retriever-" + string(lvl),
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			ctx.Before(func(bctx context.Context, sc *godog.Scenario) (context.Context, error) {
				w = newWorld(lvl, t, newBackend(t))
				if err := w.setup(); err != nil {
					return bctx, err
				}
				var texts []string
				for _, step := range sc.Steps {
					texts = append(texts, step.Text)
				}
				return bctx, w.startNamedStubs(texts)
			})
			ctx.After(func(actx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
				if w != nil {
					w.teardown()
				}
				return actx, nil
			})
			registerSteps(ctx, cur)
		},
		Options: &godog.Options{
			Format:      "pretty",
			Paths:       []string{featureDir},
			Tags:        tag,
			Strict:      true,
			TestingT:    t,
			Concurrency: 1,
			NoColors:    true,
		},
	}
	if status := suite.Run(); status != 0 {
		t.Fatalf("the %s behaviour suite failed with status %d", lvl, status)
	}
}
