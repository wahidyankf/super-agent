# Retriever CLI test gates

This page records what each test target of `apps/retriever-cli` runs, what the coverage gate measures and excludes, and
what the binding check reports. It states facts. The rules that govern them live in the repository's standards.

## Targets

| Target             | Runs                                                                       |
| ------------------ | -------------------------------------------------------------------------- |
| `lint`             | `go vet` with the integration and e2e tags, then a `gofmt` check           |
| `typecheck`        | `go build`, then compiles every test binary with those tags                |
| `test:unit`        | `go test` without tags, with a check that no test skips                    |
| `test:quick`       | `lint`, `typecheck`, `test:unit`                                           |
| `test:integration` | `go test -tags integration ./...`                                          |
| `test:e2e`         | `go test -tags e2e ./tests/behaviour/...`, which builds the binary         |
| `test:conformance` | the conformance and architecture packages, without tags                    |
| `bdd:check`        | the binding check of `specs/behaviours` (build tag `bdd`)                  |
| `test:coverage`    | unit report, integration report, then the coverage gate                    |
| `deps:audit`       | `go tool govulncheck ./...`, which reads vuln.go.dev and needs the network |

## Coverage gate

`test:coverage` runs `scripts/coverage-gate.sh`. The gate:

- runs every test with the `integration` tag, so unit and integration tests both count;
- uses `-coverpkg=./internal/...,./cmd/...`, so a test in one package counts the statements it runs in another;
- computes statement coverage over the merged profile, and fails below 99%;
- leaves out the e2e tests, which run the built binary as a process.

The gate excludes two things, with the reason for each:

- `internal/adapters/graphql/generated.go`, `models_gen.go`, and `schema.resolvers.go`: gqlgen output with a
  `Code generated` header. gqlgen regenerates them, and the graphql integration tests run them.
- `tests/support/`: test doubles, stub servers, and stub commands. They are test code, and the suites that use them are
  the measurement.

A package with no statements, such as the interface-only `internal/core/ports`, adds nothing to the total.
`internal/adapters/in/mcp/generated.go` has no generated header, so the gate counts it.

### Last measured result

Measured with `./scripts/coverage-gate.sh` on the tree after commit `56589b4`: 4839 of 5186 statements covered, 93.31%.
Later commits changed only tests, comments, formatting, and two README indexes.
The gate is below its 99% minimum. The largest uncovered counts are:

| Uncovered statements | File                                           |
| -------------------: | ---------------------------------------------- |
|                   54 | `internal/adapters/in/mcp/generated.go`        |
|                   23 | `internal/adapters/outline/decode_specs.go`    |
|                   21 | `internal/adapters/outline/decode_2b_specs.go` |
|                   17 | `internal/adapters/kiwi/reader.go`             |
|                   17 | `internal/adapters/in/mcp/tools.go`            |

## Binding check

`bdd:check` reads every feature file under `specs/behaviours` and fails for:

- a scenario with no `@unit` tag;
- a scenario with neither an `@integration` nor an `@e2e` tag, and no matching exempt tag with a justification;
- a scenario that carries both exempt tags.

An exempt tag needs a justification comment directly above the tag line, in this form:

```gherkin
# Exemption(e2e): <why this level cannot prove it>; alternative-proof: <where it is proved>
@e2e-exempt
```

The same check fails when `behaviour-coverage.json` differs from the feature files.

Last measured result: 135 scenarios have no `@unit` tag, and 19 graphql and mcp scenarios have neither an
`@integration` tag nor an exemption. No scenario carries an exempt tag yet.

## Pull-request gate

`./rhino gate run --surface pull-request` does not run any target on this page. It runs the repository's own checks,
such as format, README index, emoji, and public-safety checks.
