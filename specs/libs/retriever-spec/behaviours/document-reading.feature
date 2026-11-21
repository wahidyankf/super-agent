Feature: Reading one connector spec document
  A caller passes the bytes of one spec document to the module. The document must be one plain YAML document of at
  most 256 KiB. Each YAML construct that the spec does not allow is refused at the field that holds it, and the
  refusal is returned before any other rule runs.

  # Behaviour: SPC-001
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A valid spec document returns a model and no error
    Given the bundled example connector spec
    When the caller validates the spec document
    Then the caller gets a model of the spec
    And the caller gets no error

  # Behaviour: SPC-006
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A spec document of exactly 256 KiB is read
    Given a valid connector spec padded with a trailing comment to exactly 256 KiB
    When the caller validates the spec document
    Then the spec is accepted
    And the caller gets the model of the spec

  # Behaviour: SPC-006
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A spec document over 256 KiB is refused for the whole document
    Given a valid connector spec followed by a trailing comment that makes it longer than 256 KiB
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the whole document, not a field

  # Behaviour: SPC-007
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A YAML alias, anchor, or repeated key is refused at the field that holds it
    Given a valid connector spec that uses <construct>
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path <path>

    Examples:
      | construct                                   | path                 |
      | an alias where an operation summary belongs | /operations/1/summary |
      | an anchor on the top-level summary          | /summary             |
      | a second value for the top-level kind key   | /kind                |

  # Behaviour: SPC-008
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A YAML tag or timestamp that the spec does not allow is refused at the field that holds it
    Given a valid connector spec whose <field> carries <value>
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path <path>

    Examples:
      | field                    | value                       | path                  |
      | summary                  | a custom YAML tag           | /summary              |
      | summary                  | a bare date                 | /summary              |
      | endpoint                 | a custom YAML tag           | /endpoint             |
      | endpoint host suffix     | a custom YAML tag           | /endpoint/host_suffix |

  # Behaviour: SPC-008
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A valid connector spec with plain text, whole numbers, and true or false flags is accepted
    Given the bundled example connector spec
    When the caller validates the spec document
    Then the spec is accepted

  # Behaviour: SPC-009
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A number or flag that is not a plain finite value is refused at its field
    Given a valid connector spec in which <field> is written as <value>
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path <path>

    Examples:
      | field                      | value                              | path                                 |
      | a top-level field          | not-a-number                       | /nan                                 |
      | a top-level field          | an infinite number                 | /inf                                 |
      | a parameter required flag  | a value that is not true or false  | /operations/0/params/0/required      |
      | a parameter default        | a value that is not finite         | /operations/0/params/1/default       |

  # Behaviour: SPC-010
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A document whose root is a list is refused for the whole document
    Given a document whose root is a YAML list instead of a mapping
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the whole document, not a field

  # Behaviour: SPC-010
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A document with a YAML syntax error is refused without repeating the broken text
    Given a document with an unclosed flow sequence in its first field
    When the caller validates the spec document
    Then the spec is refused
    And no finding repeats the broken text of the document

  # Behaviour: SPC-010
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A document with a key that is not text is refused for the whole document
    Given a valid connector spec with a numeric key added beside its fields
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the whole document, not a field

  # Behaviour: SPC-010
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A document with a second YAML document after the first is refused for the whole document
    Given a valid connector spec followed by a document separator and a second document
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the whole document, not a field

  # Behaviour: SPC-010
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: An empty document is refused
    Given a document with no content
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the whole document, not a field
