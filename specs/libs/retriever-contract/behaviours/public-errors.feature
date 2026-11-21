# Some errors carry a fixed text that a person or a model may read. The library lists those errors once, and a presenter
# shows only their texts. Any other error gets a generic text, because its raw text can carry upstream detail.
Feature: Public errors
  A presenter shows the text of a public error and never the text of any other error.
  A missing Outline operation names the operation and the version floor.

  # Behaviour: CON-110
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: An error on the public list keeps its fixed text
    Given the error with the text "<text>"
    When the presenter asks for the public text of the error
    Then the public text is "<text>"

    Examples:
      | text                                                            |
      | not found                                                       |
      | invalid tag                                                     |
      | the change may or may not have happened; check before you retry |

  # Behaviour: CON-111
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: An error that is not on the public list has no public text
    Given the error with the text "<text>"
    When the presenter asks for the public text of the error
    Then there is no public text

    Examples:
      | text                                                                      |
      | the catalog needs at least one source                                     |
      | invalid credential name                                                   |
      | invalid kind name: use a lowercase slug of at most 32 characters          |
      | invalid plugin path: use an absolute path                                 |

  # Behaviour: CON-112
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A kind error wrapping a public error shows its text with the kind name
    Given a failure of the kind "kiwi" wrapping the public error "Outline rejected the API key"
    When the presenter asks for the public text of the failure
    Then the public text is "Kiwi rejected the API key"

  # Behaviour: CON-112
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A kind error wrapping an error that is not public has no public text
    Given a failure of the kind "kiwi" wrapping the error "the catalog needs at least one source"
    When the presenter asks for the public text of the failure
    Then there is no public text

  # Behaviour: CON-113
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A missing Outline operation names the operation and the version floor
    Given the Outline server does not offer the method "documents.foo"
    When the caller reads the error text
    Then the text is "this Outline server does not offer documents.foo; its version may be older than v1.12.0"
    And the error matches the operation unavailable error
