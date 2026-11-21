Feature: Plugin runs as a real process
  These behaviours hold for a real plugin process, not only for a fake one. A real plugin sees only its own credential, and
  its output shows the redaction marker where the secret was. A silent plugin is killed at the time limit and the run exits
  with 124. A pinned mismatch starts no process. A flooded frame fails the call. A plugin that exits with status 127 at once
  makes the run exit with 2. A source served through a real plugin maps its collections onto domain values.

  # Behaviour: PLG-041
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A real plugin sees exactly its own credential variable and, when declared, its identity variable
    Given a real plugin that prints its environment and a source with a credential and an identity
    When the caller makes a call through the real plugin
    Then the plugin sees only its own credential variable and its identity variable

  # Behaviour: PLG-042
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The output of a real plugin that echoes its environment holds no secret
    Given a real plugin that echoes its environment in its reply and on its error output
    When the caller makes a call through the real plugin
    Then the reply and the error output both read with the redaction marker

  # Behaviour: PLG-043
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A real silent plugin is killed at the time limit and the run exits with 124
    Given a real plugin that never answers and a time limit that has run out
    When the caller runs the call
    Then the host kills the plugin
    And the run exits with status 124

  # Behaviour: PLG-044
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A real plugin whose bytes differ from the pin starts no process
    Given a real plugin whose bytes differ from the pinned digest
    When the caller runs the call
    Then the call fails with the pin-mismatch error
    And no plugin process starts

  # Behaviour: PLG-045
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A real plugin that floods its output past 1 MiB fails the call
    Given a real plugin that writes a frame over 1 MiB
    When the caller runs the call
    Then the call fails

  # Behaviour: PLG-046
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A real plugin that exits with status 127 at once makes the run exit with 2 and never 127
    Given a real plugin that exits with status 127 before it answers
    When the caller runs the call
    Then the host reports that the plugin answered nothing
    And the run exits with status 2

  # Behaviour: PLG-047
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A source served through the real fixture plugin maps its collections onto domain values
    Given a real fixture plugin that serves a source with collections
    When the caller lists the collections through the gateway
    Then the served collections map onto domain values
