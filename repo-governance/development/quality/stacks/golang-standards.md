---
description: >-
  Sets the Go standard for the repository's nine Go modules: the command-line app and its eight libraries.
when_to_use: >-
  Use when writing, reviewing, or changing Go code under apps/ or libs/, or when adding a Go dependency, tool, or
  module.
---

# Golang Standards

These rules apply to the Go code in `apps/retriever-cli` and the six `libs/retriever-*` modules, and to any Go module
the repository adds later.

## Version and Module

- Every `go.mod` sets `go 1.27.0`, the minimum language version, and `toolchain go1.27.4`, the release the CI setup-go
  action pins. The lines differ on purpose; see the local departure below.
- Each module has its own `go.mod`, and no `go.work` file joins them. A module path is
  `github.com/wahidyankf/super-agent/` followed by the module's folder.

### Local Departure: The Go Line Stays at 1.27.0

Every module sets its `go` line one patch release below its `toolchain` line. With `go 1.27.4` beside
`toolchain go1.27.4`, `go build` asks for a go.mod update, and `go mod tidy` drops the toolchain line. Revisit this when
the go command accepts the pairing.

## Tools

- Linters and generators are pinned as Go tool dependencies with the `tool` directive in `go.mod`. Run them with `go
tool <name>`. Nothing is installed globally.
- Adding a dependency follows the [dependency bump policy](../../workflow/dependency-bump-policy.md). The lock file is
  committed with the change.

## Layout

- Use hexagonal layering: a core package decides and receives its inputs as parameters, adapters perform effects, and
  the command layer wires them together. See [Hexagonal Architecture](../../pattern/hexagonal-architecture.md).
- Keep each package small and named for its role.
- Export only what another package uses.

## Errors

- Return errors as values. Wrap an error with context only where its meaning changes.
- Infrastructure error types stop at their adapter. A failure becomes a CLI error code once, at the edge.
- Test each error path. Compare errors with `errors.Is` and `errors.As`, never by their text.

## Output and Logging

- The CLI writes through its output adapter. Core code never calls `fmt.Println` or writes to the standard streams.
- A diagnostic names the input or variable at fault, never a value. The [errors and
  diagnostics](../../../conventions/command-line-interface/006-errors-and-diagnostics.md) module sets the shape.

## Panics

A top-level handler turns any panic into the internal-failure error body with exit status `2`. The handler prints no
stack trace. The [tiers record](../../../conventions/command-line-interface/009-tiers-here.md) states this owner choice.

## Tests

- Decisions have plain Go unit tests beside their package. They carry no build tag and run in process under `-race`.
- Unit tests are hermetic: no file system, no environment reads, no real network or listener, and no real clock. A test
  that uses `t.TempDir`, `t.Setenv`, a real listener, a child process, or the real clock is integration under the
  [three-level testing standard](../three-level-testing-standard.md). The architecture guard enforces the file-system,
  environment, and process parts; review enforces the rest.
- Real `os` calls sit in the thinnest composition root, `apps/retriever-cli/cmd/retriever` for the app. Each wrapper is
  tested under the `integration` tag. The [retriever test gates](../../../../docs/reference/retriever-cli-test-gates.md)
  list the known departures.
- Each module is gated on its own at the 99% unit-coverage floor. The [repository adapter](repository-adapter.md#gates)
  states the gate and its only exclusion.
- For the app, unit coverage counts the scenarios that run at unit level and the plain unit tests outside the behaviour
  suite. For a library, it counts its plain unit tests. Go coverage counts statements, so no branch figure is claimed.
- Integration and end-to-end runs produce no coverage figure, and the pre-push gate runs no Go target.
- Only the app has Gherkin. It runs its behaviour scenarios with godog, pinned at v0.16.0, and chooses scenario or plain
  test as [Behaviour-Driven Development](../../behaviour-driven-development.md) states. A library keeps plain tests and
  no scenarios.

### Selecting a Level

The repository selects the integration and end-to-end levels by a build tag on each file, an option the `ose-rules`
Go standard lists beside a separate test directory:

- Integration tests carry `//go:build integration`, and tests that run the built binary or a real service carry
  `//go:build e2e`.
- A forgotten tag moves a test into the unit run. The unit run therefore bans skipped tests, and the lint and type-check
  targets compile every tag, so a tagged file cannot rot.
- A tag selects a level and never hides a test: each tag has a target that runs it, and the pipeline runs that target.
