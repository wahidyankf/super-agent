# scripts

The build and coverage files of the app. The shared coverage gate is not here: it is
[go-coverage-gate.sh](../../../scripts/go-coverage-gate.sh), at the repository root, and the app's `test:unit` and
`test:coverage` targets run it from the module root.

- [build-release.sh](./build-release.sh) builds the four platform archives and `SHA256SUMS` for one annotated tag. It
  backs the `build:release` target.
- [build-release_test.sh](./build-release_test.sh) tests the release build. The `test:unit` target runs it.
- [coverage-exclusions.tsv](./coverage-exclusions.tsv) lists the excluded generated files of the app, each with its
  reason. The gate reads it.
