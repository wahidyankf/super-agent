Feature: Engine construction and gateway reads
  A caller builds an engine from a validated spec and its ports. The engine is built only when the spec, the kind, the
  host suffix, the ports, and the authentication all fit together. A gateway read runs only when the spec gives the
  operation that role, and a read the spec does not serve fails before any request is sent.

  # Behaviour: ENG-001
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: An engine is built from the validated fixture without any edit
    Given the validated example spec and its configured ports
    When the caller builds the engine
    Then the engine is built and no error is returned

  # Behaviour: ENG-002
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: An engine is refused for a spec of another kind
    Given the validated example spec of kind example-wiki
    When the caller builds an engine for the kind jira
    Then the construction fails and no engine is returned

  # Behaviour: ENG-003
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: An engine is refused for a host outside the host suffix of the spec
    Given the validated example spec with the host suffix .example.test
    When the caller builds an engine whose host ends in another domain
    Then the construction fails and no engine is returned

  # Behaviour: ENG-004
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: An engine is refused when a port it needs is not configured
    Given the validated example spec and a set of ports with one port missing
    When the caller builds the engine
    Then the construction fails and no engine is returned

  # Behaviour: ENG-005
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: An engine is refused for basic authentication with no identity port
    Given the validated example spec with basic authentication and no identity port configured
    When the caller builds the engine
    Then the construction fails and no engine is returned

  # Behaviour: ENG-006
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: Basic authentication sends the identity and the key in the authorization header
    Given the validated spec with basic authentication and configured identity and key ports
    When the caller runs a read
    Then the request carries an authorization header that holds the identity and the key

  # Behaviour: ENG-007
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: Header authentication sends the key in the header the spec declares
    Given the validated spec with header authentication and a declared header name
    When the caller runs a read
    Then the request carries the key in the declared header

  # Behaviour: ENG-008
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario Outline: A time limit the spec sets applies to the request, and the default applies when the spec sets none
    Given a validated spec whose time limit is <limit>
    When the caller runs a read
    Then the request runs under a time limit of <effective> seconds

    Examples:
      | limit               | effective |
      | 10 seconds          | 10        |
      | no time limit       | 30        |

  # Behaviour: ENG-009
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario Outline: A hand-built model that breaks a rule of the spec is refused when the engine is built
    Given a hand-built model with <fault>
    When the caller builds the engine from the model
    Then the construction fails and no request is sent

    Examples:
      | fault                                         |
      | a partial path slot                           |
      | a reserved header name in its header list     |
      | a pointer that does not start with a slash    |

  # Behaviour: ENG-010
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A collections read refuses a spec that gives no operation the list-collections role
    Given a validated spec with no list-collections operation
    When the caller lists the collections
    Then the read is unsupported and no request is sent

  # Behaviour: ENG-011
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A collections read returns the ID and name of each collection
    Given a validated spec with the list-collections and document roles
    When the caller lists the collections
    Then the result holds the ID and name of each collection, and no parent ID or depth

  # Behaviour: ENG-012
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A collections read fails with the read's error when the source reply fails
    Given a validated spec whose list-collections read gets a failed reply
    When the caller lists the collections
    Then the read returns the error of the failed reply and no collections

  # Behaviour: ENG-013
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A collections read refuses a reply whose items are not a list
    Given a validated spec whose list-collections read replies with items that are not a list
    When the caller lists the collections
    Then the read fails with an unexpected response

  # Behaviour: ENG-014
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A read of an operation the spec does not declare is unsupported and sends no request
    Given the validated example spec
    When the caller runs a read of an operation id the spec does not declare
    Then the read is unsupported and no request is sent

  # Behaviour: ENG-015
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: The read path refuses a write operation and sends no request
    Given the validated example spec with a write operation
    When the caller runs the write operation through the read path
    Then the read is refused and no request is sent

  # Behaviour: ENG-016
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A read that needs a converter the engine does not serve fails before any request
    Given a validated read whose field names a converter the engine does not serve
    When the caller runs the read
    Then the read is unsupported and no request is sent
