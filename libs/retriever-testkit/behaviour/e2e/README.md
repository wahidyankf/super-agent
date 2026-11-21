# e2e

The end-to-end bindings of the retriever-testkit library. Each file here has the `e2e` build tag. A test builds one of
the test-only stub commands with `go build` into a temporary directory, runs it as a process, and checks its exit status,
its standard output and error, and the files and TLS connections it makes on the loopback address.

Each `*_test.go` file covers one command feature of
[the feature corpus](../../../../specs/libs/retriever-testkit/behaviours/), and it takes the feature's name. Shared
process helpers live in `commands_helpers_test.go`.
