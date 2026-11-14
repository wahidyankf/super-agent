# kiwi

The Kiwi TCMS source, a test-management server. It speaks JSON-RPC 2.0 over HTTPS. It reads products, documents, test
cases, and plans, and it writes only through the three methods on its closed write list.

Production files:

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
- `seam.go` holds the test-only base URL seam, which accepts a loopback host only. Its entry point reads the CA file
  with `os.ReadFile`, and the composition root calls it for tests only.

Tests sit beside the code. `transport_integration_test.go` and `seam_test.go` carry the `integration` build tag.
`transport_integration_test.go` runs the source against a stub server. `contract_test.go` checks the reply contract with
the [source contract](../../retriever-testkit/sourcecontract/README.md) suite.
