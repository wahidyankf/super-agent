---
description: >-
  Sets the Go standard for the command-line app: the Go version, the module path, tools pinned as Go tool dependencies,
  the layering, error handling, panics, and the test layout.
when_to_use: >-
  Use when writing, reviewing, or changing Go code under apps/ or libs/, or when adding a Go dependency or tool.
---

# Golang Standards

These rules apply to the Go code in `apps/retriever-cli`, and to any Go library that the repository adds later.

## Version and Module

- The `go.mod` file sets `go 1.27.0`, the minimum language version, and `toolchain go1.27.4`, the Go release the CI setup-go
  action pins. The lines differ on purpose; see the local departure below.
- The module path is `github.com/wahidyankf/super-agent/apps/retriever-cli`.

### Local Departure: The Go Line Stays at 1.27.0

This module departs from the pinned-release form in one way: its `go` line is one patch release below its `toolchain`
line. The working form is `go 1.27.0` with `toolchain go1.27.4`, because the go command refuses the other pairing. With
`go 1.27.4` beside `toolchain go1.27.4`, `go build` fails with "updates to go.mod needed", and `go mod tidy` removes the
toolchain line, so the pin is lost. Under Go 1.27.4, the working form builds, and `go mod tidy` leaves its `go.mod`
unchanged. Revisit this departure when the go command accepts the pairing.

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

- Behaviour scenarios run with godog, pinned at v0.16.0, as described in [Behaviour-Driven
  Development](../../behaviour-driven-development.md).
- Decisions have plain Go unit tests beside their package, run with `-race`. A test that uses `t.TempDir`, `t.Setenv`,
  a real listener, or a child process is not a unit test; it is integration under the [three-level testing
  standard](../three-level-testing-standard.md).
- Unit tests carry no build tag and run in process. Go coverage counts statements, so no branch figure is claimed. One
  instrumented unit run measures the whole module (`-coverpkg` and `-coverprofile`), and only that figure is held to the
  floor in [Meaningful Coverage](../testing/meaningful-coverage.md). Integration and end-to-end runs stay outside it.
- Contract behaviour is a Gherkin scenario and a developer detail is a plain test, per [Behaviour-Driven
  Development](../../behaviour-driven-development.md#scenario-or-plain-test).

### Selecting a Level

This repository selects the integration and end-to-end levels by a build tag on each file, the option the
`ose-rules` catalog's Go standard lists beside a separate test directory:

- Integration tests carry `//go:build integration`, and tests that run the built binary or a real service carry
  `//go:build e2e`.
- The cost is that a forgotten tag moves a test into the unit run. The unit run therefore bans skipped tests, and the
  lint and type-check targets compile every tag, so a tagged file cannot rot.
- A tag selects a level and never hides a test: each tag has a target that runs it, and the pipeline runs that target.
