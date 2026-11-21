Feature: Kiwi document text
  Kiwi escapes its text once, so a caller receives the text unescaped once. Control characters are removed, except the tab
  and the line feed. A case document shows its metadata and its text, and it shows the legacy fields as sections only when
  Kiwi returns them. A plan document lists its cases one line each, in the order given, and a plan with no cases says so.

  # Behaviour: KIW-001
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour renders decoded rows as Markdown and touches no connection or file; alternative-proof: retriever-connector-kiwi:test:unit
  @integration-exempt
  Scenario: Kiwi text is unescaped once and loses its control characters, except tab and line feed
    Given a Kiwi text that holds an escaped character and a control character
    When the caller reads the text
    Then the text is unescaped once
    And the control character is removed
    And the tab and the line feed are kept

  # Behaviour: KIW-002
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour renders decoded rows as Markdown and touches no connection or file; alternative-proof: retriever-connector-kiwi:test:unit
  @integration-exempt
  Scenario: A case document holds its metadata and its text
    Given a case with its metadata and its text
    When the caller reads the document of the case
    Then the document holds the metadata and the text of the case

  # Behaviour: KIW-003
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour renders decoded rows as Markdown and touches no connection or file; alternative-proof: retriever-connector-kiwi:test:unit
  @integration-exempt
  Scenario: A case with legacy fields shows each legacy field as a section, in order
    Given a case that Kiwi returns with its legacy fields
    When the caller reads the document of the case
    Then the legacy fields appear as four sections in order

  # Behaviour: KIW-004
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour renders decoded rows as Markdown and touches no connection or file; alternative-proof: retriever-connector-kiwi:test:unit
  @integration-exempt
  Scenario: A case without legacy fields shows no legacy sections
    Given a case that Kiwi returns without legacy fields
    When the caller reads the document of the case
    Then the document has no legacy sections

  # Behaviour: KIW-005
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour renders decoded rows as Markdown and touches no connection or file; alternative-proof: retriever-connector-kiwi:test:unit
  @integration-exempt
  Scenario: A plan document holds its metadata and text and one line for each case, in the order given
    Given a plan with metadata, text, and two cases
    When the caller reads the document of the plan
    Then the document holds the metadata and text of the plan
    And it has one line for each case in the order given

  # Behaviour: KIW-006
  # Exemption(e2e): the library is built into the retriever binary and has no process of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour renders decoded rows as Markdown and touches no connection or file; alternative-proof: retriever-connector-kiwi:test:unit
  @integration-exempt
  Scenario: A plan with no cases and no parent or text says so
    Given a plan with no cases and no parent and no text
    When the caller reads the document of the plan
    Then the document says that the plan has no cases
