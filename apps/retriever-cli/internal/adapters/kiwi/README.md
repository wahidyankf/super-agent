# internal/adapters/kiwi

The Kiwi TCMS source, a test-management server. It speaks JSON-RPC 2.0 over HTTPS. It reads products, documents, test
cases, and plans, and it writes only through the three methods on its closed write list.

Source files:

- `kiwi.go` holds the package entry point and the source type.
- `client.go` makes the JSON-RPC calls over HTTPS.
- `methods.go` maps each operation to its JSON-RPC method.
- `reader.go` implements the read operations.
- `writer.go` implements the write operations.
- `lookups.go` resolves names to IDs.
- `decode.go` decodes the replies.
- `markdown.go` renders a test case or a plan as Markdown.
- `ids.go` validates the IDs.
- `errors.go` maps a server error to a source error.
- `seam.go` reads the test-only base URL seam, which accepts a loopback host only.

Test files:

- `transport_integration_test.go` runs the source against a stub server. It carries the `integration` build tag.
- `contract_test.go` tests the reply contract of the source.
- `decode_test.go`, `markdown_test.go`, and `ids_test.go` test the decoding, the rendering, and the IDs.
- `methods_test.go`, `reader_test.go`, and `writer_test.go` test the operations.
- `request_guard_test.go` and `safety_test.go` test the guards that refuse an unsafe request.
- `errors_test.go` tests the error mapping.
- `seam_test.go` tests the test-only seam. It carries the `integration` build tag.
- `helpers_test.go` holds the shared fixtures.
