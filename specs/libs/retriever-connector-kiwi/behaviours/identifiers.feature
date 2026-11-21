Feature: Kiwi identifiers
  A Kiwi product is named product-<n>, a plan is plan-<n>, and a case is case-<n>. The number n is a positive whole number of
  at most nine digits with no leading zero. A position in a request takes one kind of ID, so a case ID where a plan ID is
  needed is refused. An ID that is not canonical is refused before any request is sent, and the formatters give back the
  same text the parsers accept.

  # Behaviour: KIW-007
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour is a pure in-memory rule over identifier text and reaches no connection, process, or file; alternative-proof: retriever-connector-kiwi:test:unit
  @integration-exempt
  Scenario Outline: A product ID of the canonical form is accepted
    Given the product ID <id>
    When the caller parses the product ID
    Then the product ID is accepted

    Examples:
      | id                 |
      | product-1          |
      | product-123456789  |

  # Behaviour: KIW-008
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour is a pure in-memory rule over identifier text and reaches no connection, process, or file; alternative-proof: retriever-connector-kiwi:test:unit
  @integration-exempt
  Scenario Outline: A product ID that is not canonical is refused before any request
    Given the product ID <id>
    When the caller parses the product ID
    Then the product ID is refused as invalid

    Examples:
      | id                  |
      | product-0           |
      | product-01          |
      | product-1234567890  |
      | product--1          |
      | PRODUCT-1           |
      | product.1           |
      | product-+1          |

  # Behaviour: KIW-009
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour is a pure in-memory rule over identifier text and reaches no connection, process, or file; alternative-proof: retriever-connector-kiwi:test:unit
  @integration-exempt
  Scenario Outline: A document ID of the form <id> is accepted as a plan or a case
    Given the document ID <id>
    When the caller parses the document ID
    Then the document ID is accepted

    Examples:
      | id       |
      | plan-2   |
      | case-3   |

  # Behaviour: KIW-010
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour is a pure in-memory rule over identifier text and reaches no connection, process, or file; alternative-proof: retriever-connector-kiwi:test:unit
  @integration-exempt
  Scenario Outline: A document ID in the wrong position is refused as an invalid document ID
    Given the <id> given where a <position> ID is needed
    When the caller parses the ID for that position
    Then the ID is refused as an invalid document ID

    Examples:
      | id       | position |
      | case-3   | plan     |
      | plan-2   | case     |

  # Behaviour: KIW-011
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour is a pure in-memory rule over identifier text and reaches no connection, process, or file; alternative-proof: retriever-connector-kiwi:test:unit
  @integration-exempt
  Scenario: The formatters give back the canonical text that the parsers accept
    Given a product ID, a plan ID, and a case ID
    When the caller formats each ID and parses the text again
    Then each formatted text is the canonical text of its ID
    And each parse returns the same ID

  # Behaviour: KIW-012
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour is a pure in-memory rule over identifier text and reaches no connection, process, or file; alternative-proof: retriever-connector-kiwi:test:unit
  @integration-exempt
  Scenario: A create that names a product ID in the wrong shape sends no lookup
    Given a create that names a product ID in the wrong shape
    When the caller creates the case
    Then the create is refused and no lookup is sent
