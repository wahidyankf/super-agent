Feature: Connections to a local test server
  A read or a write made over a real connection to a local test server behaves as the network guard and the time limit
  require: a loopback address is refused without the catalog setting, a real redirect is not followed, a stall stops at the
  time limit, and a write whose connection breaks after the request left is never sent again. These scenarios run against
  a real local server, so the network path is exercised, not a stub.

  # Behaviour: ENG-074
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A read to a loopback test server is refused without the catalog setting and the server receives no request
    Given a validated read whose host resolves to the loopback address of a local test server
    When the caller runs the read without the catalog setting
    Then the read is refused
    And the server receives no request

  # Behaviour: ENG-075
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A read to a loopback test server connects over TLS with the catalog setting and sends the key in the authorization header
    Given a validated read whose host resolves to the loopback address of a local TLS test server and the catalog allows private networks
    When the caller runs the read
    Then the read succeeds
    And the key arrives in the authorization header over TLS

  # Behaviour: ENG-076
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A real 302 reply to a read is refused and the server receives one request
    Given a validated read against a local test server that answers with a 302 redirect
    When the caller runs the read
    Then the read is refused
    And the server receives exactly one request

  # Behaviour: ENG-077
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A real stall hits the spec time limit and exits with the timeout status without a second request
    Given a validated read whose spec lowers the time limit to one second against a local server that never answers
    When the caller runs the read
    Then the read exits with the timeout status
    And the server receives exactly one request

  # Behaviour: ENG-078
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A write whose connection times out after the request reached a local server is an unknown outcome and is sent once
    Given a validated write against a local server that reads nothing and answers nothing until the test ends
    When the caller runs the write with a lowered time limit
    Then the write reports an unknown outcome
    And the server receives exactly one request

  # Behaviour: ENG-079
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A write over TLS to a local server is sent once, carries the key in the authorization header, and maps its declared field
    Given a validated write against a local TLS server that accepts it
    When the caller runs the write
    Then the server receives exactly one request
    And the key arrives in the authorization header
    And the declared field is mapped

  # Behaviour: ENG-080
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A TLS handshake that fails against a local server is a known failure and the write is not sent
    Given a validated write against a local TLS server whose certificate the client does not trust
    When the caller runs the write
    Then the write fails as unreachable
    And no request reaches the server
