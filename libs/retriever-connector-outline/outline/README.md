# outline

The Outline wiki source. It reads collections, documents, and comments, and it changes the wiki through the same
client. The operations are described in the operation catalog, and this package makes the calls and decodes the
replies.

Production files:

- `outline.go` holds the package entry point and the source type.
- `client.go` makes the HTTPS requests with the source's key.
- `endpoints.go` lists the endpoint of each read operation.
- `endpoints_b.go`, `endpoints_c.go`, and `endpoints_2b.go` list the endpoints of the later groups. The suffix names the
  group.
- `reader.go` implements the read operations.
- `writer.go` implements the write operations.
- `decode.go` decodes the replies. `decode_specs.go`, `decode_c_specs.go`, and `decode_2b_specs.go` hold the decoding
  specs of the operations, by group.
- `plaintext.go` converts a document body to plain text.
- `seam.go` holds the test-only base URL seam, which accepts a loopback host only. Its entry point reads the CA file
  with `os.ReadFile`, and the composition root calls it for tests only.

Tests sit beside the code. Each `*_integration_test.go` file carries the `integration` build tag. `contract_test.go`
runs the [source contract](../../retriever-testkit/sourcecontract/README.md) suite.
