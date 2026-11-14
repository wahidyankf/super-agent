# sources

The use cases. Each one resolves a source and calls its gateway through a port. The package imports no package that
reaches outside the process.

- `export.go` exports one document through the export sink, and names the file by the document ID.
- `list.go` lists the configured sources.
- `read.go` runs a read operation, with its paging and field checks.
- `sources.go` holds the service that the commands use, and the rules that choose a source.
- `write.go` runs a write operation, the administrative ones included, and calls the writer once.
- `*_test.go` files hold the unit tests of the use cases. They carry no build tag.

The operation catalog that these use cases read is in the contract, at
[operations](../../retriever-contract/operations/README.md).
