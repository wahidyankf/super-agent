# Behaviour runner

Runs the Gherkin specs in [specs/behaviours](../../../../specs/behaviours/README.md) with godog v0.16.0, in strict mode,
so an undefined or pending step fails the run. Each level has its own entry point and tag:

- Unit (`go test ./tests/behaviour/ -run TestUnitBehaviours`): the `@unit` scenarios. The CLI runs in process with no
  source, so nothing reaches a gateway, a stub, or the network.
- Integration (`go test -tags integration ./tests/behaviour/ -run TestIntegrationBehaviours`): the `@integration`
  scenarios. The real adapters run in process, wired the way the composition root wires them, against loopback stubs.
- End-to-end (`go test -tags e2e ./tests/behaviour/ -run TestE2EBehaviours`): the `@e2e` scenarios. The test builds the
  binary once from the module root and runs it as a process, with the scenario's environment only.

All levels share one set of step definitions. Scenarios run one at a time, because the integration level changes the
working directory and the process environment.

## Isolation

- Every scenario has its own folder. Its home and configuration folders point inside that folder.
- A stub starts when a step names it. Its seam variables then point the command at the stub.
- Every other outbound request goes through a proxy variable that names a closed loopback port, so it is refused at
  once and never reaches a real host.

## Files

- `world_test.go`: the state of one scenario and the three levels it runs at.
- `backend_test.go`: where a level keeps a scenario; the in-memory backend and its export sink.
- `backend_disk_test.go`: the disk backend and the built binary, tagged `integration || e2e`.
- `inprocess_wiring_test.go`: the in-process wiring, mirrored from `cmd/retriever`.
- `stubs_test.go`: the stub lifecycle and the seam variables; it loads the synthetic Outline replies.
- `process_test.go`: the `serve` and `mcp` sessions and the GraphQL request.
- `steps_cli_test.go`, `steps_stub_test.go`, `steps_process_test.go`: the step definitions.
- `suite_test.go`, `unit_suite_test.go`, `integration_suite_test.go`, `e2e_suite_test.go`: the suite entry points.
- [support/synthetic/](../support/synthetic/README.md): synthetic Outline replies for the write and admin methods, embedded.
