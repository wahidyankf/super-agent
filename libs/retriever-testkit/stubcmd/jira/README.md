# jira

The logic of the Jira stub command. `Run` parses the flags, starts the Jira stub over TLS, writes its certificate to the
`-ca-file` path, prints `jira stub listening on <url>`, and serves until the stop signal arrives. The `-email` and
`-token` flags name the synthetic credentials, or `-config` supplies them from a JSON file. `-record` appends one JSON
line for each request. The fault flags shape the replies.

`Host` holds the seams: `ReadFile`, `WriteFile`, `CreateFile`, `Now`, `Serve`, and `Notify`. `Serve` starts the TLS
server and returns a `Started` value with its URL, its DER certificate, and its stop function. `Real` binds every seam
to the process. The unit bindings pass a fake `Serve`, so they open no socket.
