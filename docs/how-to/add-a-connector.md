---
title: Add a connector
description: >-
  Steps for a contributor to add a source connector as a new library module, wire it into the app, and test it.
category: how-to
tags:
  - how-to
  - contributing
  - connector
  - go
---

# Add a connector

This guide adds a source connector, such as one for a new wiki or a new test-management server. A connector is a
library module under `libs/`, and the app wires it in. Read [library layering](../explanation/library-layering.md)
first: it states what the new module may import. This guide uses Kiwi as the model.

Adding a source kind changes the contract. The contract's version rises by a minor version in v0.x, and the
[changelog](../../CHANGELOG.md) gets an entry. Plan the change with that in mind.

## 1. Add the kind to the contract

In `libs/retriever-contract/domain/sourcekind.go`:

- add a `SourceKind` variable for the new kind, with its name and its display name;
- add the variable to the list that `BuiltinKinds` returns, and set `usesIdentity` when the source needs an account
  e-mail beside its key; and
- add the name to the text of `ErrUnknownSourceKind`, which the flags and the GraphQL arguments show.

The composition root registers the built-in set. A catalog that names a kind outside the registered set is refused,
and the message lists the registered kinds in alphabetical order.

## 2. Add the operation rows

Add a `rows_<name>.go` file in `libs/retriever-contract/operations/`, with one row for each operation the source
offers. Add the slice to the list in `builtinRows`, in `catalog.go`, so the built-in catalog holds it.
`rows_kiwi.go` and `rows_jira.go` show the form. A row names the operation, its parameters, and its effect, and the
command-line tool and the MCP tools are built from it. If a row needs a parameter kind that does not exist, add a
validator and a test for it, as the package comment asks.

## 3. Create the module

Create `libs/retriever-connector-<name>/` with these files:

- `go.mod`: the module path `github.com/wahidyankf/super-agent/libs/retriever-connector-<name>`, `go 1.27.0`,
  `toolchain go1.27.4`, and requirements on the contract and the testkit at the current version. Each requirement gets a
  `replace` to `../retriever-contract` or `../retriever-testkit`. Copy `libs/retriever-connector-kiwi/go.mod`.
- `project.json`: copy `libs/retriever-connector-kiwi/project.json`. Change the names and paths, set the tags to
  `type:connector`, `lang:go`, `domain:retriever`, and `source:<name>`, and list `retriever-contract` and
  `retriever-testkit` in `implicitDependencies`.
- `scripts/coverage-exclusions.tsv`: a header with no entry. Copy the kiwi file.
- `README.md` and `scripts/README.md`. Copy the kiwi pages and change them.

## 4. Write the source

Put the code in one package, such as `libs/retriever-connector-<name>/<name>/`. Implement the ports in
`libs/retriever-contract/ports/sources.go`:

- `SourceGateway` for the reads that the gateway serves: listing collections, listing a tree, and fetching a document.
- `SourceReader` for the read operations of the catalog. Return `domain.ErrUnsupportedOperation` for an operation the
  source does not serve.
- `SourceWriter` only when the source changes data. A read sent to the writer returns `domain.ErrWrongOperationClass`.

Keep the client, the decoders, and the errors in the module. Read the key through `domain.ApiKey`, and never print it.

## 5. Add a test seam and a stub

A connector's tests run against a stub on loopback. Follow Kiwi:

- a `seam.go` with a `TestSeamFromEnv` function that accepts a loopback address only, and that the app's composition
  root calls with the real environment;
- a stub in `libs/retriever-testkit/<name>stub/`, which answers the source's requests from a dataset; and
- a command in `libs/retriever-testkit/<name>stubcmd/`, when the end-to-end suite needs the stub as a process.

## 6. Run the source contract

Add a `contract_test.go` in the connector that runs the `sourcecontract` suite, as the Kiwi and Outline modules do. The
suite is in [libs/retriever-testkit/sourcecontract](../../libs/retriever-testkit/sourcecontract/README.md).

## 7. Wire it into the app

In `apps/retriever-cli/cmd/retriever/wiring.go`, add the new kind in three places:

- `seamKinds`, the list of kinds that have a seam;
- the switch that reads each kind's seam; and
- the registry map of builders, which gives the runtime a gateway for each kind.

The composition root is the only place that names every kind. The runtime sees the map of builders, never a kind's
package.

## 8. Declare the dependency in the app

- Add the module to `apps/retriever-cli/go.mod`, with a requirement and a `replace` to
  `../../libs/retriever-connector-<name>`.
- Add `retriever-connector-<name>` to `implicitDependencies` in `apps/retriever-cli/project.json`.

The architecture test fails when an import has no matching `implicitDependencies` entry.

## 9. Register the project

Add an entry under `extensions.software-development.projects` in `repo-config.yml`, with the path, `kind: connector`,
and `stacks: [golang]`, as the existing entries do. RHINO accepts any `kind` value, so the entry is informational.

## 10. Write the behaviour specs

Add feature files under `specs/behaviours/<name>/`. Tag each scenario `@unit`, and tag it `@integration` or `@e2e`, or
use the exempt tag with its justification. The [test gates](../reference/retriever-cli-test-gates.md) page states the
rules. Then update `apps/retriever-cli/behaviour-coverage.json`, so that it matches the feature files.

## 11. Run the gates

Run the new module's targets from the repository root:

```sh
npx nx run retriever-connector-<name>:lint
npx nx run retriever-connector-<name>:typecheck
npx nx run retriever-connector-<name>:test:unit
npx nx run retriever-connector-<name>:test:integration
```

Then run the app's `test:unit`, `test:conformance`, and `bdd:check` targets, and the pull-request gate, as
[Run the quality gates](./run-the-quality-gates.md) describes.

The continuous-integration Go job runs `lint`, `typecheck`, `test:unit`, and `test:integration` for every Go project
on Ubuntu, so a library's targets run there too. The pre-push gate runs no Go target, so run the library targets
yourself before you push.

## 12. Record the change

Add an entry to the `Unreleased` section of the [changelog](../../CHANGELOG.md). A new kind is a contract change, so the
entry names the minor version that the contract moves to. Update the
[catalog format](../reference/retriever-cli-catalog-format.md) and the
[command reference](../reference/retriever-cli-commands.md) when the source adds a kind or a command that a user sees.
