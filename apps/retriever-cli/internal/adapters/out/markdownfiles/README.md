# internal/adapters/out/markdownfiles

The default export sink. It writes each exported document as one Markdown file in the folder that `--out` names. The
folder is created when it is missing. A write goes through a temporary file and a rename, so an existing file is
replaced whole or not at all.

- `markdownfiles.go` holds the sink and its atomic write.
- `rename_failure_test.go` tests the failure of the rename. It is the only test file of the package.
