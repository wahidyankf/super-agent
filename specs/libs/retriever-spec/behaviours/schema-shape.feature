Feature: The shape a connector spec must have
  After the YAML reader accepts a document, the JSON Schema of version 1 checks its shape: which fields exist, which
  values are closed lists, and which numbers stay within their caps. A document that breaks the schema is refused at
  each broken field, and the semantic rules do not run on it. The caller can also read the schema itself.

  # Behaviour: SPC-011
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A document with a wrong or unknown top-level or operation field is refused at that field
    Given a connector spec in which <change>
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path <path>

    Examples:
      | change                                              | path                 |
      | the version is 2                                     | /version             |
      | the version is written as text                       | /version             |
      | an unknown top-level field is added                  | /extra               |
      | an unknown field is added to the second operation    | /operations/1/verb   |

  # Behaviour: SPC-012
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: An operation that lacks a required part is refused at the part it lacks
    Given a connector spec in which <change>
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path <path>

    Examples:
      | change                                                        | path                       |
      | the second operation has no summary                           | /operations/1/summary      |
      | the fourth operation has no example                           | /operations/3/examples     |
      | the spec has no operation at all                              | /operations                |
      | a header of the second operation is a number, not text       | /operations/1/headers/X-Tenant |

  # Behaviour: SPC-013
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A value outside its closed list is refused at the field that holds it
    Given a connector spec in which <change>
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path <path>

    Examples:
      | change                                                  | path                                      |
      | a response field uses a converter the spec does not know | /operations/1/response/fields/body/convert |
      | an enum parameter has no values                         | /operations/0/params/1/values             |
      | a parameter kind is not in the operation catalog        | /operations/0/params/1/type               |
      | an example reply has a status of 99, below 100          | /operations/3/examples/0/response/status  |

  # Behaviour: SPC-014
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: An auth block whose fields do not match its type is refused
    Given a connector spec whose auth is <auth>
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path <path>

    Examples:
      | auth                                            | path               |
      | basic with no identity variable                 | /auth/identity_env |
      | bearer that also names an identity variable     | /auth              |

  # Behaviour: SPC-015
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A paged read without a stop rule, or with a cap above the limit, is refused
    Given a connector spec in which <change>
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path <path>

    Examples:
      | change                                                      | path                                   |
      | the first operation pages with no maximum page count        | /operations/0/pagination/max_pages     |
      | a limit asks for 21 pages, above the cap of 20              | /limits/max_pages                      |
      | a limit asks for a timeout above 30 seconds                 | /limits/timeout_seconds                |

  # Behaviour: SPC-012
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A command-line name that is not a lower-case slug is refused at the operation
    Given a connector spec whose first operation names a command-line name of "Not A Slug"
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/0/cli

  # Behaviour: SPC-016
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The schema of the spec is a JSON Schema document of draft 2020-12
    Given the module is loaded
    When the caller reads the schema of the spec
    Then the schema declares the draft 2020-12 dialect of JSON Schema

  # Behaviour: SPC-016
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The schema of the spec fixes the version to the number 1
    Given the module is loaded
    When the caller reads the schema of the spec
    Then the schema accepts version 1 and no other version

  # Behaviour: SPC-016
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: Changing the bytes of one schema copy does not change what the next read returns
    Given the caller has read the schema of the spec
    When the caller overwrites the bytes it holds
    And the caller reads the schema of the spec again
    Then the second read returns the original schema

  # Behaviour: SPC-016
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The schema lists the parameter kinds of the operation catalog
    Given the module is loaded
    When the caller reads the schema of the spec
    Then the parameter kinds it allows are the kinds of the operation catalog
