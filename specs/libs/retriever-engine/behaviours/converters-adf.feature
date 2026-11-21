Feature: Atlassian Document Format to Markdown
  A field that names the markdown-from-adf converter maps an ADF document to Markdown, on a read and on a write. The
  rendering keeps the block and inline nodes on its allow-list, escapes every text, and stops at a depth limit, a node
  limit, or an output cap, and it says so when it stops. Input that is not an ADF document fails the read.

  # Behaviour: ENG-017
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: An ADF read maps the document to Markdown in a plain field and in a nested record
    Given a validated read whose field names markdown-from-adf and a reply that holds an ADF document
    When the caller runs the read
    Then the field holds the document as Markdown

  # Behaviour: ENG-018
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: An ADF write maps the reply document to Markdown
    Given a validated write whose reply field names markdown-from-adf
    When the caller runs the write
    Then the field holds the reply document as Markdown

  # Behaviour: ENG-019
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: An ADF field missing from the reply maps to null like any other optional field
    Given a validated read whose optional ADF field is missing from the reply
    When the caller runs the read
    Then the field is null

  # Behaviour: ENG-020
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A reply value that is not an ADF document fails the read without naming the value
    Given a validated read whose ADF field holds a value that is not an ADF document
    When the caller runs the read
    Then the read fails with an unexpected response
    And the error does not name the value

  # Behaviour: ENG-021
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario Outline: Each block node on the allow-list renders as its Markdown form
    Given an ADF document holding one <node> node
    When the caller renders the document
    Then the output holds the Markdown form of the <node> node

    Examples:
      | node             |
      | paragraph        |
      | heading          |
      | bullet list      |
      | ordered list     |
      | code block       |
      | blockquote       |
      | rule             |

  # Behaviour: ENG-022
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario Outline: Each inline node and mark on the allow-list renders as its Markdown form
    Given an ADF document holding one <inline> inline node or mark
    When the caller renders the document
    Then the output holds the Markdown form of the <inline>

    Examples:
      | inline        |
      | strong mark   |
      | em mark       |
      | strike mark   |
      | link mark     |
      | mention       |
      | emoji         |
      | hardBreak     |
      | status        |
      | date          |

  # Behaviour: ENG-023
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: Text with markup in an ADF document is escaped and cleaned in the output
    Given an ADF document whose text nodes hold markup and control characters
    When the caller renders the document
    Then every text is escaped
    And every control character is removed

  # Behaviour: ENG-024
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: Input that is not JSON, or is JSON that is not a node, fails as an unexpected response
    Given an input that is not JSON or that is a JSON value that is not a document node
    When the caller renders the input
    Then the rendering fails with an unexpected response

  # Behaviour: ENG-025
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A tree deeper than the depth limit renders what fits and ends with the truncation marker
    Given an ADF document nested deeper than the depth limit
    When the caller renders the document
    Then the output ends with the truncation marker
    And the text below the limit is not shown

  # Behaviour: ENG-026
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: More nodes than the node limit render what fits and say so
    Given an ADF document with more nodes than the node limit
    When the caller renders the document
    Then the output holds the nodes that fit
    And the output says that it stopped

  # Behaviour: ENG-027
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A text longer than the output cap is cut at a whole character below the cap with the marker
    Given an ADF document whose text is longer than the output cap of one MiB
    When the caller renders the document
    Then the output is cut at a whole character below the cap
    And the output ends with the marker

  # Behaviour: ENG-028
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A code block that does not fit the output is dropped whole and never written half-open
    Given an ADF document whose code block does not fit the remaining output
    When the caller renders the document
    Then the code block is absent from the output
    And no half-open code fence is written
