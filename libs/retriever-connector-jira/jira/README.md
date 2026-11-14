# jira

The Jira Cloud source. It reads projects, issues found by JQL, and the comments of an issue. It converts Atlassian
Document Format to Markdown. It never writes: the adapter has no write path.

Production files:

- `jira.go` holds the package entry point and the source type.
- `client.go` makes the HTTPS requests.
- `auth.go` builds the basic-auth header from the account e-mail and the token. It never prints either.
- `gateway.go` implements the source gateway.
- `methods.go` maps each operation to its request.
- `reader.go` implements the read operations, and `issue.go` holds the issue reads.
- `decode.go` decodes the JSON replies.
- `adf.go` converts Atlassian Document Format to Markdown.
- `markdown.go` renders the Markdown.
- `sanitize.go` removes the text that must not reach a caller.
- `ids.go` validates the issue and project keys.
- `errors.go` maps an HTTP status to a source error.
- `seam.go` holds the test-only base URL seam, which accepts a loopback host only. Its entry point reads the CA file
  with `os.ReadFile`, and the composition root calls it for tests only.

Tests sit beside the code. `integration_test.go` runs the source against a stub server. It and
`auth_scan_integration_test.go` carry the `integration` build tag.
