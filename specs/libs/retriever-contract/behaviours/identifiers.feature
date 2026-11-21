# Identifiers are the names a caller passes to the library: collection IDs, document IDs, source IDs, tags, UUIDs, and
# share IDs. Each one has a fixed grammar, and text outside it is refused before it reaches any source.
Feature: Identifiers
  A caller builds a validated identifier from text, or checks text against an identifier grammar.
  A refused text returns an error that names the kind of identifier.

  # Behaviour: CON-002
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A collection or document ID of letters, digits, _ and - is accepted
    Given the text "<id>"
    When the caller builds a <kind> ID from the text
    Then the ID is built and its text is "<id>"

    Examples:
      | kind       | id       |
      | collection | col-tree |
      | collection | col_2    |
      | collection | A1-b_2   |
      | document   | doc-1    |
      | document   | Doc_9    |

  # Behaviour: CON-003
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A collection or document ID with another character is refused
    Given the text "<id>"
    When the caller builds a <kind> ID from the text
    Then the call returns an invalid <kind> ID error and no ID

    Examples:
      | kind       | id       |
      | collection | col tree |
      | collection | col/tree |
      | collection | col.tree |
      | document   | doc 1    |
      | document   | doc#1    |
      | document   |          |

  # Behaviour: CON-004
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: An ID of exactly 255 characters passes the ID check
    Given an ID made of 255 letter a characters
    When the caller validates the ID
    Then the check passes

  # Behaviour: CON-004
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: An ID of 256 characters fails the ID check
    Given an ID made of 256 letter a characters
    When the caller validates the ID
    Then the check returns the invalid ID error

  # Behaviour: CON-005
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A source ID that is a lowercase slug is accepted
    Given the text "<source>"
    When the caller builds a source ID from the text
    Then the source ID is built and its text is "<source>"

    Examples:
      | source    |
      | outline   |
      | kiwi      |
      | jira      |
      | a1-b2     |
      | team-2-eu |

  # Behaviour: CON-006
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A source ID that is not a lowercase slug is refused
    Given the text "<source>"
    When the caller builds a source ID from the text
    Then the call returns the invalid source ID error and no source ID

    Examples:
      | source   |
      | Outline  |
      | a--b     |
      | 1outline |
      | outline- |
      | out line |
      | a/b      |
      |          |

  # Behaviour: CON-007
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A source ID of exactly 32 characters is accepted
    Given a source ID made of 32 letter a characters
    When the caller builds a source ID from the text
    Then the source ID is built and its text is the 32 letter a characters

  # Behaviour: CON-007
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A source ID of 33 characters is refused
    Given a source ID made of 33 letter a characters
    When the caller builds a source ID from the text
    Then the call returns the invalid source ID error and no source ID

  # Behaviour: CON-008
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A tag that is a lowercase slug is accepted
    Given the text "<tag>"
    When the caller builds a tag from the text
    Then the tag is built and its text is "<tag>"

    Examples:
      | tag     |
      | wiki    |
      | 2026    |
      | team-a  |
      | tests-2 |

  # Behaviour: CON-009
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A tag with an uppercase letter, double hyphen, or symbol is refused
    Given the text "<tag>"
    When the caller builds a tag from the text
    Then the call returns the invalid tag error and no tag

    Examples:
      | tag     |
      | Wiki    |
      | team--a |
      | team_a  |
      | -team   |
      | team a  |
      |         |

  # Behaviour: CON-010
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A tag of 32 characters is accepted
    Given a tag made of 32 letter a characters
    When the caller builds a tag from the text
    Then the tag is built and its text is the 32 letter a characters

  # Behaviour: CON-010
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A tag of 33 characters is refused
    Given a tag made of 33 letter a characters
    When the caller builds a tag from the text
    Then the call returns the invalid tag error and no tag

  # Behaviour: CON-011
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A tag set drops a repeated tag and keeps the rest in sorted order
    Given the tags "wiki", "tests", and "wiki" in that order
    When the caller builds a tag set from them
    Then the tag set is built and its values are "tests" and "wiki"

  # Behaviour: CON-012
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A tag set of sixteen distinct tags is accepted
    Given sixteen distinct tags named t01 to t16
    When the caller builds a tag set from them
    Then the tag set is built and holds sixteen tags

  # Behaviour: CON-012
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A tag set of seventeen distinct tags is refused
    Given seventeen distinct tags named t01 to t17
    When the caller builds a tag set from them
    Then the call returns the invalid tag error and no tag set

  # Behaviour: CON-013
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A tag set contains another only when it holds every tag of that set
    Given a tag set holding the tags "wiki" and "tests"
    When the caller asks whether the set contains the tags "<wanted>"
    Then the answer is <answer>

    Examples:
      | wanted      | answer |
      | wiki        | true   |
      | wiki, tests | true   |
      | wiki, kiwi  | false  |
      | jira        | false  |

  # Behaviour: CON-013
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: An empty tag set is contained by every tag set
    Given a tag set holding the tags "wiki" and "tests"
    When the caller asks whether the set contains an empty tag set
    Then the answer is true

  # Behaviour: CON-014
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A canonical UUID with hyphens is accepted in either letter case
    Given the text "<uuid>"
    When the caller validates the text as a UUID
    Then the check passes

    Examples:
      | uuid                                 |
      | 550e8400-e29b-41d4-a716-446655440000 |
      | 550E8400-E29B-41D4-A716-446655440000 |
      | 550e8400-E29B-41d4-a716-446655440000 |

  # Behaviour: CON-015
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: Text that is not a canonical UUID is refused as an invalid UUID
    Given the text "<uuid>"
    When the caller validates the text as a UUID
    Then the check returns the invalid UUID error

    Examples:
      | uuid                                |
      | 550e8400e29b41d4a716446655440000    |
      | 550e8400-e29b-41d4-a716-44665544000 |
      | urlslug                             |
      |                                     |

  # Behaviour: CON-016
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A share ID that is a UUID or a lowercase slug is accepted
    Given the text "<share>"
    When the caller validates the text as a share ID
    Then the check passes

    Examples:
      | share                                |
      | 550e8400-e29b-41d4-a716-446655440000 |
      | public-wiki-2                        |
      | a                                    |

  # Behaviour: CON-017
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A share ID that is neither a UUID nor a lowercase slug is refused
    Given the text "<share>"
    When the caller validates the text as a share ID
    Then the check returns the invalid share ID error

    Examples:
      | share    |
      | Share_1  |
      | share id |
      | SHARE    |
      |          |
