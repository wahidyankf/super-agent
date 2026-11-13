# Architecture tests

Checks the dependency direction of the functional core with the Go parser, not with a build. The core packages may
import only other core packages and effect-free libraries. They never import the adapters, the composition root, or
the tests.

`hermetic_test.go` holds the rule that keeps untagged tests off the real disk, the environment, the working directory, and
child processes. It is pure: it reads the source it is given, and its negative tests run untagged.
`hermetic_integration_test.go` walks the module and applies the rule to every untagged test file. The rule has no
exemption. See [the test gates](../../../../docs/reference/retriever-cli-test-gates.md).

The checks that read the source tree from disk carry the `integration` build tag. Run the package with
`go test -tags integration ./tests/architecture/` from the module root.
