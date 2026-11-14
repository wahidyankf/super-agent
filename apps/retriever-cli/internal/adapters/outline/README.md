# internal/adapters/outline

The Outline wiki source. It reads collections, documents, and comments, and it changes the wiki through the same
client. The operations are described in the operation catalog, and this package makes the calls and decodes the
replies.

Source files:

- `outline.go` holds the package entry point and the source type.
- `client.go` makes the HTTPS requests with the source's key.
- `endpoints.go` lists the endpoint of each read operation.
- `endpoints_b.go`, `endpoints_c.go`, and `endpoints_2b.go` list the endpoints of the later groups. The suffix names the
  group.
- `reader.go` implements the read operations.
- `writer.go` implements the write operations.
- `decode.go` decodes the replies.
- `decode_specs.go`, `decode_c_specs.go`, and `decode_2b_specs.go` hold the decoding specs of the operations, by group.
- `plaintext.go` converts a document body to plain text.
- `seam.go` reads the test-only base URL seam, which accepts a loopback host only.

Test files:

- `integration_test.go` and `two_sources_integration_test.go` run the source against a stub server. They carry the
  `integration` build tag.
- `contract_test.go` checks the replies against the contract that the stub servers hold.
- `outline_test.go` tests the package entry point.
- `endpoints_test.go` and `endpoints_2b_test.go` test the endpoint lists.
- `decode_specs_test.go` and `decode_2b_test.go` test the decoding specs.
- `reply_test.go` and `reply_size_test.go` test the replies, including their size.
- `projection_tripwire_test.go` tests the field projection of the replies.
- `plaintext_test.go` tests the plain-text conversion.
- `write_test.go`, `write_envelope_test.go`, and `write_reply_test.go` test the write operations.
- `admin_test.go`, `admin_c2_test.go`, `admin_c3_test.go`, and `admin_c4_test.go` test the administrative operations.
- `creator_keys_test.go` and `credref_test.go` test the key references.
- `safety_test.go` tests the guards that refuse an unsafe request.
- `seam_test.go` tests the test-only seam.
- `helpers_test.go` holds the shared fixtures.
