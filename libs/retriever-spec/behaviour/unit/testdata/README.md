# testdata

- [example-wiki.yml](./example-wiki.yml) is a copy of the bundled connector spec in
  [testdata](../../../testdata/README.md) of the module. The tests embed it with `go:embed`, because a behaviour test
  is a separate package and cannot reach the module's own testdata directory. The copy holds the same bytes, and it uses
  only the `example.test` host suffix and the `EXAMPLE_WIKI_*` variable names.
