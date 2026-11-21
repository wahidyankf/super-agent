# unit

The hermetic unit bindings of the plugin's behaviour scenarios. Each file binds one feature file, and each test function
binds one scenario, or one Scenario Outline with all of its rows. The tests are in the external package `unit_test` and
call the library's exported API. Each feature file's tests use the `Seams` of `plugin.Call`, which inject the file
copies, the process start, and the clock.

No test here reads a real file, starts a process, or waits on the clock. The files, processes, and timers are doubles
from [doubles_test.go](./doubles_test.go).

- [doubles_test.go](./doubles_test.go): the in-memory files, the fake command and its pipes, the timers, and the
  gateway builder that the other files share. It has no scenario of its own.
- [pinning-and-start_test.go](./pinning-and-start_test.go): the pin check, the copy, the start failures, and the kill.
- [environment-and-secrets_test.go](./environment-and-secrets_test.go): the scrubbed environment and the redaction.
- [exchange-and-timeouts_test.go](./exchange-and-timeouts_test.go): the one-request exchange, the reply shapes, the
  frame and error-output caps, and the time limit.
- [gateway-mapping_test.go](./gateway-mapping_test.go): the mapping of each gateway method's reply onto domain values.
- [real-process_test.go](./real-process_test.go): the rules a real process must meet, bound through the doubles.
