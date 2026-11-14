# testdata

- [example-wiki.yml](./example-wiki.yml) is a bundled connector spec for an example wiki. It holds four operations: a
  cursor-paged list, a read of one document with Markdown, a create, and a delete. The tests embed it with `go:embed` in
  `helpers_test.go`, so no test reads a file at run time.

The spec uses only the `example.test` host suffix and the `EXAMPLE_WIKI_*` variable names. It holds no credential and no
real host, and its example values are invented.
