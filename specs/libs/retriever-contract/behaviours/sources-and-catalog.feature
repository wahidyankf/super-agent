# The catalog is the set of sources a run may read. It holds one to sixty-four sources with unique IDs, and it refuses a
# source whose kind rules, credential, or identity do not hold across the catalog.
Feature: Sources and catalog
  A caller builds a catalog from sources, then finds a source by its ID, by its filter, or by its kind.
  A refused catalog names the position and the field of the source that broke the rule.

  # Behaviour: CON-067
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A catalog built from no sources is refused
    Given an empty list of sources
    When the caller builds the catalog
    Then the call returns the error "the catalog needs at least one source" and no catalog

  # Behaviour: CON-068
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A catalog of 64 sources is accepted
    Given 64 valid outline sources with distinct IDs and distinct credentials
    When the caller builds the catalog
    Then the catalog is built and holds 64 sources

  # Behaviour: CON-068
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A catalog of 65 sources is refused
    Given 65 valid outline sources with distinct IDs and distinct credentials
    When the caller builds the catalog
    Then the call returns the error "the catalog holds at most 64 sources" and no catalog

  # Behaviour: CON-069
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A source built with no ID is refused with its position and field
    Given a list of one source that has no ID
    When the caller builds the catalog
    Then the call returns the error "source 1 id: invalid source ID" and no catalog

  # Behaviour: CON-070
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: Two sources with one ID are refused with the position of the second
    Given two valid outline sources that both have the ID "wiki"
    When the caller builds the catalog
    Then the call returns the error "source 2 id: duplicate" and no catalog

  # Behaviour: CON-071
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A jira source with no identity name is refused
    Given one valid jira source with no identity name
    When the caller builds the catalog
    Then the call returns the error "source 1 identity_env: required for a jira source" and no catalog

  # Behaviour: CON-072
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: An outline source that names an identity is refused
    Given one valid outline source that names the identity "OUTLINE_EMAIL"
    When the caller builds the catalog
    Then the call returns the error "source 1 identity_env: not used by an outline source" and no catalog

  # Behaviour: CON-073
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: Two sources of one host may share a credential name
    Given two valid outline sources on the host "docs.example.test" that share the credential "OUTLINE_API_KEY"
    When the caller builds the catalog
    Then the catalog is built and holds 2 sources

  # Behaviour: CON-073
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A credential name that serves two hosts is refused at the second source
    Given a valid outline source on "docs.example.test" and a second on "wiki.example.test", both with the credential "OUTLINE_API_KEY"
    When the caller builds the catalog
    Then the call returns the error "source 2 credential_env: one credential serves one host only" and no catalog

  # Behaviour: CON-074
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: An identity name that serves two hosts is refused at the second source
    Given two valid jira sources with the identity "JIRA_EMAIL", on the hosts "a.example.test" and "b.example.test"
    When the caller builds the catalog
    Then the call returns the error "source 2 identity_env: one identity serves one host only" and no catalog

  # Behaviour: CON-075
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: The catalog lists its sources sorted by ID
    Given a catalog of the outline sources "wiki", "docs", and "team"
    When the caller lists the sources of the catalog
    Then the sources are listed in the order "docs", "team", "wiki"

  # Behaviour: CON-076
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A source is found by its ID only when the catalog holds it
    Given a catalog of the outline sources "docs" and "wiki"
    When the caller finds the source with the ID "<id>"
    Then the result is "<result>"

    Examples:
      | id   | result    |
      | docs | found     |
      | team | not found |

  # Behaviour: CON-077
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario Outline: A filter selects the sources of its kind that carry all of its tags
    Given a catalog of the sources "a" (outline, tags "wiki" and "tests"), "b" (outline, tag "wiki"), and "c" (kiwi, tags "wiki" and "tests")
    When the caller finds the sources with the filter "<filter>"
    Then the sources found are "<found>"

    Examples:
      | filter                            | found   |
      | no kind and no tags               | a, b, c |
      | kind outline                      | a, b    |
      | tag wiki                          | a, b, c |
      | tags wiki and tests               | a, c    |
      | kind kiwi and tags wiki and tests | c       |
      | kind jira                         | none    |

  # Behaviour: CON-078
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A filter with a kind that is not known is refused
    Given the filter kind "github"
    When the caller builds a source filter from the kind and no tags
    Then the call returns the error "unknown source kind (known: outline, kiwi, jira)" and no filter

  # Behaviour: CON-078
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A filter with an invalid tag is refused
    Given no filter kind and the tag "Wiki"
    When the caller builds a source filter from the tag
    Then the call returns the error "invalid tag" and no filter

  # Behaviour: CON-079
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A source named by ID is selected when it exists and has the kind asked for
    Given a catalog of the outline source "docs" and the kiwi source "tests"
    When the caller selects a source of the kind "outline" with the ID "docs"
    Then the source "docs" is selected

  # Behaviour: CON-079
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A named source that the catalog does not hold is refused
    Given a catalog of the outline source "docs"
    When the caller selects a source of the kind "outline" with the ID "team"
    Then the call returns the unknown source error for "team" and no source

  # Behaviour: CON-079
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A named source of another kind is refused
    Given a catalog of the outline source "docs" and the kiwi source "tests"
    When the caller selects a source of the kind "outline" with the ID "tests"
    Then the call returns the error "the source is of another kind" and no source

  # Behaviour: CON-080
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: With no ID, the only source of the kind is selected
    Given a catalog of the outline source "docs" and the kiwi source "tests"
    When the caller selects a source of the kind "kiwi" with no ID
    Then the source "tests" is selected

  # Behaviour: CON-080
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: With no ID, a kind with no source is refused
    Given a catalog of the outline source "docs"
    When the caller selects a source of the kind "kiwi" with no ID
    Then the call returns the error "no kiwi source is configured" and no source

  # Behaviour: CON-080
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: With no ID, a kind with two sources is refused and both IDs are listed
    Given a catalog of the outline sources "wiki" and "docs"
    When the caller selects a source of the kind "outline" with no ID
    Then the call returns the error "more than one outline source exists: docs, wiki" and no source

  # Behaviour: CON-081
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A source summary shows the ID, kind, and tags, and no endpoint
    Given one valid outline source "docs" with the tags "wiki" and "tests"
    When the caller reads the summary of the source
    Then the summary holds the ID "docs", the kind "outline", and the tags "tests" and "wiki" and nothing else

  # Behaviour: CON-082
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A source given a plugin reports the pinned plugin and no other source does
    Given one valid outline source "docs" with the plugin path "/opt/retriever/plugins/example" and its pin
    When the caller reads whether the source has a plugin
    Then the source has a plugin and its plugin path is "/opt/retriever/plugins/example"

  # Behaviour: CON-082
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-contract:test:unit
  @integration-exempt
  Scenario: A source with no plugin reports none
    Given one valid outline source "docs" with no plugin
    When the caller reads whether the source has a plugin
    Then the source has no plugin
