Feature: Documents and built identifiers
  A document read renders one object as a Markdown document: a title, the metadata lines, and a converted description. Every
  value is escaped, a line with an empty value is skipped only where the layout says so, and an unassigned value shows its
  default. A built identifier is made from a key in the reply, and it is built from the endpoint origin and never from a
  URL that the reply holds.

  # Behaviour: ENG-042
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: Markup in a summary or a field value stays text in the document
    Given a validated document read whose summary and field values hold a heading and a link
    When the caller runs the read
    Then each value appears as the literal text the reply holds

  # Behaviour: ENG-043
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A metadata line with an empty value is skipped where its layout skips it and kept where it does not
    Given a validated document layout with a skipped line and a line without a skip rule, both empty
    When the caller runs the read
    Then the skipped line is absent from the document
    And the line without a skip rule is present and empty

  # Behaviour: ENG-044
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: An unassigned value shows its default and a null description ends the document with empty text
    Given a validated document layout whose assignee is unset and whose description is null
    When the caller runs the read
    Then the assignee line shows the default text Unassigned
    And the document ends with no trailing line break after the empty description

  # Behaviour: ENG-045
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A full document of an object with every field renders the title, each metadata line, and the converted description
    Given a validated document read of an object that holds every declared field
    When the caller runs the read
    Then the document holds the title, each metadata line, and the converted description in layout order

  # Behaviour: ENG-046
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A document layout on a list read is refused before any request
    Given a hand-built model with a document layout on a list response
    When the caller builds the engine from the model
    Then the construction fails and no request is sent

  # Behaviour: ENG-047
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A browse URL is built from the endpoint origin and ignores a URL that the reply holds
    Given a validated read with a built browse URL and a reply that plants another address in its own URL field
    When the caller runs the read
    Then the browse URL carries the endpoint origin only

  # Behaviour: ENG-048
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A reply key outside the issue-key shape fails the read and the error names the field, never the value
    Given a validated read whose built ID field holds a key outside the issue-key shape
    When the caller runs the read
    Then the read fails naming the field
    And the error does not show the planted value

  # Behaviour: ENG-049
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A project key is the text before the last hyphen of the key
    Given a validated read that builds a project key from a document ID and from a collection ID
    When the caller runs the read
    Then each project key is the text before the last hyphen of its key

  # Behaviour: ENG-050
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario Outline: A built identifier whose key is <problem> fails the read naming the field
    Given a validated read whose built ID key is <problem>
    When the caller runs the read
    Then the read fails with an unexpected response naming the field

    Examples:
      | problem                |
      | missing                |
      | held as a number       |

  # Behaviour: ENG-051
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A list read builds the identifiers of each item from that item's own key
    Given a validated list read whose items each carry a key
    When the caller runs the read
    Then each item carries the project part of its own key

  # Behaviour: ENG-052
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A built URL past the shared output budget fails with the reply-size error
    Given a validated read whose built URL would exceed the output budget although the raw reply fits the cap
    When the caller runs the read
    Then the read fails with the reply-size error
