# internal/adapters/jira

The Jira Cloud source. It reads projects, issues found by JQL, and the comments of an issue. It converts Atlassian
Document Format to Markdown. It never writes: the adapter has no write path.

Source files:

- `jira.go` holds the package entry point and the source type.
- `client.go` makes the HTTPS requests.
- `auth.go` builds the basic-auth header from the account e-mail and the token. It never prints either.
- `gateway.go` implements the source gateway.
- `methods.go` maps each operation to its request.
- `reader.go` implements the read operations.
- `issue.go` and `issues_test.go` cover the issue reads.
- `decode.go` decodes the JSON replies.
- `adf.go` converts Atlassian Document Format to Markdown.
- `markdown.go` renders the Markdown.
- `sanitize.go` removes the text that must not reach a caller.
- `ids.go` validates the issue and project keys.
- `errors.go` maps an HTTP status to a source error.
- `seam.go` reads the test-only base URL seam, which accepts a loopback host only.

Test files:

- `integration_test.go` runs the source against a stub server. It carries the `integration` build tag.
- `adf_test.go`, `adf_edge_test.go`, and `adf_fuzz_test.go` test the conversion, its edge cases, and a fuzz target.
- `markdown_test.go` and `sanitize_test.go` test the rendering and the sanitising.
- `auth_test.go`, `client_test.go`, and `errors_test.go` test the requests and the errors.
- `gateway_test.go`, `methods_test.go`, `reader_test.go`, and `issues_test.go` test the gateway and the reads.
- `ids_test.go` tests the key validation.
- `seam_test.go` tests the test-only seam.
- `jira_test.go` and `helpers_test.go` test the package entry point and hold the shared fixtures.
