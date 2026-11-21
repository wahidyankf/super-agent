# kiwi

The logic of the Kiwi stub command. `Run` parses the flags, reads the synthetic key from the `KIWISTUB_API_KEY`
variable, writes the certificate to the `-ca-file` path, prints `kiwi stub listening on <url>`, and serves until the
stop signal arrives. `-dataset-file` and `-fault-status` shape the stub.

`Host` holds the seams: `Getenv`, `Now`, `ReadFile`, `WriteFile`, `Listen`, `Random`, and `Notify`. `Real` binds them to
the process. The unit bindings pass fakes, so they open no socket and read no entropy. The integration bindings run
`Real` on a loopback socket.
