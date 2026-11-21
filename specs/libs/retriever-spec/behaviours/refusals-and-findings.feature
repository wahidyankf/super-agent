Feature: Refusals and findings
  A refused spec returns no model. It returns findings, one for each broken rule, in path order. Each finding carries
  the stable code and the path of the field at fault, and it never shows a value of the spec. A caller tests the
  refusal with the sentinel error, never with the text.

  # Behaviour: SPC-002
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A spec that breaks one rule is refused with no model and a finding at the broken field
    Given a connector spec that breaks the rule of <case>
    When the caller validates the spec document
    Then the caller gets no model of the spec
    And the finding names the path <path>
    And the finding carries the code retriever.connector.spec_invalid

    Examples:
      | case                                                 | path                                |
      | a summary with braces                                | /summary                            |
      | an operation with no example                         | /operations/3/examples              |
      | an unknown parameter kind                            | /operations/0/params/1/type         |
      | an operation id outside its kind                     | /operations/0/id                    |
      | a path slot with no declared parameter               | /operations/1/path                  |

  # Behaviour: SPC-002
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A spec that breaks two rules lists both findings in path order
    Given a connector spec whose summary has braces and whose fourth operation has no example
    When the caller validates the spec document
    Then the caller gets no model of the spec
    And the findings are listed in path order, so /operations/3/examples comes before /summary

  # Behaviour: SPC-002
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The error of a refused spec matches the spec error sentinel
    Given a connector spec that breaks one rule
    When the caller validates the spec document
    Then the error of the result matches the spec error sentinel

  # Behaviour: SPC-003
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A document over the size cap is refused with a finding that names the whole document
    Given a document of more than 256 KiB
    When the caller validates the spec document
    Then the finding names the whole document, shown as "(document)"

  # Behaviour: SPC-004
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A field name that is not plain text is withheld from the finding
    Given a connector spec with an unknown field whose name is a long random token
    When the caller validates the spec document
    Then the spec is refused
    And the finding shows the placeholder "(name withheld)" in place of the field name
    And no finding shows the token

  # Behaviour: SPC-004
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A plain unknown field name is shown in its finding path
    Given a connector spec with an extra top-level field named extra
    When the caller validates the spec document
    Then the finding names the path /extra

  # Behaviour: SPC-004
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A value shaped like a credential is refused and is never shown in a finding
    Given a connector spec that puts <shape> into <place>
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path <path>
    And no finding shows the credential-shaped value

    Examples:
      | shape                                 | place                               | path                                              |
      | a token-shaped string                 | the summary                         | /summary                                          |
      | a bearer-shaped string                | the title of an example reply body  | /operations/1/examples/0/response/body/title      |
      | an Authorization header               | the headers of an example reply     | /operations/1/examples/0/response/headers/Authorization |

  # Behaviour: SPC-005
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A finding prints as one line with its code, its path, and its rule
    Given a connector spec that breaks the rule of a summary with braces
    When the caller validates the spec document
    Then the caller prints the finding as one line
    And the line reads retriever.connector.spec_invalid, then the path /summary, then the broken rule

  # Behaviour: SPC-005
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A finding for the whole document prints the document in place of a path
    Given a finding that names the whole document and breaks the size rule
    When the caller prints the finding as one line
    Then the line reads retriever.connector.spec_invalid, then (document), then the broken rule

  # Behaviour: SPC-049
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The error of a refused spec lists each finding on its own line and holds no value of the spec
    Given a connector spec that breaks two rules
    When the caller reads the error text of the refused spec
    Then the error text has one line for each finding
    And the error text shows no value of the spec
