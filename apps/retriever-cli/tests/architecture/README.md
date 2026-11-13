# Architecture tests

Checks the dependency direction of the functional core with the Go parser, not with a build. The core packages may
import only other core packages and effect-free libraries. They never import the adapters, the composition root, or
the tests.

Run them with `go test ./tests/architecture/` from the module root.
