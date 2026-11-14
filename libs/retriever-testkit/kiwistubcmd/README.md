# Kiwi stub command

A test-only command that serves the Kiwi stub over TLS on a loopback port, as a process. The Bearer key that the stub
accepts comes from the `KIWISTUB_API_KEY` variable, which the test sets to a synthetic value. The end-to-end layer
points the real binary at it through the Kiwi seam. It is not part of the retriever binary.
