Feature: Bundled specs and the built-in catalog
  The module has two loaders. Parse reads a document that a user writes, so it refuses a built-in source kind and a
  catalog_id field. ParseBundled reads a document that ships inside the module, so it accepts both, and it checks each
  catalog_id against the built-in catalog. A user document can never take the bundled path.

  # Behaviour: SPC-037
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The bundled loader accepts a bundled spec whose kind is the built-in source kind jira
    Given a bundled connector spec of the built-in kind jira with three operations
    When the caller reads the document with the bundled loader
    Then the spec is accepted
    And the model has the kind jira and three operations

  # Behaviour: SPC-037
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The user loader refuses the same bundled jira spec at its kind
    Given a bundled connector spec of the built-in kind jira with three operations
    When the caller validates the spec document with the user loader
    Then the spec is refused
    And the finding names the path /kind

  # Behaviour: SPC-037
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The user loader refuses a catalog_id on an operation of a user spec
    Given a user connector spec of an ordinary kind whose first operation names a catalog_id
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/0/catalog_id

  # Behaviour: SPC-038
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A bundled operation whose parameters match its catalog row is accepted
    Given a bundled connector spec whose operations name catalog rows with the same parameter names, types, and required flags
    When the caller reads the document with the bundled loader
    Then the spec is accepted

  # Behaviour: SPC-038
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A bundled operation that differs from its catalog row is refused at the field that differs
    Given a bundled connector spec whose operation differs from its catalog row by <difference>
    When the caller reads the document with the bundled loader
    Then the spec is refused
    And the finding names the path <path>

    Examples:
      | difference                                 | path                            |
      | a parameter of another type                | /operations/1/params/1/type     |
      | a parameter that flips its required flag   | /operations/1/params/1/required |
      | a parameter name the row does not name     | /operations/2/params/0/name     |
      | a row parameter the operation omits        | /operations/1/params            |
      | a catalog_id of another built-in kind      | /operations/1/catalog_id        |
      | a catalog_id that matches no catalog row   | /operations/1/catalog_id        |
      | a catalog_id that another operation claims | /operations/2/catalog_id        |

  # Behaviour: SPC-038
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A bundled operation with no catalog row that turns off both surfaces is accepted
    Given a bundled connector spec with a hidden operation that has no catalog row and turns off its command-line and MCP surfaces
    When the caller reads the document with the bundled loader
    Then the spec is accepted

  # Behaviour: SPC-038
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A bundled operation with no catalog row that leaves one surface on is refused at that surface
    Given a bundled connector spec with a hidden operation that has no catalog row and leaves its <surface> surface on
    When the caller reads the document with the bundled loader
    Then the spec is refused
    And the finding names the path <path>

    Examples:
      | surface         | path               |
      | command-line    | /operations/0/cli  |
      | MCP             | /operations/0/mcp  |
