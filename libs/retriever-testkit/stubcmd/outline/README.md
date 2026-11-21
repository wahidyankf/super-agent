# outline

The logic of the Outline stub command. `Run` parses the flags, starts the Outline stub over TLS, writes its certificate
to the `-ca-file` path, prints `outline stub listening on <url>`, and serves until the stop signal arrives.
`-collection` repeats in listing order, `-list-status` and `-document-status` set a fault, and `-documents-file`
supplies the documents. Two faults at once are refused.

`Host` holds the seams: `ReadFile`, `WriteFile`, `Now`, `Serve`, and `Notify`. `Serve` starts the TLS server and returns
a `Started` value with its URL, its DER certificate, and its stop function. `Real` binds every seam to the process. The
unit bindings pass a fake `Serve`, so they open no socket.
