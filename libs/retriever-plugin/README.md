# retriever-plugin

The pinned plugin host of ADR 001. It starts one out-of-process connector plugin per call and speaks JSON-RPC 2.0
with it over the plugin's standard input and output. A plugin is the only path on which code runs (boundary B4), so
the host holds it tightly: the pin, the scrubbed environment, the time limit, the frame cap, and the redaction.

- Module path: `github.com/wahidyankf/super-agent/libs/retriever-plugin`.
- Nx project `retriever-plugin`, with the tags `type:plugin`, `lang:go`, and `domain:retriever`.

## Packages

- The root package `plugin` holds the host: `plugin.go` (the call, the pin check, the framing, the redaction),
  `gateway.go` (the source ports over the call), `plugin_test.go` and `gateway_test.go` (the hermetic unit tests),
  and `plugin_integration_test.go` (the `integration` tests over real plugins, including one source served through
  the gateway).

## Public surface

- `Call(ctx, cfg, method, params, seams)` runs one JSON-RPC call against the pinned plugin: it copies the file into
  a fresh private folder, hashes the copy, runs the copy only when the hash equals the pin, sends one request with
  id 1, and reads one reply under the time limit. The folder is removed on every path.
- `Gateway` serves one plugin source through `Call`: it implements `SourceGateway`, `SourceReader`, and
  `SourceWriter`, so the runtime sees one shape for engine and plugin kinds.
- `PinMismatchError` (`retriever.connector.plugin_pin_mismatch`), `ProtocolError`
  (`retriever.connector.plugin_protocol`), `StartError` (`retriever.connector.plugin_start_failed`), and
  `TimeoutError` (which unwraps to `domain.ErrTimedOut`, so the CLI exits `124`).

## Tests

- `plugin_test.go` drives every path with fakes: no real file, process, or wait is touched.
- `plugin_integration_test.go` runs real plugins: an env-printing plugin proves the scrubbed environment, and a
  silent plugin proves the kill and the timeout. It carries the `integration` tag.
- `gateway_test.go` maps every reply shape with fakes; `gateway_integration_test.go` serves a source through a real
  plugin.
- `behaviour/` binds every scenario of [the plugin specs](../../specs/libs/retriever-plugin/behaviours/): `unit/` with
  injected seams, and `integration/` with real child processes and files. See [behaviour](./behaviour/README.md).
