# configpaths

Decides where the sources catalog and the local env file are read from. The order is the flag, then the environment
variable, then the file in the working directory, then the user configuration folder. With a profiles folder, the run
takes one profile from `--profile` or `RETRIEVER_PROFILE` and reads only its folder. The rules are in
[configuration](../../../docs/reference/retriever-cli-configuration.md).

- `configpaths.go` resolves the two paths. It reads no file.
- `configpaths_test.go` tests the order and the relative-path handling. `configpaths_refusal_test.go` tests the refused
  cases. `profiles_test.go` tests the profile selection and the folder isolation.
