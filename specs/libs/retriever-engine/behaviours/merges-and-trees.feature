Feature: Merged lists and trees
  A merge read maps several declared lists into one result. Each record carries the constant label of its own list, even
  when the reply holds its own kind field. A tree read flattens a nested reply depth first and gives each node its parent ID
  and its depth. A tree keeps at most 65 levels of nodes, and a paged merge or tree maps each page on its own.

  # Behaviour: ENG-081
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A merge keeps the declared order of its lists and tags each record with its list label
    Given a validated merge read of two user lists and one document list, and a reply with one record of each
    When the caller runs the read
    Then the records come back in declared list order, then in reply order
    And each record carries the constant label of its own list

  # Behaviour: ENG-082
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A merge label wins over a kind field that the reply holds on each record
    Given a validated merge read whose reply holds its own kind field on each record
    When the caller runs the read
    Then every merged record carries the constant label of its list, not the kind field

  # Behaviour: ENG-083
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario Outline: A merge whose list is <problem> fails the read as unexpected
    Given a validated merge read whose list is <problem>
    When the caller runs the read
    Then the read fails with an unexpected response

    Examples:
      | problem                             |
      | a pointer that names no list        |
      | a list whose item is not an object  |

  # Behaviour: ENG-084
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A merge that needs a converter the engine does not serve fails before any request
    Given a validated merge read whose list field names a converter the engine does not serve
    When the caller runs the read
    Then the read is unsupported and no request is sent

  # Behaviour: ENG-085
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A paged merge maps each page with the merge lists
    Given a validated paged merge read and a replay of two pages
    When the caller runs the read
    Then each page is mapped with the merge lists in declared list order and reply order

  # Behaviour: ENG-086
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A tree read returns its nodes depth first with each parent ID and depth
    Given a validated tree read and a reply with two roots, one holding a child and a grandchild
    When the caller runs the read
    Then the nodes come back depth first
    And each node carries the ID of its parent and its depth from zero

  # Behaviour: ENG-087
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A tree nested past its max depth keeps the levels up to the cap and drops each deeper node
    Given a validated tree read with the default max depth and a reply nested 70 levels deep
    When the caller runs the read
    Then the result keeps 65 levels of nodes, depths zero to 64
    And each deeper node is dropped with its subtree

  # Behaviour: ENG-088
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A tree read with a max depth of one keeps the root and its children only
    Given a validated tree read whose max depth is one
    When the caller runs the read
    Then the result holds the roots and their direct children only

  # Behaviour: ENG-089
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A tree reply nested past 64 JSON levels decodes, while any other reply nested that deep fails
    Given a tree reply nested past 64 JSON levels and a read reply of another kind nested equally deep
    When the caller runs both reads
    Then the tree reply decodes
    And the other reply fails with an unexpected response

  # Behaviour: ENG-090
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A hand-built tree model without its required ID field is refused before any request
    Given a hand-built tree model without a required ID field
    When the caller builds the engine from the model
    Then the construction fails and no request is sent

  # Behaviour: ENG-091
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A paged tree read flattens each page on its own with depths from zero on every page
    Given a validated paged tree read and a replay of a first page of two roots with a child and a short second page
    When the caller runs the read
    Then each page is flattened depth first
    And the depths start from zero on every page
