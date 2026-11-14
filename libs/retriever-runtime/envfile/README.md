# envfile

Supplies source keys and account e-mails. A value comes from the process environment first, and then from the local env
file. An empty value counts as missing. A value is never printed.

- `dotenv.go` parses the `KEY=VALUE` format: comments, an optional `export` prefix, quotes, and the last value of a
  repeated key.
- `credentials.go` looks up a key by its variable name. The composition root passes the file system of the env file's
  folder.
- `new_test.go`, `dotenv_test.go`, `outline_ref_test.go`, and `faults_test.go` run without a build tag.
- `credentials_test.go`, `identity_test.go`, and `helpers_test.go` carry the `integration` build tag. So does
  `openfile_integration_test.go`, which opens a real file.
