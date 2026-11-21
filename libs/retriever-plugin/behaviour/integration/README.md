# integration

The integration bindings of the plugin's behaviour scenarios. Each test file has the `//go:build integration` tag and
the package `integration_test`. Run them with `go test -tags integration ./behaviour/integration/`.

The tests run real processes and real files. `TestMain` builds the fixture plugin from
[testdata/plugin-helper.go](../../testdata/plugin-helper.go) into a temporary folder and removes it afterwards. Other
tests write shell-script plugins into a temporary folder, and they wait on process IDs and marker files rather than
on fixed sleeps.

- [support_test.go](./support_test.go): the fixture build, the real-disk `plugin.Files`, the credential double, the
  source and gateway builders, and the process-liveness checks.
- [pinning-and-start_test.go](./pinning-and-start_test.go): the pin check on real bytes, the real copy, the start
  failures that real files reach, and the kill of a real process group.
- [environment-and-secrets_test.go](./environment-and-secrets_test.go): the scrubbed environment of a real child, and
  the failures that start no child.
- [exchange-and-timeouts_test.go](./exchange-and-timeouts_test.go): real replies, real frame and error-output limits,
  the real time limit, and cancellation.
- [gateway-mapping_test.go](./gateway-mapping_test.go): the request a real child receives, and the failures a real
  child causes through each gateway method.
- [real-process_test.go](./real-process_test.go): the rules that only a real child can show.

The scenarios whose behaviour is a pure in-memory rule (the mapping of a reply shape, the redaction text, and the
default time limit) carry an `@integration-exempt` tag with their reason. Their proof is in
[unit/](../unit/README.md).
