# engine

The engine package. It runs the read and write operations of a validated connector spec against one source. The package
reads no file, no environment variable, and no process. The key, the identity, the clock, and the network reach it
through the `Ports` that the caller passes.

## Files

- [engine.go](./engine.go) holds `New`, `Ports`, `Network`, `Clock`, `AllowPrivateNetwork`, and `Read`. It also holds
  the gateway reads `ListCollections`, `CollectionTree`, and `Document`, which run the spec's role operations. `New`
  checks the source kind, the host suffix, the ports, and the model rules before it builds an engine.
- [write.go](./write.go) holds `Write`, the port of the change operations. It binds the arguments, resolves each
  declared name, sends the one write, and maps the reply. It also holds the name lookup and the outcome rules.
- [render.go](./render.go) builds each request: the path slots, the encoded query, the headers, and the JSON body. It
  also holds `ExampleTarget`, which the replay stub uses to key each canned reply.
- [send.go](./send.go) sends one request. It sets the auth header, runs the request under the time limit, reads the
  reply up to the size cap, and classifies the status. It retries a rate limit and maps each failure to its error. For a
  write it also decides whether a failure is an unknown outcome.
- [guard.go](./guard.go) holds the transport and the dial check. The check refuses an address that the engine must not
  reach, and it dials only the address that it has checked.
- [reply.go](./reply.go) decodes one JSON value under the depth and token caps. It resolves JSON pointers, removes
  control characters from text, and maps the reply to records.
- [paging.go](./paging.go) walks an offset read and a cursor read, up to the page cap.
- [errors.go](./errors.go) holds the sentinel errors, `Code`, and `ExitStatus`.
- [replay/](./replay/README.md) holds the replay stub.
- [testdata/](./testdata/README.md) holds the spec fixture that the tests read.

## Test files

- The tests in `package engine_test` cover behaviour through the public API, with the replay stub and fake ports.
- The tests in `package engine` cover the unexported helpers: the caps, the classifier, the renderer, and the guard.
- `loopback_integration_test.go` has the `integration` build tag. It runs real TLS against a loopback server, with the
  real dialer. It covers the time limit, the redirect refusal, and the dial refusal, with and without the
  `AllowPrivateNetwork` option.
- The untagged tests use fakes for the clock, the credentials, and the identities. They open no socket and start no
  timer, and the replay stub answers every request. The write tests also run the guarded transport over an in-memory
  pipe.
- `write_test.go` and `write_network_test.go` cover the write path. The network file runs the guarded transport over an
  in-memory pipe, so a refused dial, a failed handshake, and a connection cut after the send run without a socket.
- `write_integration_test.go` has the `integration` build tag. It proves on a real connection that a timed out write is
  sent once.

## Rules that the code follows

- A value that fills a path slot is one whole segment. It is escaped, and a value that would leave its segment is
  refused.
- A header value that fills a slot is refused when it holds a control character or is not UTF-8.
- A spec may lower a limit, and it may never raise one. The built-in values are 30 seconds, 10 MiB, 20 pages, and 3
  retries.
- A read never follows a redirect. A 3xx reply fails with `redirected`.
- A failed read returns its error, and it never returns a partial record.
- A write is sent once. Only a 429 reply sends it again, after its wait. A failure after the request may have left is
  `write_outcome_unknown`, and no failure of a write is retried in any other case.
- A name is resolved only by one unpaged read with exactly one match. A paged read is refused, because a second match
  past its last page could be hidden.

## Partly met

- S5: the ADR asks for a check of the connected address in the dialer's control hook. That hook needs syscall types,
  which the module layout forbids in the engine's production code. The engine dials the checked IP literal instead, so
  the connected address is the checked address. The result is the same, and the mechanism differs.
- R1: the engine writes no log, because the logger is not part of this pull request. There is no log text to redact
  yet.

## Not in this module yet

- The Markdown converters, `markdown-from-html` and `markdown-from-adf`, belong to later pull requests of ADR 001. An
  operation that needs one returns `unsupported_operation` before any request is sent.
- Dry runs, previews, and confirmation flags are not part of the engine, because `v0.2.0` removed them on purpose. A
  write simply runs.
- No production code wires a real clock or a real dialer. The composition root supplies them through `Ports`.
- A cursor read that reaches its page cap returns the pages it has, with no continuation.
