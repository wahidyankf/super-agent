# markdownfiles

The default export sink. It writes each exported document as one Markdown file in the folder that `--out` names. The
folder is created when it is missing. A write goes through a temporary file and a rename, so an existing file is
replaced whole or not at all.

- `markdownfiles.go` holds the sink and its atomic write. It reaches the disk only through the folder calls and the
  temporary-file function that its caller passes in.
- `sink_fake_test.go` holds the fake sink that the untagged tests use.
- `markdownfiles_integration_test.go`, `real_disk_integration_test.go`, `faults_integration_test.go`, and
  `rename_failure_integration_test.go` write real files, so they carry the `integration` build tag.
