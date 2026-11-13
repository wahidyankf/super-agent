# Retriever CLI test gates

This page records what each test target of `apps/retriever-cli` runs, what the coverage gate measures and excludes,
what the binding check enforces, and which targets CI runs on each platform. It states facts. The rules that govern them
live in the repository's standards.

## Targets

| Target             | Runs                                                                                  |
| ------------------ | ------------------------------------------------------------------------------------- |
| `lint`             | `go vet` with the integration and e2e tags, then a `gofmt` check                      |
| `typecheck`        | `go build ./...`, then compiles every test package with those tags, without running   |
| `test:unit`        | a check that no `_test.go` file calls `.Skip(`, the gate self-test, then the gate     |
| `test:quick`       | `lint`, `typecheck`, then `test:unit`                                                 |
| `test:integration` | `go test -count=1 -tags integration ./...`                                            |
| `test:e2e`         | `go test -count=1 -tags e2e ./tests/behaviour/...`; the suite builds the binary first |
| `test:conformance` | `go test -count=1 ./tests/conformance/... ./tests/architecture/...`, with no tags     |
| `bdd:check`        | the binding check of `specs/behaviours`, described below                              |
| `test:coverage`    | `scripts/coverage-gate.sh`, the same gate as `test:unit` without the skip check       |
| `build`            | `go build -trimpath -o dist/retriever ./cmd/retriever`                                |
| `deps:audit`       | `go tool govulncheck ./...`, which reads vuln.go.dev and needs the network            |

## Coverage gate

`test:unit` and `test:coverage` run `scripts/coverage-gate.sh`. `test:unit` runs the gate's self-test,
`scripts/coverage-gate_test.sh`, first. The gate measures unit-level statement coverage of the whole module:

- It makes one instrumented run of the untagged tests, with `-race`.
- It measures every production package under `internal/` and `cmd/`. A test in one package counts the statements it runs
  in another.
- The Gherkin unit binding, `TestUnitBehaviours`, is untagged, so every `@unit` scenario counts.
- The integration tests (tag `integration`) and the end-to-end tests (tag `e2e`) are outside the number. The
  integration and e2e targets run them.
- It computes statement coverage over the merged profile. The floor is 99 percent, and the `COVERAGE_MINIMUM`
  environment variable can change it.

It fails when a test fails, when no statement is measured, and when a production package has statements but no test
file, because nothing would measure that package. Its `--check-only` option validates the exclusions and the package
list without running any test.

### Exclusions

The exclusions are data in `scripts/coverage-exclusions.tsv`. Each line holds a kind, a path, and a reason, separated by
tabs. Two kinds are allowed:

- `generated`: a Go file whose `Code generated ... DO NOT EDIT.` line comes before its package clause. Two entries
  qualify, `internal/adapters/graphql/generated.go` and `models_gen.go`, which gqlgen writes. The graphql unit and
  integration tests run them.
- `test-support`: the `tests/support` directory of test doubles, stub servers, and stub commands. The behaviour suites
  use them, and they are test code.

The gate fails on any other kind, on an entry with no reason, on a path pattern, on a path that does not exist, on a
`generated` file without the header before its package clause, and on a production package with no test file.
`scripts/coverage-gate_test.sh` runs each of these cases against fixture modules.

`internal/adapters/graphql/schema.resolvers.go` is not excluded. Its gqlgen line comes after the package clause, so under
the Go convention it is not a generated-code file, and the gate counts its statements. A file without the header is
counted even when its name says generated. The MCP tool table, `internal/adapters/in/mcp/operation_tools.go`, was named
`generated.go` before. It is hand-written, and the gate counts it.

A package with no statements, such as the interface-only `internal/core/ports`, adds nothing to the total.

### What unit tests may do

Unit tests run in process, beside their package. They may touch the file system only through `t.TempDir`, and they may
open a loopback `httptest` server. The registry tests have no build tag, so they run at the unit level.

An untagged unit test does not change the process environment. The composition root takes the environment as a function:
`run` in `cmd/retriever` and `SecretLookup` in `internal/adapters/in/cli` receive it as a parameter, so a unit test supplies
its own values. The architecture test `TestUntaggedTestsNeverChangeTheEnvironment` fails when an untagged test calls
`t.Setenv`, `os.Setenv`, `os.Unsetenv`, or `os.Clearenv`. It exempts one file, `tests/behaviour/world_test.go`, the
in-process behaviour harness. That harness serves the unit and integration levels from one file and sets the environment
for both, so the exemption stays until the integration-only part moves to a tagged file. The check reads the direct calls
of each file. It does not check writes outside `t.TempDir`.

This page records no coverage figure. A figure comes from a run of `npx nx run retriever-cli:test:coverage` on a given
tree, and it belongs with the change that produced it.

## Binding check

`bdd:check` runs two tests of `tests/conformance` with the `bdd` build tag. They read every feature file under
`specs/behaviours`:

- Every `Scenario` and `Scenario Outline` needs an `@unit` tag. No exemption exists for it.
- Every scenario needs an `@integration` tag or an `@e2e` tag, or the matching exempt tag, `@integration-exempt` or
  `@e2e-exempt`.
- An exempt tag needs a justification: a comment line in the lines directly above the scenario's tags, in this form:

  ```gherkin
  # Exemption(e2e): <why this level cannot prove it>; alternative-proof: <where it is proved>
  @e2e-exempt
  ```

- A scenario that carries both `@integration` and `@integration-exempt`, or both `@e2e` and `@e2e-exempt`, fails. A
  scenario with both exempt tags is a detail, and it belongs in a plain Go test.
- `behaviour-coverage.json` must match the feature files. It records, for each feature, the scenario count and the count
  of `@unit`, `@integration`, and `@e2e` tags, with the totals. A difference fails the check.

## CI runs

The Go job runs only when an affected Nx project carries the `lang:go` tag. Within that job:

- `ubuntu-24.04` runs `lint`, `typecheck`, `test:unit`, `test:integration`, `test:e2e`, `test:conformance`,
  `bdd:check`, and `deps:audit`.
- `macos-15` runs `test:unit`, `test:e2e`, and `test:conformance`.

The macOS targets cover the behaviour that depends on the platform: signals, exit statuses 130, 141, and 143, the user
configuration directory, atomic renames, file modes, symbolic links, and port binding.

A cross-compile check, on `ubuntu-24.04`, builds `./cmd/retriever` for `darwin/arm64`, `darwin/amd64`, `linux/amd64`,
and `linux/arm64`. It builds only. Nothing is uploaded or published.

The Go toolchain is pinned once, by the `toolchain` line of `apps/retriever-cli/go.mod` (`go1.27.4`). The `go` line sets
the minimum language version. The Go job fails when the installed toolchain differs from the `toolchain` line.

Windows is not supported. See the [exit codes](./retriever-cli-exit-codes.md#supported-platforms).

## Pull-request gate

`./rhino gate run --surface pull-request` does not run any target on this page. It runs the repository's own checks,
such as format, README index, emoji, and public-safety checks.
