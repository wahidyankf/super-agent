# internal/adapters/catalog

Loads the sources catalog, the YAML file that the user names with `--catalog`, `RETRIEVER_CATALOG`, or a file in the
working directory. The loader refuses an unknown field, a repeated field, a second document, a file over 64 KiB, and any
rule of the catalog format. The rules are in
[the catalog format](../../../../../docs/reference/retriever-cli-catalog-format.md).

- `catalog.go` reads the file, validates each source, and returns the catalog. Its error text never includes
  the file path.
- `catalog_test.go` tests the loader. It carries the `integration` build tag, so run it with `-tags integration`.
