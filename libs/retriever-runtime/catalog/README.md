# catalog

Loads the sources catalog, the YAML file that the user names with `--catalog`, `RETRIEVER_CATALOG`, or a file in the
working directory. The loader refuses an unknown field, a repeated field, a second document, a file over 64 KiB, and any
rule of the catalog format. The rules are in
[the catalog format](../../../docs/reference/retriever-cli-catalog-format.md).

- `catalog.go` reads the file through the file system that it is given, validates each source, and returns the catalog.
  Its error text never includes the file path.
- `catalog_integration_test.go` reads a real file, so it carries the `integration` build tag. The other tests run
  without a tag.
