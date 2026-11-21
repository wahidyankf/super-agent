Feature: Rules that compare two fields of a connector spec
  Some rules cannot be stated in a schema, because each one compares two fields of the document: an operation id against
  its kind, a request slot against a declared parameter, a name against a read. These rules run only after the YAML reader
  and the schema accept the document. Each broken rule is refused at the field that breaks it.

  # Behaviour: SPC-017
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: An operation id that does not fit its kind or repeats another is refused at that id
    Given a connector spec in which <change>
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path <path>

    Examples:
      | change                                                           | path                |
      | the first operation id starts with a kind other than the spec kind | /operations/0/id    |
      | the second operation id repeats the first operation id           | /operations/1/id    |

  # Behaviour: SPC-018
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A path, body, or header slot that names no declared parameter is refused at the slot
    Given a connector spec in which <change>
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path <path>

    Examples:
      | change                                                                  | path                              |
      | the path of the second operation has a slot with no declared parameter  | /operations/1/path                |
      | a body slot of the third operation names no body parameter              | /operations/2/body/title          |
      | a body slot inside a list names no body parameter                       | /operations/2/body/labels/1       |
      | a header slot of the second operation names no parameter                | /operations/1/headers/X-Tenant    |

  # Behaviour: SPC-018
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A declared path parameter that the path does not use is refused at that parameter
    Given a connector spec whose second operation declares a path parameter that its path never uses
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/1/params/0/name

  # Behaviour: SPC-019
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: Two parameters of one operation that share a name are refused at the second name
    Given a connector spec whose first operation declares two parameters with the same name
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/0/params/1/name

  # Behaviour: SPC-019
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A cursor parameter name that is not a query parameter of the operation is refused
    Given a connector spec whose first operation names a cursor parameter that it does not declare as a query parameter
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/0/pagination/cursor_param

  # Behaviour: SPC-019
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A name lookup that does not point at a read operation of the same spec is refused
    Given a connector spec whose second operation resolves a name through <target>
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/1/params/0/resolve/operation

    Examples:
      | target                                              |
      | an operation id that the spec does not declare      |
      | a write operation of the same spec                  |

  # Behaviour: SPC-019
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A markdown output that names no field of the operation is refused at the markdown field
    Given a connector spec whose second operation names a markdown output field that the operation does not map
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/1/response/markdown
