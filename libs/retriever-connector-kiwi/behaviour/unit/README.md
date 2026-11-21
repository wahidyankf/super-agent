# unit

The hermetic unit bindings of the retriever-connector-kiwi library. Each test is an external test of the library's exported
API, in the package `unit_test`. The doubles are in-memory: a scripted round tripper, the testkit's in-process Kiwi stub,
a fake clock, and fake credential ports. The test-only seam of the library is reached through its exported `TestSeam`
names, and the seam reads its environment and its files through injected readers. No test here opens a socket, reads a real
file, or reads a real environment variable.

Each `*_test.go` file covers one feature file of
[the feature corpus](../../../../specs/libs/retriever-connector-kiwi/behaviours/), and it takes the feature's name. The
identifiers file also binds one writes scenario, because both features name the same refusal. Shared doubles and builders
live in `support_test.go`.
