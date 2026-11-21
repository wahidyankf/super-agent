Feature: Settings a user writes in a connector spec
  A user sets the endpoint, the authentication, the limits, and the command-line and MCP names of each operation in the
  spec document. The spec names variables and never values. A limit may only lower a built-in cap, and a spec cannot
  enable a plugin.

  # Behaviour: SPC-042
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A base path or host suffix that does not fit its form is refused at that setting
    Given a connector spec whose endpoint sets <setting> to <value>
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path <path>

    Examples:
      | setting                    | value                       | path                    |
      | the base path             | v1, with no leading slash   | /endpoint/base_path     |
      | the base path             | an https address            | /endpoint/base_path     |
      | the first host suffix     | example.test, with no dot   | /endpoint/host_suffix/0 |

  # Behaviour: SPC-042
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: An endpoint with a base path and a dotted host suffix is accepted
    Given the bundled example connector spec
    When the caller validates the spec document
    Then the spec is accepted

  # Behaviour: SPC-043
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: An auth type is accepted with exactly the variable names that it needs
    Given a connector spec whose auth is <auth>
    When the caller validates the spec document
    Then the spec is accepted

    Examples:
      | auth                                                              |
      | bearer with the name of the key variable                          |
      | basic with the names of the key variable and the identity variable |
      | header with the name of the key variable and a header name       |

  # Behaviour: SPC-043
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: An auth type that lacks or adds a variable name it does not take is refused at the auth
    Given a connector spec whose auth is <auth>
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path <path>

    Examples:
      | auth                                            | path               |
      | basic with no identity variable                 | /auth/identity_env |
      | bearer that also names an identity variable     | /auth              |
      | header with no header name                      | /auth/header_name  |

  # Behaviour: SPC-043
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A spec that names a plugin is refused at the plugin field
    Given a user connector spec that adds a plugin block with a path
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /plugin

  # Behaviour: SPC-044
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A limit at or below its built-in cap is accepted
    Given a connector spec whose limits set <limit> to <value>
    When the caller validates the spec document
    Then the spec is accepted

    Examples:
      | limit               | value    |
      | the timeout         | 30       |
      | the reply size      | 10485760 |

  # Behaviour: SPC-044
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A limit above its built-in cap is refused at that limit
    Given a connector spec whose limits set <limit> to <value>
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path <path>

    Examples:
      | limit               | value    | path                         |
      | the timeout         | 31       | /limits/timeout_seconds      |
      | the reply size      | 10485761 | /limits/max_reply_bytes      |
      | the page cap        | 21       | /limits/max_pages            |

  # Behaviour: SPC-047
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: An operation that turns off its command-line surface is accepted
    Given a connector spec whose first operation sets its command-line surface to false
    When the caller validates the spec document
    Then the spec is accepted

  # Behaviour: SPC-047
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: An operation that names its command-line and MCP surfaces is accepted
    Given a connector spec whose first operation names the command find-docs and the MCP tool find-docs
    When the caller validates the spec document
    Then the spec is accepted

  # Behaviour: SPC-047
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A surface name that is not a lower-case slug is refused at the surface
    Given a connector spec whose first operation names its <surface> surface "<name>"
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path <path>

    Examples:
      | surface | name      | path                |
      | command | Not A Slug | /operations/0/cli   |
      | MCP     | Find Docs  | /operations/0/mcp   |

  # Behaviour: SPC-047
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: An operation that omits both surface keys shows both surfaces on in the review
    Given the bundled example connector spec, whose operations name neither surface
    When the caller reviews the valid spec
    Then the review has the line "1. example-wiki.docs.list: read, GET /v1/docs, writes no, cli on, mcp on"
