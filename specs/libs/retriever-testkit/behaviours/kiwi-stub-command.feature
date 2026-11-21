# The Kiwi stub command serves one Kiwi stub as a process. It needs a certificate file path and the synthetic key in its
# environment. It exits 0 after a clean run and 1 after a failed one, and it prints the failure on standard error.
Feature: Kiwi stub command
  A test runs the Kiwi stub command with flags and the synthetic key. The command checks its inputs before it listens.

  # Behaviour: TK-067
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  Scenario: The command refuses to start without a certificate file path
    Given the Kiwi stub command with no -ca-file flag and the key set
    When the command runs
    Then the command exits with status 1 and prints "the -ca-file flag is required" and listens on no port

  # Behaviour: TK-068
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  Scenario: The command refuses to start when the key variable is not set
    Given the Kiwi stub command with the -ca-file flag and no key set
    When the command runs
    Then the command exits with status 1 and prints "KIWISTUB_API_KEY must hold the synthetic key" and listens on no port

  # Behaviour: TK-069
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  Scenario: The command refuses a flag that it does not know and names the flag
    Given the Kiwi stub command with the flag "-no-such-flag"
    When the command runs
    Then the command exits with status 1 and prints a diagnostic that names the flag and listens on no port

  # Behaviour: TK-070
  Scenario: The command writes the certificate and then prints its listening address
    Given the Kiwi stub command with the -ca-file flag and the key set
    When the command starts
    Then the certificate file is written and the command prints "kiwi stub listening on" followed by its URL

  # Behaviour: TK-071
  Scenario: A failure after start stops the stub and prints to standard error
    Given the Kiwi stub command whose certificate file cannot be written
    When the command runs
    Then the stub is stopped, the command exits with status 1, and the failure is printed on standard error

  # Behaviour: TK-071
  Scenario: A clean run exits with status 0
    Given the Kiwi stub command with the -ca-file flag and the key set that runs until its stop signal
    When the command runs and the stop signal arrives
    Then the command exits with status 0
