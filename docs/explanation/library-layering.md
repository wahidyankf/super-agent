---
title: Library layering
description: >-
  Which Go module may import which in the libraries split, why the contract sits at the bottom, and what a connector
  change touches.
category: explanation
tags:
  - explanation
  - go
  - architecture
  - libraries
---

# Library layering

The Go code of super-agent is one app and seven libraries. Each library is its own Go module, so an import rule is
also a rule between modules. This page explains the layers, the rules between them, and why the rules exist. The module
paths, versions, and tags are in [Go modules and tags](../reference/go-modules-and-tags.md).

## The layers

The contract is the bottom layer. Every other module depends on it, and it depends on no other module of this
repository. An arrow in the diagram points from the importer to the module it imports.

```text
                       retriever-cli (the app)
                        |           |          |
                        |           |          +----------------+
                        v           v                           v
                 retriever-runtime  retriever-connector-outline  retriever-connector-kiwi
                        |           retriever-connector-jira
                        |                 |
                        +--------+--------+-----------------------+
                                 v
                        retriever-contract

    retriever-testkit ----> retriever-contract        (production code of the testkit)
    tests of the app, the runtime, and the connectors ----> retriever-testkit   (test code only)
```

The app also imports the contract directly, and so do the runtime and the connectors. The diagram leaves those arrows
out to keep it readable.

## Who may import whom

| Module                    | Production code may import                | Test code may also import  |
| ------------------------- | ----------------------------------------- | -------------------------- |
| `retriever-contract`      | nothing in this repository                | nothing in this repository |
| `retriever-runtime`       | `retriever-contract`                      | `retriever-testkit`        |
| `retriever-connector-*`   | `retriever-contract`                      | `retriever-testkit`        |
| `retriever-testkit`       | `retriever-contract`                      | nothing else               |
| `retriever-spec`          | `retriever-contract`                      | `retriever-testkit`        |
| `retriever-cli` (the app) | `retriever-contract`, runtime, connectors | `retriever-testkit`        |

These rules follow from them:

- No module imports the app.
- A runtime does not import a connector, and a connector does not import the runtime or another connector.
- The contract imports no other module, in production code or in tests.
- Production code never imports the testkit.
- The contract, the spec module, and the runtime's `sources` package reach the outside world only
  through a port. Their production code
  imports none of the standard packages that reach outside the process: `net`, `net/http`, `os`, `os/exec`,
  `path/filepath`, `io/fs`, `log`, and `syscall`.

Third-party modules sit outside these rules. The runtime uses `go.yaml.in/yaml/v3` to parse the catalog, for example.

## Why the rules exist

**A connector change touches only itself and the app.** A connector imports only the contract in production code. No
other connector and no runtime imports it. A change to the Outline connector therefore changes the Outline package and
the app that wires it. The Kiwi and Jira connectors do not change, the runtime does not change, and the contract does
not change.

**The contract is the stable part.** Every module depends on it, so it depends on nothing. Its version is the
`ContractVersion` constant, and a contract change bumps the minor version in v0.x. The [changelog](../../CHANGELOG.md)
records each such change.

**The kinds and the rows are values, not a registry.** The contract keeps no global list that a package appends to at
start-up. The composition root builds a `KindSet` and a row `Catalog`, and it passes them to the runtime. A new kind is
one more definition in the set that the root builds.

**The runtime works through ports.** Its use cases call a `SourceGateway`, a `SourceReader`, or a `SourceWriter`, and
the app supplies the concrete one through a builder in the registry. The runtime never names a connector, so a new
connector needs no change in the runtime.

**Test code stays out of the product.** The testkit holds the stubs, the in-memory file tree, and the source contract
suite. Only tests import it, so no stub code enters the production dependency graph.

**Affected builds stay exact.** Nx cannot read Go imports. Each Nx project therefore lists the modules it imports in
`implicitDependencies`, and `nx affected` uses that list to choose what to rebuild. The architecture test checks that
each list names every module the Go files import, so a missing edge fails a test instead of skipping a build.

## How the rules are checked

The architecture tests in [apps/retriever-cli/tests/architecture](../../apps/retriever-cli/tests/architecture/README.md)
apply the table to the Go source with the parser. The rule table has its own tests, which run without a build tag. The
walk over the real files carries the `integration` tag, and it runs in the `test:conformance` target. The tests also
check that each module's `go.mod` requires and replaces the siblings it imports.

## Known exceptions

The rules above cover the contract and the runtime's `sources` package. They do not cover the rest of the libraries, so
two exceptions exist:

- Each connector's `seam.go` has an entry point, `TestSeamFromEnv`, which calls `os.ReadFile` directly. The composition
  root calls it with the real environment, and only for tests. The seam accepts a loopback address only.
- The runtime's other packages, such as `catalog` and `markdownfiles`, name `os` types and take the file system as a
  parameter. Their production code calls no `os` function that reaches the disk. The app's composition root passes the
  real calls in.

The [test gates](../reference/retriever-cli-test-gates.md) page states the coverage and hermetic rules that apply to
these packages.
