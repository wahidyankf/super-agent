Feature: Markdown document layout of a single-object read
  A single-object read may render its fields as one Markdown document: a title, a list of metadata lines with closed
  kinds, and a description body with a fixed heading. Only a single-object read may declare a layout, and the layout
  names only scalar fields of the operation. A layout and a markdown output field cannot both be set.

  # Behaviour: SPC-028
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A single-object read with a title, metadata lines, and a description body is accepted
    Given a connector spec whose single-object read renders a document with a title, metadata lines, and a body
    When the caller validates the spec document
    Then the spec is accepted

  # Behaviour: SPC-028
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A document layout on a list read is refused at the document
    Given a connector spec whose list read declares a document layout
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/0/response/document

  # Behaviour: SPC-028
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A document layout that sits beside a markdown output field is refused at the document
    Given a connector spec whose single-object read sets both a document layout and a markdown output field
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/0/response/document

  # Behaviour: SPC-028
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A metadata line whose label is outside the closed form is refused at its label
    Given a connector spec whose first metadata line has a label that is not in the closed form
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/0/response/document/lines/0/label

  # Behaviour: SPC-028
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A join on a verbatim metadata line is refused at the join
    Given a connector spec whose verbatim metadata line also sets a join
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/0/response/document/lines/9/join

  # Behaviour: SPC-028
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A description body whose converter does not match its field is refused at the converter
    Given a connector spec whose description body names a converter that its field does not use
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/0/response/document/body/convert

  # Behaviour: SPC-028
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A document layout whose line names a field the operation does not declare is refused
    Given a connector spec whose document layout names a field that the operation does not declare
    When the caller validates the spec document
    Then the spec is refused
    And the finding names a path under the document of the first operation
