Feature: Kiwi reads and documents
  A caller reads Kiwi products, collection trees, and documents. A product list is one request. A collection tree lists each
  plan, then its cases, then its child plans, depth first, and it holds at most 200 plans. A plan or a case that the caller
  asks for by its ID is returned as a document, and an unknown plan or case is named as not found. A read refuses any
  operation that is not one of the Kiwi reads, and it never sends a request for it.

  # Behaviour: KIW-024
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The product list returns every product in one request
    Given a scripted Kiwi gateway that holds two products
    When the caller lists the products
    Then the result holds both products
    And exactly one request is sent

  # Behaviour: KIW-025
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The collection tree of a product lists each plan, then its cases, then its child plans
    Given a scripted Kiwi gateway whose product holds a plan with cases and a child plan
    When the caller reads the collection tree of the product
    Then the nodes come back as the plan, its cases, and then the child plan

  # Behaviour: KIW-026
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A product with no plans has an empty collection tree and no error
    Given a scripted Kiwi gateway whose product holds no plans
    When the caller reads the collection tree of the product
    Then the tree is empty and no error is returned

  # Behaviour: KIW-027
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A product that does not exist is named as not found
    Given a scripted Kiwi gateway that holds no product with the asked ID
    When the caller reads the collection tree of that product
    Then the read fails as not found

  # Behaviour: KIW-028
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A collection tree with more than 200 plans is refused before any case is read
    Given a scripted Kiwi gateway whose product holds 201 plans
    When the caller reads the collection tree of the product
    Then the read fails with the page-cap error and no case read is sent

  # Behaviour: KIW-029
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: Plans whose parent links form a loop are each listed once
    Given a scripted Kiwi gateway whose plans link each other as parents in a loop
    When the caller reads the collection tree of the product
    Then each plan is listed exactly once

  # Behaviour: KIW-030
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: Two cases with the same sort key are listed in ID order whatever order the server sent them in
    Given a scripted Kiwi gateway that sends two cases with the same sort key in reverse ID order
    When the caller reads the collection tree of the product
    Then the cases are listed in ID order

  # Behaviour: KIW-031
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The document of a plan holds its ID, title, collection, and the cases it lists
    Given a scripted Kiwi gateway that holds a plan with two cases
    When the caller reads the document of the plan
    Then the document holds the plan's ID, title, and collection
    And the document lists its cases

  # Behaviour: KIW-032
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The document of a case names its status, priority, and category by name
    Given a scripted Kiwi gateway that holds a case with a status, a priority, and a category
    When the caller reads the document of the case
    Then the document names the status, the priority, and the category
    And the product is the product of the category

  # Behaviour: KIW-033
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A document read for a missing <kind> is named as not found
    Given a scripted Kiwi gateway that holds no <kind> with the asked ID
    When the caller reads the document of that <kind>
    Then the read fails as not found

    Examples:
      | kind |
      | plan |
      | case |

  # Behaviour: KIW-034
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The lookup tables of the case documents are read once for the whole gateway
    Given a scripted Kiwi gateway and two case documents to read
    When the caller reads both documents
    Then each lookup table is read exactly once

  # Behaviour: KIW-035
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A reply that names a plan or case ID outside the ID range fails the read as unexpected
    Given a scripted Kiwi gateway whose reply names an ID of zero, a negative number, or a number too large for the range
    When the caller reads the collection tree or the document of that ID
    Then the read fails with an unexpected response

  # Behaviour: KIW-036
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the refusal happens before any request is built, so no connection or file is reached; alternative-proof: retriever-connector-kiwi:test:unit
  @integration-exempt
  Scenario: A read that names an operation outside the Kiwi read table is refused before any request
    Given a read of an operation that is unknown, a write, a read of another kind, or a read with a bad argument
    When the caller runs the read
    Then the call is refused and no request is sent

  # Behaviour: KIW-037
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A fault in any reply that a read needs is returned and the read goes no further
    Given a scripted Kiwi gateway whose second reply of a read is a fault
    When the caller runs the read
    Then the read returns that fault
    And no later reply is requested

  # Behaviour: KIW-038
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A reply that is not an object of case IDs to whole numbers fails the read as unexpected
    Given a scripted Kiwi gateway whose sort-key reply is a list, a scalar, a non-number key, or a value that is not a whole number
    When the caller reads the case order of a product
    Then the read fails with an unexpected response

  # Behaviour: KIW-039
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: An unknown field in a row of the reply is ignored and a null where text is expected keeps the zero value
    Given a scripted Kiwi gateway whose rows hold an extra field and a null text field
    When the caller reads the rows
    Then the extra field is ignored
    And the null text field holds the zero value
