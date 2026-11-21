# Source kinds name the systems that Retriever reads. The built-in kinds are outline, kiwi, and jira. A composition root can
# register more kinds at start-up, and each kind name is checked before it is used.
Feature: Source kinds
  A caller names the kind of a source with a slug, and gets the kind's display name and identity rule.
  A failure of one kind names that kind, and never another.

  # Behaviour: CON-048
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A built-in kind name is accepted with its display name and identity rule
    Given the kind name "<kind>"
    When the caller builds a source kind from the name
    Then the kind name is "<kind>", its display name is "<display>", and its identity rule is "<identity>"

    Examples:
      | kind    | display | identity     |
      | outline | Outline | not used     |
      | kiwi    | Kiwi    | not used     |
      | jira    | Jira    | required     |

  # Behaviour: CON-049
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A name that is not a built-in kind is refused and the kinds are listed
    Given the kind name "<kind>"
    When the caller builds a source kind from the name
    Then the call returns the error "unknown source kind (known: outline, kiwi, jira)" and no kind

    Examples:
      | kind   |
      | github |
      | Kiwi   |
      | kiwi-2 |
      |        |

  # Behaviour: CON-050
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A registered kind takes its name as its display name when it has none
    Given a kind set built from the definition "acme-wiki" with no display name
    When the caller asks the set for the kind "acme-wiki"
    Then the kind is found, its name is "acme-wiki", and its display name is "acme-wiki"

  # Behaviour: CON-050
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A registered kind keeps the display name it was given
    Given a kind set built from the definition "acme-wiki" with the display name "Acme Wiki"
    When the caller asks the set for the kind "acme-wiki"
    Then the kind is found, and its display name is "Acme Wiki"

  # Behaviour: CON-051
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A kind set refuses a kind name that breaks the slug rule or is used twice
    Given a kind set built from the names "<names>"
    When the caller builds the kind set
    Then the call returns the error "<message>" and no kind set

    Examples:
      | names             | message                                                                |
      | Acme              | invalid kind name: use a lowercase slug of at most 32 characters      |
      | 1wiki             | invalid kind name: use a lowercase slug of at most 32 characters      |
      | acme_wiki         | invalid kind name: use a lowercase slug of at most 32 characters      |
      | acme-wiki, acme-wiki | duplicate kind name                                                |

  # Behaviour: CON-051
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A kind set refuses an empty list of definitions
    Given a kind set built from no definitions
    When the caller builds the kind set
    Then the call returns the error "a kind set needs at least one kind" and no kind set

  # Behaviour: CON-051
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A kind name of 33 characters is refused
    Given a kind set built from a name of 33 lowercase letter a characters
    When the caller builds the kind set
    Then the call returns the error "invalid kind name: use a lowercase slug of at most 32 characters" and no kind set

  # Behaviour: CON-052
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A kind set finds a registered kind by name
    Given a kind set with the kinds "kiwi" and "acme-wiki"
    When the caller asks the set for the kind "kiwi"
    Then the kind is found, and its name is "kiwi"

  # Behaviour: CON-052
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A kind set names the registered kinds when it does not know a name
    Given a kind set with the kinds "kiwi" and "acme-wiki"
    When the caller asks the set for the kind "jira"
    Then the call returns the error "unknown source kind (known: acme-wiki, kiwi)" and no kind

  # Behaviour: CON-053
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A failure of a kind never shows the word Outline for another kind
    Given a failure of the kind "<kind>" whose text is "<failure>"
    When the caller reads the text of the failure
    Then the text is "<shown>"

    Examples:
      | kind    | failure                    | shown                      |
      | kiwi    | Outline rejected the API key | Kiwi rejected the API key |
      | jira    | Outline denied access      | Jira denied access         |
      | outline | Outline denied access      | Outline denied access      |
