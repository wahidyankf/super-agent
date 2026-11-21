Feature: Shapes of a reply that a read maps
  A read maps its reply in one shape: a list of items, the first item of a list, a merge of two to eight labelled lists,
  or a tree that flattens nested nodes. A response names at most one of these shapes, so two shapes in one response are
  refused. A tree needs a required id field, because each node's parent is the id of the parent node.

  # Behaviour: SPC-025
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A response that names two reply shapes at once is refused at the response
    Given a connector spec whose read names <shapes> in one response
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/4/response

    Examples:
      | shapes                  |
      | items and first        |
      | merge and items        |
      | merge and first        |
      | merge and top-level fields |

  # Behaviour: SPC-025
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A merge of lists is accepted only with two to eight lists
    Given a connector spec whose merge read has <count> lists
    When the caller validates the spec document
    Then the spec is <verdict>

    Examples:
      | count | verdict  |
      | 1     | refused  |
      | 2     | accepted |
      | 8     | accepted |
      | 9     | refused  |

  # Behaviour: SPC-025
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A merge label or label field that is not lower-case is refused at that label
    Given a connector spec whose merge read sets <field> to an upper-case name
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path <path>

    Examples:
      | field                          | path                                           |
      | the label of its first list    | /operations/4/response/merge/lists/0/label     |
      | the label field of the merge   | /operations/4/response/merge/label_field       |

  # Behaviour: SPC-025
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A first-item read is accepted when its response names only the first list
    Given a connector spec whose read maps the first item of a list with a first pointer
    When the caller validates the spec document
    Then the spec is accepted

  # Behaviour: SPC-026
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A tree read is accepted when it rides on an items list and names a required id
    Given a connector spec whose read flattens a nested tree of nodes that each carry a required id
    When the caller validates the spec document
    Then the spec is accepted

  # Behaviour: SPC-026
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A tree read that breaks its shape is refused at the part that breaks it
    Given a connector spec whose tree read <change>
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path <path>

    Examples:
      | change                                              | path                                  |
      | has no items list to ride on                        | /operations/4/response/items          |
      | also names a first item                             | /operations/4/response                |
      | sets a depth cap above 64                           | /operations/4/response/tree/max_depth |

  # Behaviour: SPC-026
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A tree read whose reply holds no required id field is refused at the tree
    Given a connector spec whose tree read maps its nodes without a required id field
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/4/response/tree

  # Behaviour: SPC-036
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A read that names the exact unavailable message of its method is accepted
    Given a connector spec whose read names the exact body message that proves its method is missing
    When the caller validates the spec document
    Then the spec is accepted
