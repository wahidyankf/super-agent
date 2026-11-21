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
	registerDiscoverySteps(ctx, cur)
}

// runSuite runs every scenario of the corpus except those that carry the excluded tag. The unit level excludes
// nothing, so its excluded tag is empty. Strict mode fails the run on any undefined or pending step. Scenarios run one
// at a time, because an in-process level changes the working directory and the environment of this process at the disk
// levels. The run also fails when the number of scenarios it executes differs from the number the corpus holds without
// the excluded ones, so a filter cannot silently skip a scenario.
func runSuite(t *testing.T, lvl level, excluded string, newBackend func(*testing.T) backend) {
	t.Helper()
	tags := ""
	if excluded != "" {
		tags = "~" + excluded
	}
	expected := corpusCount(t, excluded)
	executed := 0
	var w *world
	cur := func() *world { return w }
	suite := godog.TestSuite{
		Name: "retriever-" + string(lvl),
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			ctx.Before(func(bctx context.Context, sc *godog.Scenario) (context.Context, error) {
				executed++
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
			Tags:        tags,
			Strict:      true,
			TestingT:    t,
			Concurrency: 1,
			NoColors:    true,
		},
	}
	status := suite.Run()
	t.Logf("the %s level executed %d scenarios; the corpus holds %d without %q", lvl, executed, expected, excluded)
	if status != 0 {
		t.Fatalf("the %s behaviour suite failed with status %d", lvl, status)
	}
	if executed != expected {
		t.Fatalf("the %s behaviour suite executed %d scenarios; the corpus holds %d without %q", lvl, executed, expected, excluded)
	}
}

// corpusCount returns how many scenarios the corpus holds that do not carry the excluded tag. A scenario outline counts
// once for each of its Examples rows, because godog runs each row as its own scenario.
func corpusCount(t *testing.T, excluded string) int {
	t.Helper()
	features, err := (godog.TestSuite{Options: &godog.Options{Paths: []string{featureDir}, Strict: true}}).RetrieveFeatures()
	if err != nil {
		t.Fatalf("parse the behaviour corpus: %v", err)
	}
	count := 0
	for _, feature := range features {
		for _, pickle := range feature.Pickles {
			carries := false
			for _, tag := range pickle.Tags {
				if excluded != "" && tag.Name == excluded {
					carries = true
				}
			}
			if !carries {
				count++
			}
		}
	}
	return count
}
