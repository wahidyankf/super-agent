# The catalog loader reads the sources file once and returns a validated catalog, or a named error. The file is the whole
# catalog: a missing file is an error, and the loader never invents a source.
Feature: Sources catalog loading
  A caller loads the sources catalog file. The file must be a regular file of the size limit, must hold the sources of
  a registered kind, and must follow the catalog schema.
  A refused file returns an error that names the rule and never a value.

  # Behaviour: RUN-025
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A catalog file that is absent names where to set the path
    Given no catalog file at the path "/work/sources.yml"
    When the caller loads the catalog
    Then the call returns the error "not found: set --catalog or RETRIEVER_CATALOG, or create ./sources.yml" and no catalog

  # Behaviour: RUN-026
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A catalog file that is present is the whole catalog
    Given a catalog file with two outline sources "docs" and "wiki"
    When the caller loads the catalog
    Then the catalog holds the sources "docs" and "wiki" and no other

  # Behaviour: RUN-027
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A catalog path that is a symbolic link or a folder is refused
    Given the catalog path "/work/sources.yml" that is a "<kind>"
    When the caller loads the catalog
    Then the call returns the error "must be a regular file, not a symlink or a folder" and no catalog

    Examples:
      | kind         |
      | symbolic link |
      | folder       |

  # Behaviour: RUN-028
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A catalog at the size limit is read and one byte over is refused
    Given a catalog file of "<size>" bytes
    When the caller loads the catalog
    Then the result is "<result>"

    Examples:
      | size          | result                       |
      | 65536         | the catalog is loaded        |
      | 65537         | the call returns an error that states "must be at most 64 KiB" |

  # Behaviour: RUN-029
  # Exemption(integration): the rule is checked on the parsed catalog text, which the unit binding loads from an in-memory folder; the real file read is bound at integration by the absent, present, link, and size scenarios; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A catalog that still holds the removed access key is refused
    Given a catalog file with a source that holds the field "access" with the value "EXAMPLE-SECRET-VALUE"
    When the caller loads the catalog
    Then the call returns the error "unknown field \"access\": the access level was removed; remove the key and limit the source's API token instead" and the value is not in the error

  # Behaviour: RUN-030
  # Exemption(integration): the rule is checked on the parsed catalog text, which the unit binding loads from an in-memory folder; the real file read is bound at integration by the absent, present, link, and size scenarios; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A catalog that is invalid YAML, or has an unknown or repeated field
    Given a catalog file that "<problem>"
    When the caller loads the catalog
    Then the call returns the error "the file is not valid YAML, or it has an unknown or repeated field" and no catalog

    Examples:
      | problem                                   |
      | is not valid YAML                         |
      | holds an unknown field                    |
      | holds a repeated field                    |

  # Behaviour: RUN-031
  # Exemption(integration): the rule is checked on the parsed catalog text, which the unit binding loads from an in-memory folder; the real file read is bound at integration by the absent, present, link, and size scenarios; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A catalog whose version is not 1 is refused
    Given a catalog file with the version 2
    When the caller loads the catalog
    Then the call returns the error "version: must be 1" and no catalog

  # Behaviour: RUN-032
  # Exemption(integration): the rule is checked on the parsed catalog text, which the unit binding loads from an in-memory folder; the real file read is bound at integration by the absent, present, link, and size scenarios; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A catalog file that holds two YAML documents is refused
    Given a catalog file with two YAML documents
    When the caller loads the catalog
    Then the call returns the error "the file must hold one YAML document" and no catalog

  # Behaviour: RUN-033
  # Exemption(integration): the rule is checked on the parsed catalog text, which the unit binding loads from an in-memory folder; the real file read is bound at integration by the absent, present, link, and size scenarios; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A catalog path that is not absolute is refused when the loader is made
    Given the catalog path "sources.yml"
    When the caller makes the loader for the path
    Then the call returns the error "the catalog path must be absolute" and no loader

  # Behaviour: RUN-034
  # Exemption(integration): the rule is checked on the parsed catalog text, which the unit binding loads from an in-memory folder; the real file read is bound at integration by the absent, present, link, and size scenarios; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A source of a kind that the kind set does not register is refused
    Given a catalog file with one source of the kind "github" and a kind set of outline and kiwi only
    When the caller loads the catalog
    Then the call returns the unknown source kind error and no catalog

  # Behaviour: RUN-035
  # Exemption(integration): the rule is checked on the parsed catalog text, which the unit binding loads from an in-memory folder; the real file read is bound at integration by the absent, present, link, and size scenarios; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A spec pin that matches the loaded spec file loads the source
    Given a catalog with the spec kind "acme" whose source pins the spec digest of the loaded spec file
    When the caller loads the catalog with the spec digests
    Then the catalog holds the source

  # Behaviour: RUN-036
  # Exemption(integration): the rule is checked on the parsed catalog text, which the unit binding loads from an in-memory folder; the real file read is bound at integration by the absent, present, link, and size scenarios; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A spec pin that does not match the loaded spec fails with its code
    Given a catalog with the spec kind "acme" whose source pins a digest that differs from the loaded spec file
    When the caller loads the catalog with the spec digests
    Then the call fails with the code "retriever.connector.spec_pin_mismatch" and no catalog

  # Behaviour: RUN-037
  # Exemption(integration): the rule is checked on the parsed catalog text, which the unit binding loads from an in-memory folder; the real file read is bound at integration by the absent, present, link, and size scenarios; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A spec pin that is not 64 lower-case hex characters is refused
    Given a catalog with the spec kind "acme" whose source pins the value "<pin>"
    When the caller loads the catalog with the spec digests
    Then the call returns an error that states "must be 64 lower-case hex characters" and no catalog

    Examples:
      | pin                                                              |
      | 0123456789ABCDEF0123456789abcdef0123456789abcdef0123456789abcdef |
      | 0123456789abcdef                                                 |

  # Behaviour: RUN-038
  # Exemption(integration): the rule is checked on the parsed catalog text, which the unit binding loads from an in-memory folder; the real file read is bound at integration by the absent, present, link, and size scenarios; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A spec pin on a built-in kind is refused because it is not a spec kind
    Given a catalog with the outline source "docs" that pins a spec digest
    When the caller loads the catalog with the spec digests
    Then the call returns an error that states "only for a spec kind" and no catalog

  # Behaviour: RUN-039
  # Exemption(integration): the rule is checked on the parsed catalog text, which the unit binding loads from an in-memory folder; the real file read is bound at integration by the absent, present, link, and size scenarios; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A spec kind is pinned only when every source of it pins its spec
    Given a catalog with two sources of the spec kind "acme", one pinning its spec and one not
    When the caller reads the pinned kinds after the load
    Then the kind "acme" is not pinned

  # Behaviour: RUN-039
  # Exemption(integration): the rule is checked on the parsed catalog text, which the unit binding loads from an in-memory folder; the real file read is bound at integration by the absent, present, link, and size scenarios; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A spec kind with no source is not pinned
    Given a catalog with no source of the spec kind "acme"
    When the caller reads the pinned kinds after the load
    Then the kind "acme" is not pinned

  # Behaviour: RUN-040
  # Exemption(integration): the rule is checked on the parsed catalog text, which the unit binding loads from an in-memory folder; the real file read is bound at integration by the absent, present, link, and size scenarios; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A plugin block enables a plugin, and a source without one has none
    Given a catalog with the outline source "docs" that has a plugin block with the absolute path "/opt/retriever/plugins/example" and a valid pin, and the outline source "wiki" with no plugin block
    When the caller loads the catalog
    Then "docs" has a plugin and "wiki" has none

  # Behaviour: RUN-041
  # Exemption(integration): the rule is checked on the parsed catalog text, which the unit binding loads from an in-memory folder; the real file read is bound at integration by the absent, present, link, and size scenarios; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A plugin block with a relative path or a misshaped pin refuses the catalog
    Given a catalog with the outline source "docs" whose plugin block "<problem>"
    When the caller loads the catalog
    Then the call returns an error and no catalog

    Examples:
      | problem                             |
      | has the relative path "plugins/x"   |
      | has a pin of 63 hex characters      |

  # Behaviour: RUN-042
  # Exemption(integration): the rule is checked on the parsed catalog text, which the unit binding loads from an in-memory folder; the real file read is bound at integration by the absent, present, link, and size scenarios; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A profile catalog refuses a credential name without the profile prefix
    Given a profile "work" catalog with the outline source "docs" whose credential is "OUTLINE_API_KEY"
    When the caller loads the catalog for the profile "work"
    Then the call returns the error "must start with the profile prefix" and no catalog

  # Behaviour: RUN-042
  # Exemption(integration): the rule is checked on the parsed catalog text, which the unit binding loads from an in-memory folder; the real file read is bound at integration by the absent, present, link, and size scenarios; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A hyphenated profile name gives an underscore prefix that is accepted
    Given a profile "team-a" catalog with the outline source "docs" whose credential is "TEAM_A_OUTLINE_API_KEY"
    When the caller loads the catalog for the profile "team-a"
    Then the catalog holds the source "docs"

  # Behaviour: RUN-043
  # Exemption(integration): the rule is checked on the parsed catalog text, which the unit binding loads from an in-memory folder; the real file read is bound at integration by the absent, present, link, and size scenarios; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A single catalog with no profile keeps the names it was written with
    Given a catalog with the outline source "docs" whose credential is "OUTLINE_API_KEY"
    When the caller loads the catalog with no profile
    Then the catalog holds the source "docs"

  # Behaviour: RUN-044
  # Exemption(integration): the rule is checked on the parsed catalog text, which the unit binding loads from an in-memory folder; the real file read is bound at integration by the absent, present, link, and size scenarios; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A catalog holds one identity name on one host and loads
    Given a catalog with two jira sources on the host "docs.example.test" that share the identity "JIRA_EMAIL"
    When the caller loads the catalog
    Then the catalog holds the two sources

  # Behaviour: RUN-045
  # Exemption(integration): the rule is checked on the parsed catalog text, which the unit binding loads from an in-memory folder; the real file read is bound at integration by the absent, present, link, and size scenarios; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A catalog error for an unknown kind never echoes the value that was given
    Given a catalog file with the kind "EXAMPLE-SECRET-VALUE"
    When the caller loads the catalog
    Then the call returns an error and the error text does not hold "EXAMPLE-SECRET-VALUE"
