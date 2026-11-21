Feature: Kiwi calls over a real TLS connection
  A call made over a real TLS connection reaches the test server with the endpoint host as its TLS name and its Host header,
  and with the Bearer key. A client that does not trust the server's certificate fails the handshake, so no request and
  no key reach the server. These scenarios run against a real local TLS server, so the network path is exercised.

  # Behaviour: KIW-046
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A read over a real TLS connection reaches the server with the endpoint host and an accepted Bearer key
    Given a local TLS server that trusts the endpoint host and a gateway configured with its certificate
    When the caller runs a read
    Then the TLS name and the Host header are the endpoint host
    And the server accepts the Bearer key

  # Behaviour: KIW-047
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A client that does not trust the server certificate sends no request and no key
    Given a local TLS server whose certificate the client does not trust
    When the caller runs a read
    Then the TLS handshake fails
    And the server receives no request and no key
