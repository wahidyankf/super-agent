Feature: Plain-text review of a valid spec
  Before any engine runs a spec, a person reads what it can do. The review is plain text. It names the endpoint, the
  authentication, the limits, and each operation with its access label, method, path, writes, and surfaces. It names
  each variable by its name and never shows a value. It is the same text for the same spec.

  # Behaviour: SPC-039
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The review of the bundled example opens with the name, the summary, the endpoint, and the auth
    Given the bundled example connector spec
    When the caller reviews the valid spec
    Then the first line of the review reads "connector spec example-wiki, version 1"
    And the review has a line for the endpoint that names the base path /v1 and hosts ending in .example.test
    And the review has a line for the auth that names bearer and the variable EXAMPLE_WIKI_API_KEY

  # Behaviour: SPC-039
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The review lists each operation with its label, method, path, writes flag, and surfaces
    Given the bundled example connector spec
    When the caller reviews the valid spec
    Then the review has the line "1. example-wiki.docs.list: read, GET /v1/docs, writes no, cli on, mcp on"
    And the review has the line "3. example-wiki.docs.create: write, POST /v1/docs, writes yes, cli on, mcp on"
    And the review lists four operations

  # Behaviour: SPC-039
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The review of a spec with a POST read adds a block that names its body slots
    Given a valid connector spec with one POST read whose body takes the slot collection_id
    When the caller reviews the valid spec
    Then the review has a POST reads block that names the operation, its POST path, and the slot collection_id

  # Behaviour: SPC-039
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The review of a spec with no POST read has no POST reads block
    Given the bundled example connector spec
    When the caller reviews the valid spec
    Then the review has no POST reads block

  # Behaviour: SPC-040
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The review of a spec shows no value that the spec holds
    Given a valid connector spec whose example replies and names hold distinctive text values
    When the caller reviews the valid spec
    Then the review shows no value of the spec
    And the review names each variable by its name

  # Behaviour: SPC-040
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The review of the same spec is the same text on every call
    Given the bundled example connector spec
    When the caller reviews the valid spec twice
    Then the two reviews are the same text

  # Behaviour: SPC-041
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The review states a limit the spec does not set as a cap
    Given the bundled example connector spec, which sets a timeout but no reply size
    When the caller reviews the valid spec
    Then the review has a limits line that reads "timeout 20s" for the timeout the spec sets
    And the review has a limits line that reads "reply 10485760 bytes (cap)" for the reply size it does not set

  # Behaviour: SPC-041
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: The review names each reply shape and rule that an operation uses
    Given a valid connector spec whose operation uses <shape>
    When the caller reviews the valid spec
    Then the review has a block that names the operation and its <shape>

    Examples:
      | shape                          |
      | tree read                      |
      | merged list                    |
      | first-item read                |
      | document layout                |
      | nested record                  |
      | open-key flag map              |
      | built IDs                      |
      | query key alias               |
      | key echo                       |
      | fallback read                  |
      | issue key and page end         |
      | required reply fields          |
      | body paging                    |
      | POST read                      |
      | bundled mapping                |

  # Behaviour: SPC-041
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The review states the reply size limit the spec sets
    Given a valid connector spec that sets its reply size limit to 1048576 bytes
    When the caller reviews the valid spec
    Then the review states the reply size as 1048576 bytes
