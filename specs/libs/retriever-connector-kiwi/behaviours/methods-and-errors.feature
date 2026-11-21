Feature: Kiwi method table, argument guard, and errors
  The adapter sends only the ten Kiwi methods that its table allows. Three of them change Kiwi, and only those three
  have the write failure rule. An argument that Kiwi refuses is refused by the adapter at any depth before the request
  leaves. Each HTTP status and each JSON-RPC error maps to one error, and no error text carries the key, even when a reply
  echoes it. Every failure names Kiwi as its kind.

  # Behaviour: KIW-013
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour is a fixed in-memory table of method names and reaches no connection or file; alternative-proof: retriever-connector-kiwi:test:unit
  @integration-exempt
  Scenario: The method table holds exactly the ten allowed methods and no removal or authentication method
    Given the Kiwi method table of the adapter
    When the caller lists the allowed methods
    Then the list holds exactly ten methods
    And no method that removes, deletes, or authenticates is in the list

  # Behaviour: KIW-014
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour is a fixed in-memory table of method names and reaches no connection or file; alternative-proof: retriever-connector-kiwi:test:unit
  @integration-exempt
  Scenario: Exactly three allowed methods change Kiwi, and only those three carry the write failure rule
    Given the Kiwi method table of the adapter
    When the caller lists the methods that change Kiwi
    Then the list holds the three write methods and each of them is in the allowed list

  # Behaviour: KIW-015
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A method outside the allowed list never reaches the network
    Given a call that names a method outside the allowed list
    When the caller makes the call
    Then the call is refused and no request reaches the network

  # Behaviour: KIW-016
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: An argument that Kiwi refuses is refused by the adapter at any depth before the request leaves
    Given a call with a forbidden argument key at a nested depth
    When the caller makes the call
    Then the call is refused and no request is sent

  # Behaviour: KIW-017
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A value that JSON cannot hold is refused before any request
    Given a call with an argument value that JSON cannot encode
    When the caller makes the call
    Then the call is refused and no request is sent

  # Behaviour: KIW-018
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: Every request that the adapter sends passes the argument guard
    Given a scripted gateway through which every request path of the adapter runs
    When the caller runs every path
    Then every recorded request passes the argument guard
    And every allowed method is exercised

  # Behaviour: KIW-019
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: An HTTP status of <status> on a <call> call fails with <error>
    Given a scripted Kiwi gateway that answers a <call> call with HTTP status <status>
    When the caller makes the call
    Then the call fails with <error>

    Examples:
      | status | call  | error                      |
      | 401    | read  | the key-rejected error     |
      | 403    | read  | the access-denied error    |
      | 404    | read  | the unexpected-response error |
      | 500    | read  | the upstream-failure error |
      | 500    | write | the unknown-outcome error  |
      | 503    | write | the unknown-outcome error  |
      | 401    | write | the key-rejected error     |

  # Behaviour: KIW-020
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A JSON-RPC error code and its message map to one error each
    Given a scripted Kiwi gateway that answers with each JSON-RPC error code in the table
    When the caller makes a call for each code
    Then each call fails with the error of its code

  # Behaviour: KIW-021
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: No error text carries the key, even when the reply echoes it
    Given a scripted Kiwi gateway whose error reply echoes the key
    When the caller makes the call
    Then the error text holds no key

  # Behaviour: KIW-022
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A missing key is named by its variable and nothing is sent
    Given a gateway whose key variable is not set
    When the caller makes a call
    Then the error names the key variable
    And no request is sent

  # Behaviour: KIW-023
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: Every failure the adapter returns names Kiwi as its kind
    Given a scripted Kiwi gateway that fails each kind of call
    When the caller makes each call
    Then each failure names Kiwi as its kind
