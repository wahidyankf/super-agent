# unit

The hermetic unit bindings of the retriever-engine library. Each test is an external test of the library's exported
API, with in-memory doubles for its seams: a fake clock, a scripted round tripper, and fake credential and identity
ports. No test here reads a file, an environment variable, the real clock, a socket, or a process.

Each `*_test.go` file covers one feature file of
[the feature corpus](../../../../specs/libs/retriever-engine/behaviours/), and it takes the feature's name. Shared
doubles and builders live in `bindings_support_test.go`.
