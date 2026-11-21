Feature: POST reads and query-key aliases
  A read over POST sends its one declared slot as a JSON value, with the content type the engine sets itself, and it sends
  no other body content. Its body is capped at 64 KiB and at eight levels of nesting. A query parameter may travel under a
  wire key of its own, and the engine serves the aliased paging and filters under those keys.

  # Behaviour: ENG-134
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A POST read sends its declared slot once as a JSON value with the fixed JSON content type
    Given a validated POST read with one declared body slot
    When the caller runs the read
    Then exactly one request is sent
    And its body holds the slot as a JSON value
    And its content type is application/json

  # Behaviour: ENG-135
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A POST read with an argument the spec does not declare is refused before any request
    Given a validated POST read and a call with an argument the spec does not declare
    When the caller runs the read
    Then the call is refused and no request is sent

  # Behaviour: ENG-136
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A POST read body one byte over 64 KiB is refused before any request
    Given a validated POST read whose rendered body is one byte over 64 KiB
    When the caller runs the read
    Then the call is refused and no request is sent

  # Behaviour: ENG-137
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A POST read whose body nests past eight levels is refused before any request
    Given a validated POST read whose slot fills a list that adds a ninth level of nesting
    When the caller runs the read
    Then the call is refused and no request is sent

  # Behaviour: ENG-138
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario Outline: A hand-built model that breaks the access matrix or the POST read rules is refused when the engine is built
    Given a hand-built model with <fault>
    When the caller builds the engine from the model
    Then the construction fails and no request is sent

    Examples:
      | fault                                  |
      | an access label that the matrix refuses |
      | a POST read that breaks its rules       |

  # Behaviour: ENG-139
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A query parameter with a wire key of its own travels under that key and the example still binds
    Given a validated read whose query parameter has a wire key of its own
    When the caller runs the read through the replay stub
    Then the request carries the wire key
    And the mapped record is unchanged

  # Behaviour: ENG-140
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: An aliased offset read advances the aliased offset and stops at the short page
    Given a validated offset read whose offset parameter has a wire key and a replay of two pages
    When the caller runs the read
    Then every request carries the aliased offset and the page size
    And the read stops at the short page

  # Behaviour: ENG-141
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: An aliased cursor read sends the cursor under its alias and never under the catalog name
    Given a validated cursor read whose cursor parameter has a wire key and a replay of two pages
    When the caller runs the read
    Then the second request carries the cursor under the alias

  # Behaviour: ENG-142
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: An aliased filter in a page-all read travels under its alias while the fixed paging keys keep their names
    Given a validated page-all read whose filter parameter has a wire key
    When the caller runs the read
    Then the filter travels under its alias
    And the fixed paging keys stay limit and offset

  # Behaviour: ENG-143
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A hand-built model with an alias on a path parameter is refused when the engine is built
    Given a hand-built model whose path parameter has a wire key
    When the caller builds the engine from the model
    Then the construction fails and no request is sent
