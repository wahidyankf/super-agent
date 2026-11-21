Feature: Parameters, query keys, built keys, and converters
  A read takes typed parameters from the operation catalog. A query parameter may travel under a wire key of its own,
  which the spec may set only for a query parameter. A reply field may build a key from an issue key, a project key, a
  collection key, or a browse address, and a built field reads one key only. Each converter and each parameter kind
  is a name from a closed list.

  # Behaviour: SPC-031
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A query parameter with a wire key of its own is accepted
    Given a connector spec whose query parameter travels under a different wire key
    When the caller validates the spec document
    Then the spec is accepted

  # Behaviour: SPC-031
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A wire key that a path or body parameter cannot take is refused
    Given a connector spec whose <parameter> sets a wire key
    When the caller validates the spec document
    Then the spec is refused

    Examples:
      | parameter                          |
      | path parameter                     |
      | body parameter                     |

  # Behaviour: SPC-031
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A wire key that repeats a name or another query key of the same operation is refused
    Given a connector spec whose query parameter sets a wire key that <clash>
    When the caller validates the spec document
    Then the spec is refused

    Examples:
      | clash                                         |
      | repeats the parameter name                    |
      | is the same as another query key              |

  # Behaviour: SPC-032
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A built key field that names one source pointer is accepted
    Given a connector spec whose list read builds a project key from one issue key pointer
    When the caller validates the spec document
    Then the spec is accepted

  # Behaviour: SPC-032
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A field that holds a pointer and also builds a key is refused at the field
    Given a connector spec whose project key field names a pointer and also builds a key
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/0/response/fields/projectKey

  # Behaviour: SPC-032
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A built key field that also sets a format is refused at the field
    Given a connector spec whose built project key field also sets a format
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/0/response/fields/projectKey

  # Behaviour: SPC-033
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A reply field may name each closed converter
    Given a connector spec whose reply field uses the converter <converter>
    When the caller validates the spec document
    Then the spec is accepted

    Examples:
      | converter            |
      | plain                |
      | markdown-from-adf    |
      | hostname             |
      | text-from-editor-data |
      | jira-time            |
      | jira-date            |
      | jira-title           |

  # Behaviour: SPC-033
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A reply field that names a converter outside the closed list is refused at the converter
    Given a connector spec whose reply field names a converter that the spec does not know
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/1/response/fields/body/convert

  # Behaviour: SPC-033
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: An issue-key parameter of the issue key type is accepted
    Given a connector spec whose read takes a parameter of the kind IssueKey
    When the caller validates the spec document
    Then the spec is accepted

  # Behaviour: SPC-048
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A parameter of a catalog kind is accepted, including icon, URL list, and event list kinds
    Given a connector spec whose read takes a parameter of the kind <kind>
    When the caller validates the spec document
    Then the spec is accepted

    Examples:
      | kind      |
      | ID        |
      | Limit     |
      | Enum      |
      | Icon      |
      | URLList   |
      | EventList |

  # Behaviour: SPC-048
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: An enum parameter with no closed values is refused at its values
    Given a connector spec whose first operation adds an enum parameter with no values after its two parameters
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/0/params/2/values
