# Outline stub command

A test-only command that serves the Outline stub over TLS on a loopback port, as a process. Its logic lives in
[stubcmd/outline](../stubcmd/outline/README.md), and this package only passes the real host to that logic and exits with
its status. It prints `outline stub listening on <url>` once it listens, and it writes the certificate to the file that
`-ca-file` names. The end-to-end layer points the real binary at it through the loopback seam. It is not part of the
retriever binary.
