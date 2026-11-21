# integration

The integration bindings of the retriever-testkit library. Each file here has the `integration` build tag and crosses a
boundary of the machine only through loopback: a TLS server on the loopback address, a real connection that a client
opens to it, or a file that a test writes under its own temporary directory. No test here needs a real secret, and none
reaches a host outside the machine.

Each `*_test.go` file covers the boundary scenarios of one feature file of
[the feature corpus](../../../../specs/libs/retriever-testkit/behaviours/), and it takes the feature's name. Shared
helpers for running a command in process live in `commands_helpers_test.go`.
