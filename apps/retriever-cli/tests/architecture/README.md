# Architecture tests

These tests read the Go source with the parser, not with a build. They check two things across every module of the
repository: which module may import which, and that the hermetic rule holds for untagged tests.

## Import rules

`imports_rules_test.go` and `imports_test.go` hold the layering table. The table names each module's kind, and the
checks read the kind from the `type:` tag of its Nx project. A contract imports no other module. A runtime, a connector,
or the testkit imports only the contract in production code, and a test of those modules may also import the testkit.
The app may import the contract, the runtime, and the connectors, but production code may not import the testkit, and
no module imports the app. The full table is in [library layering](../../../../docs/explanation/library-layering.md).

`imports_integration_test.go` walks the real files and applies the table. It also checks three more things:

- each module's `go.mod` starts with its own module path, and each Nx project carries its `type:` tag;
- each module's Nx `implicitDependencies` names every other module that its Go files import; and
- the production code of the contract and of the runtime's `sources` package imports no package that reaches outside
  the process, such as `os` or `net/http`.

## Hermetic guard

`hermetic_test.go` holds the rule that keeps an untagged test off the real disk, the environment, the working
directory, and child processes. It is pure: it reads the source it is given, so its negative tests run untagged.
`hermetic_integration_test.go` walks the app and every library, and it applies the rule to each untagged test file.
The rule has no exemption. See [the test gates](../../../../docs/reference/retriever-cli-test-gates.md).

## Running the tests

The checks that read the source tree from disk carry the `integration` build tag. Run them from the app module root
with `go test -tags integration ./tests/architecture/`. The rule tests in `imports_rules_test.go` and `hermetic_test.go`
run without a tag.
