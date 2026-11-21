Feature: Plugin environment and secrets
  A plugin sees only its own credential variable, and its identity variable when the source declares one. The host reads
  each credential for each call and gives it to the plugin only through that scrubbed environment, so a secret reaches no
  error text. A plugin's output that echoes a secret reads with the secret replaced by a fixed marker. A source with no
  plugin is refused before any secret is read.

  # Behaviour: PLG-001
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The plugin process receives exactly its own credential variable and nothing of the caller
    Given a source whose credential is set and a caller environment with other variables
    When the caller makes a call through the plugin
    Then the plugin process receives only its own credential variable

  # Behaviour: PLG-002
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A source with an identity passes its identity variable as well as its credential variable
    Given a source with an identity and a credential
    When the caller makes a call through the plugin
    Then the plugin process receives its identity variable and its credential variable

  # Behaviour: PLG-003
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A source with no identity passes only its credential variable
    Given a source with a credential and no identity
    When the caller makes a call through the plugin
    Then the plugin process receives only its credential variable

  # Behaviour: PLG-004
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The host reads the source credential for each call
    Given a source with a credential
    When the caller makes two calls through the plugin
    Then the host reads the credential once for each call

  # Behaviour: PLG-005
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A missing credential or a bad identity fails the call before any process starts
    Given a source whose <fault> prevents the call
    When the caller makes a call through the plugin
    Then the call fails with its own error for the <fault>
    And no plugin process starts

    Examples:
      | fault               |
      | missing credential  |
      | bad identity        |

  # Behaviour: PLG-006
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-plugin:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A gateway for a source with no plugin is refused before any secret is read
    Given a source that names no plugin
    When the caller makes a call through the gateway
    Then the call is refused and no secret is read

  # Behaviour: PLG-007
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-plugin:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: Every occurrence of a secret in a plugin reply or error reads as the redaction marker
    Given a plugin reply and error text that hold a secret several times
    When the host redacts the text
    Then each occurrence of the secret reads as the fixed redaction marker
    And other text passes through unchanged

  # Behaviour: PLG-008
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-plugin:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: An empty secret changes no text when redacted
    Given a plugin reply and an empty secret
    When the host redacts the text
    Then the text is unchanged

