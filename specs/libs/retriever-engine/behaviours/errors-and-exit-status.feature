Feature: Error codes and exit statuses
  Every failure carries a stable code that names its area and its reason, and the area is the source kind, except for a
  refused network connection, which names the connector area. The command line exits with status 124 for a timeout or a
  rate-limit wait, and with status 2 for any other failure that did not complete.

  # Behaviour: ENG-053
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A failure carries a stable code that names its source kind and its reason
    Given a validated read of a source kind that fails with a known reason
    When the caller runs the read
    Then the failure carries the stable code for that source kind and reason

  # Behaviour: ENG-054
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A refused network connection carries a code that names the connector area
    Given a validated read whose host is refused by the network guard
    When the caller runs the read
    Then the failure carries a code in the connector area

  # Behaviour: ENG-055
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario Outline: A failure of kind <outcome> maps to exit status <status>
    Given a failure of the kind <outcome>
    When the caller reads the exit status of the failure
    Then the exit status is <status>

    Examples:
      | outcome                  | status |
      | a timeout                | 124    |
      | a rate-limit wait        | 124    |
      | any other failure        | 2      |
