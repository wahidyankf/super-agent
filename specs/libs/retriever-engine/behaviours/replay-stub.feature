Feature: Replay stub
  The replay package is a stub source for tests of a connector. It answers a request with the canned reply of the spec
  example that the request matches, and it records every request it receives. A scripted reply list is used in order and
  its last reply repeats. A stub refuses to start from an example that the catalog or the JSON form refuses.

  # Behaviour: ENG-174
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the replay stub is an in-memory test double with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A request that matches a read example receives the canned status, header, and body of that example
    Given a replay stub built from a validated spec with a read example
    When a request matching the example is sent through the stub
    Then the reply holds the canned status, header, and body of the example

  # Behaviour: ENG-175
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the replay stub is an in-memory test double with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A write example answers its matching write and the stub records the body the write sent
    Given a replay stub built from a validated spec with a write example
    When the write is sent through the stub
    Then the reply holds the canned write reply
    And the record holds the body that the write sent

  # Behaviour: ENG-176
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the replay stub is an in-memory test double with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: The stub records every request and counts the requests that matched no example
    Given a replay stub built from a validated spec
    When one matching request and one unmatched request are sent
    Then the record holds both requests
    And the unmatched count is one

  # Behaviour: ENG-177
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the replay stub is an in-memory test double with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A scripted reply list is used once in order and its last reply repeats
    Given a replay stub scripted with two replies
    When three requests are sent through the stub
    Then the first request receives the first reply
    And the second and third requests receive the second reply

  # Behaviour: ENG-178
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the replay stub is an in-memory test double with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A scripted failure returns its error and the request is still recorded
    Given a replay stub scripted with a transport failure
    When a request is sent through the stub
    Then the stub returns the scripted error
    And the request is recorded

  # Behaviour: ENG-179
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the replay stub is an in-memory test double with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A reply with no body on its example sends an empty body
    Given a replay stub built from a validated spec with a canned reply that has no body
    When a matching request is sent through the stub
    Then the reply body is empty

  # Behaviour: ENG-180
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the replay stub is an in-memory test double with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: The stub reads and closes the body of each request it receives
    Given a replay stub and a request that carries a body
    When the request is sent through the stub
    Then the stub has read and closed the request body

  # Behaviour: ENG-181
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the replay stub is an in-memory test double with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: Two read examples with the same request answer with the first one
    Given a validated spec with two read examples whose requests are the same
    When a request that matches both examples is sent through the stub
    Then the reply is the canned reply of the first example

  # Behaviour: ENG-182
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the replay stub is an in-memory test double with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A stub is refused when an example request does not bind to its operation
    Given a spec whose example request names a parameter that the catalog refuses
    When the caller builds the stub
    Then the construction fails

  # Behaviour: ENG-183
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the replay stub is an in-memory test double with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A stub is refused when an example body has no JSON form
    Given a spec whose example body has no JSON form
    When the caller builds the stub
    Then the construction fails
