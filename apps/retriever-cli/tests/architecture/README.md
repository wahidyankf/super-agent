# Architecture tests

Checks the dependency direction of the functional core with the Go parser, not with a build. The core packages may
import only other core packages and effect-free libraries. They never import the adapters, the composition root, or
the tests.

`environment_test.go` checks that no untagged test changes the process environment, with one named exemption, the
behaviour harness. See [the test gates](../../../../docs/reference/retriever-cli-test-gates.md).

Run them with `go test ./tests/architecture/` from the module root.
