# replay

The replay stub is an HTTP round tripper that answers from the read examples of a connector spec. A test gives it to the
engine in place of the network, so the engine runs a read without a socket or a timer.

- `New(spec)` reads each read example. The request of an example is bound with the operation catalog's rules, so an
  example that the catalog would refuse makes `New` fail. A write example is never served.
- `Script(method, target, replies...)` sets the replies for one method and target. The replies are sent in order, and
  the last one repeats. A later `Script` for the same key replaces the earlier one.
- `Fail(method, target, err)` makes a request to that key return the error.
- `RoundTrip` records each request, drains and closes its body, and then answers. A script takes precedence over an
  example. A request that matches neither is recorded, counted, and fails with `ErrNoExchange`.
- `Requests()` returns every request the stub received, with its method, path, raw query, and headers. `Unmatched()`
  returns how many requests matched nothing.

The stub reads no file and opens no socket. Its tests are in `stub_test.go`, which runs the stub through the public
API, and in `stub_internal_test.go` and `stub_examples_test.go`, which cover its helpers and its example rules.
