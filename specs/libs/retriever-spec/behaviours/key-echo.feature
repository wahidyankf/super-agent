Feature: Key echo of a single-object read
  A read of one object may echo its key: after mapping, the reply value at a pointer must equal the request argument
  named by a parameter, or the read fails. Only a read of one object may declare the echo, and it may not be declared on
  a list read or on a write.

  # Behaviour: SPC-027
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A single-object read that echoes its key from a named parameter is accepted
    Given a connector spec whose single-object read echoes the key parameter at a reply pointer
    When the caller validates the spec document
    Then the spec is accepted

  # Behaviour: SPC-027
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A key echo on a list read is refused at the response
    Given a connector spec whose list read declares a key echo
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/0/response/echo

  # Behaviour: SPC-027
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A key echo that names a parameter the operation does not declare is refused at the parameter
    Given a connector spec whose single-object read echoes a parameter named other that it does not declare
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/0/response/echo/param

  # Behaviour: SPC-027
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A key echo with no pointer to read the reply value from is refused
    Given a connector spec whose single-object read echoes a key with no pointer
    When the caller validates the spec document
    Then the spec is refused

  # Behaviour: SPC-027
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A key echo on a write is refused
    Given a connector spec whose write declares a key echo
    When the caller validates the spec document
    Then the spec is refused
