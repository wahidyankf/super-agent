Feature: Checks of a spec model built in code
  A caller may build a spec model in code instead of reading a document. The exported check functions return the
  findings for that model. Each finding names the field path and a fixed rule. A check of a model with no broken rule
  returns no finding.

  # Behaviour: SPC-046
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The access matrix check names an admin label over GET at the method
    Given a spec model whose one operation is labelled admin and sent with method GET
    When the caller runs the access matrix check
    Then the check returns one finding
    And the finding names the path /operations/0/method

  # Behaviour: SPC-046
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The access matrix check names a read over POST that has no body at the body
    Given a spec model whose one operation is a read over POST with no body
    When the caller runs the access matrix check
    Then the check returns one finding
    And the finding names the path /operations/0/body

  # Behaviour: SPC-046
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The object field check names a fallback pointer on a field that has no pointer to fall back from
    Given a spec model whose field id sets a fallback pointer and no pointer
    When the caller runs the object field check
    Then the check returns one finding
    And the finding names the path /operations/0/response/fields/id/fallback_pointer

  # Behaviour: SPC-046
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A check of a hand-built model that breaks one rule returns a finding that names a path and a fixed rule
    Given a spec model that breaks <rule>
    When the caller runs <check>
    Then the check returns a finding that names a field path and a fixed rule

    Examples:
      | rule                                              | check               |
      | a tree with a max_depth above 64                  | the tree check      |
      | a merge with a list count out of its range        | the merge check     |
      | a nested record with 33 sub-fields                | the object field check |
      | a merge pair in a response shape                  | the response shape check |

  # Behaviour: SPC-046
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The access matrix check returns no finding for the bundled example model
    Given the spec model of the bundled example connector spec
    When the caller runs the access matrix check
    Then the check returns no finding

  # Behaviour: SPC-046
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The bundled mapping check returns a finding that names a field path and a fixed rule for a broken catalog mapping
    Given a bundled spec model whose operations break the catalog mapping rule
    When the caller runs the bundled mapping check
    Then the check returns a finding that names a field path and a fixed rule
