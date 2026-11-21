# Jira stub command

A test-only command that serves the Jira stub over TLS on a loopback port, as a process. Its logic lives in
[stubcmd/jira](../stubcmd/jira/README.md), and this package only passes the real host to that logic and exits with its
status. The `-email` and `-token` flags name the synthetic credentials that the stub accepts. The end-to-end layer
points the real binary at it through the loopback seam. It is not part of the retriever binary.
