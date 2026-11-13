# Examples

This folder holds the example sources catalog, [sources.example.yml](sources.example.yml). Every host and every name in it
is an example, and the example variables are named `EXAMPLE_*`. The file holds no key. Each source names the environment
variable that holds its key, and the key comes from the process environment or the env file.

To use it, copy the file to `sources.yml`, or pass its path to `--catalog`. A real catalog names real hosts, so it stays out
of the repository. The catalog fields and rules are in the
[catalog format reference](../../../docs/reference/retriever-cli-catalog-format.md).
