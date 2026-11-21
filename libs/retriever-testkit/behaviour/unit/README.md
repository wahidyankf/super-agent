# unit

The hermetic unit bindings of the retriever-testkit library. Each test is an external test of the library's exported
API. It uses the library's own in-memory seams: an in-memory file tree, in-process round trippers of the stubs, a fixed
clock, and a fixed entropy source for the stub certificates. No test here reads a file, an environment variable, the
real clock, a socket, or a process.

Each `*_test.go` file covers one feature file of
[the feature corpus](../../../../specs/libs/retriever-testkit/behaviours/), and it takes the feature's name. The command
features run the logic of the command packages under [stubcmd/](../../stubcmd/README.md) with fake hosts. A fake host
replaces the files, the clock, the TLS server, and the stop signal, so a command test opens no socket and runs no TLS.
