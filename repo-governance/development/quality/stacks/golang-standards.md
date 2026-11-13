---
description: >-
  Sets the Go standard for the command-line app: the Go version, the module path, tools pinned as Go tool dependencies,
  the layering, error handling, panics, and the test layout.
when_to_use: >-
  Use when writing, reviewing, or changing Go code under apps/ or libs/, or when adding a Go dependency or tool.
---

# Golang Standards

These rules apply to the Go code in `apps/super-agent-cli`, and to any Go library that the repository adds later.

## Version and Module

- The `go.mod` file sets `go 1.26.1` and `toolchain go1.26.9`.
- The module path is `github.com/wahidyankf/super-agent/apps/super-agent-cli`.

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
- Decisions have plain Go unit tests. Files use `t.TempDir()`, and network tests use a local `httptest` server.
- Build tags do not hide tests.
