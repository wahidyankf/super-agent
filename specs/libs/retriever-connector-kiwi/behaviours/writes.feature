Feature: Kiwi writes
  A caller creates a case, updates a case, or adds a case to a plan. Names in a write are resolved to Kiwi IDs before the
  write is sent, and a name that matches no row or more than one row stops the write. A write sends one request for the
  change. Every failure that happens after the request leaves is an unknown outcome, and the write is never sent again.
  A failure before the write sends nothing.

  # Behaviour: KIW-066
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: Creating a case resolves its names to IDs and sends one create
    Given a scripted Kiwi gateway that holds the names of a product, a category, a status, and a priority
    When the caller creates a case with those names
    Then the create carries the IDs of those names
    And exactly one create request is sent

  # Behaviour: KIW-067
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: Creating a case sends the optional status and text when the caller gives them
    Given a scripted Kiwi gateway and a create that names a status and gives a text
    When the caller creates the case
    Then the create carries the status ID and the text

  # Behaviour: KIW-068
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A create whose name matches <matches> sends no write
    Given a scripted Kiwi gateway in which a name of the create matches <matches>
    When the caller creates the case
    Then the create is refused and no write is sent

    Examples:
      | matches     |
      | no row      |
      | several rows |

  # Behaviour: KIW-069
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: Updating a case sends only the field the caller names
    Given a scripted Kiwi gateway that holds a case
    When the caller updates the summary of the case
    Then the case is checked first
    And the update carries only the summary

  # Behaviour: KIW-070
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: Updating a case with a status and a priority resolves both names to IDs
    Given a scripted Kiwi gateway that holds a case, a status, and a priority
    When the caller updates the status and the priority of the case
    Then the update carries the status ID and the priority ID

  # Behaviour: KIW-071
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: Updating a case that does not exist sends no update
    Given a scripted Kiwi gateway that holds no case with the asked ID
    When the caller updates that case
    Then the update fails before the update request and no update is sent

  # Behaviour: KIW-072
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: Adding a case to a plan sends one add that links the plan and the case
    Given a scripted Kiwi gateway that holds a plan and a case
    When the caller adds the case to the plan
    Then exactly one add request is sent
    And the add carries the plan ID and the case ID

  # Behaviour: KIW-073
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: Adding a case to a plan that does not exist is named as not found
    Given a scripted Kiwi gateway that holds no plan with the asked ID
    When the caller adds a case to that plan
    Then the add fails as not found

  # Behaviour: KIW-074
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the refusal happens before any request is built, so no connection or file is reached; alternative-proof: retriever-connector-kiwi:test:unit
  @integration-exempt
  Scenario: A write refuses a read operation and sends no request
    Given a write of an operation that is a read
    When the caller runs the write
    Then the write is refused and no request is sent

  # Behaviour: KIW-075
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the refusal happens before any request is built, so no connection or file is reached; alternative-proof: retriever-connector-kiwi:test:unit
  @integration-exempt
  Scenario: A write of an operation outside the catalog is unsupported
    Given a write of an operation that the catalog does not hold
    When the caller runs the write
    Then the write is unsupported and no request is sent

  # Behaviour: KIW-076
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the refusal happens before any request is built, so no connection or file is reached; alternative-proof: retriever-connector-kiwi:test:unit
  @integration-exempt
  Scenario: A write whose product ID has the wrong shape is refused before any lookup is sent
    Given a create that names a product ID in the wrong shape
    When the caller creates the case
    Then the create is refused and no lookup is sent

  # Behaviour: KIW-077
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the refusal happens before any request is built, so no connection or file is reached; alternative-proof: retriever-connector-kiwi:test:unit
  @integration-exempt
  Scenario: An update that names a plan ID is refused before any request
    Given an update that names a plan ID where a case ID is needed
    When the caller runs the update
    Then the update is refused and no request is sent

  # Behaviour: KIW-078
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the refusal happens before any request is built, so no connection or file is reached; alternative-proof: retriever-connector-kiwi:test:unit
  @integration-exempt
  Scenario: A write of another kind's operation is not sent to Kiwi
    Given a valid write of an operation that belongs to another connector
    When the caller runs the write
    Then the write is unsupported and no request is sent

  # Behaviour: KIW-079
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A create whose name lookup fails stops before the write is sent
    Given a scripted Kiwi gateway whose name lookup of a create fails
    When the caller creates the case
    Then the create returns the lookup fault
    And no create request is sent

  # Behaviour: KIW-080
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: An update whose <step> fails stops before the write is sent
    Given a scripted Kiwi gateway whose <step> of an update fails
    When the caller updates the case
    Then the update returns the fault of that step
    And no update request is sent

    Examples:
      | step                       |
      | case lookup                |
      | status or priority lookup  |

  # Behaviour: KIW-081
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A create whose reply does not name the new case is an unknown outcome and is not sent again
    Given a scripted Kiwi gateway whose create reply does not name the new case
    When the caller creates the case
    Then the create reports an unknown outcome
    And exactly one create request is sent

  # Behaviour: KIW-082
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: An add-case whose exchange fails is an unknown outcome and is not sent again
    Given a scripted Kiwi gateway whose add-case exchange fails after the request left
    When the caller adds the case to the plan
    Then the add reports an unknown outcome
    And exactly one add request is sent

  # Behaviour: KIW-083
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A write whose body breaks after Kiwi answered its status is an unknown outcome
    Given a scripted Kiwi gateway that answers a write with a status and then breaks the reply body
    When the caller runs the write
    Then the write reports an unknown outcome
    And exactly one request is sent
