# internal/adapters/envfile

Supplies source keys and account e-mails. A value comes from the process environment first, and then from the local env
file. An empty value counts as missing. A value is never printed.

- `dotenv.go` parses the `KEY=VALUE` format: comments, an optional `export` prefix, quotes, and the last value of a
  repeated key.
- `credentials.go` looks up a key by its variable name.
- `new_test.go` tests the constructor of the reader.
- `dotenv_test.go` tests the parser.
- `outline_ref_test.go` tests the reference to an Outline key.
- `credentials_test.go`, `identity_test.go`, and `helpers_test.go` test the key and identity lookups and hold their
  fixtures. They carry the `integration` build tag, so run them with `-tags integration`.
