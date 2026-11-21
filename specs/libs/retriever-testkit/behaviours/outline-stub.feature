# The Outline stub is a fake Outline API for tests. It replays recorded replies, answers scripted faults, and records each
# request. A test uses it to check what the Outline client sends and how it handles each reply.
Feature: Outline API stub
  A test starts the Outline stub and calls it like the Outline API.
  The stub answers the methods it offers, refuses every other call, and keeps the replies that a test sets.

  # Behaviour: TK-001
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: The collections list answers one page and names the next page
    Given the Outline stub with its collections
    When the client calls the collections list method
    Then the stub answers the collections of the page and echoes the offset and limit of the page

  # Behaviour: TK-002
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario Outline: The collections info method answers one collection or not found
    Given the Outline stub with its collections
    When the client calls the collections info method with the ID "<id>"
    Then the stub answers with the status "<status>"

    Examples:
      | id | status |
      | c1 | 200    |
      | c9 | 404    |

  # Behaviour: TK-003
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: The collections documents method nests each child under its parent
    Given the Outline stub with a collection holding a document and a child document
    When the client calls the collections documents method for the collection
    Then the child document is nested under its parent document

  # Behaviour: TK-004
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: The documents info method answers the shape the header asks for
    Given the Outline stub with the document "doc-1"
    When the client calls the documents info method for "doc-1" with the version header
    Then the stub answers the document in the shape the header asks for

  # Behaviour: TK-005
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario Outline: The documents export method answers one document's Markdown or 404
    Given the Outline stub with the document "doc-1" that has the Markdown "# Notes"
    When the client calls the documents export method with the ID "<id>"
    Then the stub answers with the status "<status>"

    Examples:
      | id    | status |
      | doc-1 | 200    |
      | doc-9 | 404    |

  # Behaviour: TK-006
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: The documents list method answers one page and names the next
    Given the Outline stub with more documents than one page holds
    When the client calls the documents list method
    Then the stub answers one page of documents and names the next page

  # Behaviour: TK-007
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario Outline: The documents search method finds a term in a title or body
    Given the Outline stub with the documents "Install Guide" and "Notes" whose bodies hold "Install"
    When the client searches for "<term>"
    Then the stub answers the documents whose title or body holds the term

    Examples:
      | term    |
      | install |
      | INSTALL |

  # Behaviour: TK-008
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: The documents search titles method ignores words found only in bodies
    Given the Outline stub with the document "Notes" whose body holds "Install" and no title holds it
    When the client searches the titles for "install"
    Then the stub answers no document

  # Behaviour: TK-009
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario Outline: Only a POST reaches a replay and any other verb is refused
    Given the Outline stub with its replies
    When the client sends a "<verb>" request to the collections list method
    Then the stub answers with the status "<status>"

    Examples:
      | verb | status |
      | GET  | 405    |
      | PUT  | 405    |

  # Behaviour: TK-010
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A method the stub does not offer gets the endpoint not found message
    Given the Outline stub with its replies
    When the client calls the method "users.list" that the stub does not offer
    Then the stub answers with the status 404 and the text "Endpoint not found"

  # Behaviour: TK-011
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A set reply answers with its status and body
    Given the Outline stub with the reply of the collections info method set to the status 500 and the body "{\"ok\":false}"
    When the client calls the collections info method with the ID "anything"
    Then the stub answers with the status 500

  # Behaviour: TK-012
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A fault for one method answers only calls of that method
    Given the Outline stub with a fault of the status 500 for the documents info method
    When the client calls the documents info method
    Then the stub answers with the status 500

  # Behaviour: TK-013
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: The reply key check refuses a key that the router does not send
    Given a reply of the collections list method that holds the key "fabricatedKey"
    When the stub checks the keys of the reply against the router keys
    Then the check refuses the reply

  # Behaviour: TK-013
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: The reply key check accepts the keys that the presenters send
    Given a reply of the collections list method that holds only the keys the presenters send
    When the stub checks the keys of the reply against the router keys
    Then the check accepts the reply

  # Behaviour: TK-014
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: The CA file of the stub is written with owner-only access
    Given the Outline stub started over TLS
    When the caller writes the certificate of the stub to a file
    Then the file holds the PEM certificate and the file is readable by its owner only
