# scripts

The coverage exclusions of this module. [coverage-exclusions.tsv](./coverage-exclusions.tsv) is read by the shared
coverage gate, [go-coverage-gate.sh](../../../scripts/go-coverage-gate.sh), when the `test:unit` or `test:coverage`
target of this module runs. The file holds no entry, because this module has no generated Go file.
