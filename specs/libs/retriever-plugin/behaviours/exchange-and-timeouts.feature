Feature: Plugin exchange, reply shape, and timeouts
  The host sends one JSON-RPC 2.0 request and reads one reply. A reply is accepted only when it is one JSON-RPC 2.0 object
  with id 1 and exactly one of result or error. A frame over 1 MiB fails the call. The error output of a plugin is kept to
  its first 64 KiB. A plugin that never answers is killed at the time limit, which is 30 seconds by default, and the
  timeout maps to exit status 124. A cancelled caller ends the plugin and fails with the caller's own error.

  # Behaviour: PLG-009
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: One call sends one request and reads one reply, passes the error output through, and removes the private folder
    Given a plugin that answers one request with a result
    When the caller makes a call through the plugin
    Then the request carries the method of the call
    And the result decodes
    And the error output passes through
    And the private folder is removed

  # Behaviour: PLG-010
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A reply that breaks the one-object rule is refused with a protocol error
    Given a plugin whose reply is <reply>
    When the caller makes a call through the plugin
    Then the call fails with a protocol error

    Examples:
      | reply                                                          |
      | an empty frame or a blank frame                                |
      | a batch of replies                                             |
      | text that is not JSON                                          |
      | a JSON value that is not an object                             |
      | a reply followed by extra data                                 |
      | a reply with the version 1.0                                   |
      | a reply with no version                                        |
      | a notification with no id                                      |
      | a reply whose id is null, 2, or the text "1"                   |
      | a reply that holds an error object                             |
      | a reply that holds both a result and an error                  |
      | a reply that holds no result                                   |

  # Behaviour: PLG-011
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A reply with an object result passes through
    Given a plugin whose reply holds an object result
    When the caller makes a call through the plugin
    Then the result passes through unchanged

  # Behaviour: PLG-012
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A reply with a null result is refused by the gateway as a missing object
    Given a plugin whose reply holds a null result
    When the caller reads a document through the gateway
    Then the gateway refuses the reply as a missing object

  # Behaviour: PLG-013
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A frame over 1 MiB fails the call
    Given a plugin that floods its output with a frame over 1 MiB
    When the caller makes a call through the plugin
    Then the call fails

  # Behaviour: PLG-014
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The error output of a plugin is kept to its first 64 KiB however it is chunked
    Given a plugin whose error output is longer than 64 KiB
    When the caller makes a call through the plugin
    Then the kept error output holds exactly the first 64 KiB

  # Behaviour: PLG-015
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A plugin that never answers is killed at the time limit and the call fails with exit status 124
    Given a plugin that never answers and a time limit that has run out
    When the caller makes a call through the plugin
    Then the plugin process is killed
    And the call fails with the timeout error
    And the exit status is 124

  # Behaviour: PLG-016
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-plugin:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The timeout error is retryable and maps to exit status 124
    Given a call that fails with the timeout error
    When the caller reads the error
    Then the error is marked retryable
    And it maps to exit status 124

  # Behaviour: PLG-017
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-plugin:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A time limit that is not set takes the 30 second default
    Given a plugin with no time limit set
    When the caller makes a call through the plugin
    Then the time limit is 30 seconds

  # Behaviour: PLG-018
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A cancelled caller ends the plugin and the call fails with the caller's own error
    Given a plugin that never answers and a caller whose context is cancelled
    When the caller makes a call through the plugin
    Then the plugin process is killed
    And the call fails with the context's own error

  # Behaviour: PLG-019
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A protocol failure from the exchange passes through unchanged, and any other failure maps onto the fixed rules
    Given a call that fails once with a protocol failure and once with another failure
    When the caller reads each error
    Then the protocol failure keeps its rule
    And the other failure maps onto the fixed protocol rule
