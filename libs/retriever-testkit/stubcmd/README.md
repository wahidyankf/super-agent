# stubcmd

The importable logic of the test-only stub commands. Each command keeps its flags, its messages, and its wiring in one
package here. The package takes its seams through a `Host` value: files, the clock, the TLS server, and the stop signal.
`Real` binds every seam to the operating system, and `Run` returns the exit status of the command.

The `main` package of each command only calls `Run` with the real host and exits with its status. Those packages are
[kiwistubcmd/](../kiwistubcmd/README.md), [jirastubcmd/](../jirastubcmd/README.md), and
[outlinestubcmd/](../outlinestubcmd/README.md).

- [kiwi/](kiwi/README.md): the Kiwi stub command.
- [jira/](jira/README.md): the Jira stub command.
- [outline/](outline/README.md): the Outline stub command.
