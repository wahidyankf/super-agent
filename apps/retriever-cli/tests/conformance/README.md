# Conformance runner

Runs every published check in [assertions.json](../../../../specs/fixtures/cli-conformance/assertions.json), in
process, with no source and no network. Each check compares the exit status, the stdout and stderr checks, and the
failure body with the fixture.

A check that differs fails the run, unless the fixture lists it under `knownGaps`. A listed gap that passes also
fails, so the list stays exact and a fixed gap must be removed from it.

Run it with `go test ./tests/conformance/` from the module root.
