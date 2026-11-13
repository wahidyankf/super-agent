# internal/core/sources

The use cases. Each one resolves a source, checks its access level, and calls its gateway through a port.

- `administer.go` runs the administrative operations. A critical operation needs a confirmation, and a dry run shows the
  plan first.
- `export.go` exports one document through the export sink, and builds the file name from the title.
- `list.go` lists the configured sources.
- `read.go` runs a read operation, with its paging and field checks.
- `slug.go` turns a title into a file-name stem.
- `sources.go` holds the service that the commands use, and the rules that choose a source.
- `write.go` runs a write operation. A dry run returns the plan without a call.
- `operations/` holds the operation catalog that these use cases read. See [operations](./operations/README.md).
- `*_test.go` files hold the unit tests of the use cases.
