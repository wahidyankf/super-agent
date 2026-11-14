# engine

The engine package. It runs the read operations of a validated connector spec against one source. The package reads
no file, no environment variable, and no process. The key, the identity, the clock, and the network reach it through
the `Ports` that the caller passes.

## Files

- [engine.go](./engine.go) holds `New`, `Ports`, `Network`, `Clock`, `AllowPrivateNetwork`, and `Read`. It also holds
  the gateway reads `ListCollections`, `CollectionTree`, and `Document`, which run the spec's role operations. `New`
  checks the source kind, the host suffix, the ports, and the model rules before it builds an engine.
- [render.go](./render.go) builds each request: the path slots, the encoded query, the headers, and the JSON body. It
  also holds `ExampleTarget`, which the replay stub uses to key each canned reply.
- [send.go](./send.go) sends one request. It sets the auth header, runs the request under the time limit, reads the
  reply up to the size cap, and classifies the status. It retries a rate limit and maps each failure to its error.
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
  real dialer.

## Rules that the code follows

- A value that fills a path slot is one whole segment. It is escaped, and a value that would leave its segment is
  refused.
- A header value that fills a slot is refused when it holds a control character or is not UTF-8.
- A spec may lower a limit, and it may never raise one. The built-in values are 30 seconds, 10 MiB, 20 pages, and 3
  retries.
- A read never follows a redirect. A 3xx reply fails with `redirected`.
- A failed read returns its error, and it never returns a partial record.
