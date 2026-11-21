# A source may run an out-of-process plugin. A plugin reference names an absolute program path and the SHA-256 pin of the
# bytes that the user approved. A reference that does not name a program exactly is refused.
Feature: Plugin references
  A caller builds a plugin reference from a path and a pin, and the pin must be the SHA-256 of approved bytes.
  The refusal states the rule that the input broke.

  # Behaviour: CON-064
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: An absolute path with a lower-case hex pin of 64 characters is accepted
    Given the plugin path "/opt/retriever/plugins/example" and the pin of 64 lower-case hex characters
    When the caller builds a plugin reference from the path and the pin
    Then the plugin reference is built, its path is "/opt/retriever/plugins/example", and its pin is the same pin

  # Behaviour: CON-065
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A plugin path that is not absolute is refused
    Given the plugin path "<path>" and the pin of 64 lower-case hex characters
    When the caller builds a plugin reference from the path and the pin
    Then the call returns the error "invalid plugin path: use an absolute path" and no plugin reference

    Examples:
      | path                   |
      | plugins/example        |
      | ./plugins/example      |
      |                        |

  # Behaviour: CON-066
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A plugin pin that is not 64 lower-case hex characters is refused
    Given the plugin path "/opt/retriever/plugins/example" and the pin "<pin>"
    When the caller builds a plugin reference from the path and the pin
    Then the call returns the error "invalid plugin pin: use 64 lower-case hex characters" and no plugin reference

    Examples:
      | pin                                                      |
      | 0123456789ABCDEF0123456789abcdef0123456789abcdef0123456789abcdef |
      | 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcde  |
      | 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdefg |
      |                                                          |
