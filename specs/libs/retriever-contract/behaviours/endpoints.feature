# An endpoint is the HTTPS origin of a source: a host name only, with no path, port, or user information. Retriever refuses
# every other shape before it sends a request.
Feature: Source endpoints
  A caller builds the endpoint of a source from an address, and the address must be an HTTPS host name.
  The refusal names the rule that the address broke.

  # Behaviour: CON-054
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: An HTTPS host name is accepted, and one trailing slash is dropped
    Given the address "<address>"
    When the caller builds an endpoint from the address
    Then the endpoint is accepted, its host is "docs.example.test", and its text is "https://docs.example.test"

    Examples:
      | address                       |
      | https://docs.example.test     |
      | https://docs.example.test/    |

  # Behaviour: CON-055
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A plain HTTP address is refused because HTTPS is required
    Given the address "<address>"
    When the caller builds an endpoint from the address
    Then the call returns the error "HTTPS is required for Outline" and no endpoint

    Examples:
      | address                   |
      | http://docs.example.test  |
      | http://docs.example.test/ |

  # Behaviour: CON-056
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A non-origin address is refused as an invalid endpoint
    Given the address "<address>"
    When the caller builds an endpoint from the address
    Then the call returns the error "invalid endpoint" and no endpoint

    Examples:
      | address                       |
      | ftp://docs.example.test       |
      | docs.example.test             |
      | https://docs.example.test/wiki |
      | https://docs.example.test:8443 |
      | https://user@docs.example.test |
      | https://localhost             |
      | https://example               |
      | https://192.0.2.10            |
      | https://docs.example.1test    |
      | https://Docs.example.test     |
