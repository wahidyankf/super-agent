Feature: Record fields of a reply
  A reply field maps one scalar value, one nested record of sub-fields, or one open-key map of flags. A nested record
  holds one to 32 sub-fields, each a scalar, and it nests one level only. The only open-key form is a map whose values are
  flags. A field may also check its value against a closed format name, and the names are closed.

  # Behaviour: SPC-029
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A nested record of scalar sub-fields is accepted
    Given a connector spec whose read maps a user record with scalar sub-fields
    When the caller validates the spec document
    Then the spec is accepted

  # Behaviour: SPC-029
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A nested record that holds another record is refused at the inner record
    Given a connector spec whose nested record holds a sub-field that is itself a record
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/4/response/fields/user/object/id/object

  # Behaviour: SPC-029
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A nested record with 33 sub-fields is refused at the record
    Given a connector spec whose nested record declares 33 sub-fields
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/4/response/fields/user/object

  # Behaviour: SPC-029
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A field that is both a record and a pointer is refused at the field
    Given a connector spec whose user field declares a record and also a pointer
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/4/response/fields/user/pointer

  # Behaviour: SPC-030
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: An open-key map with flag values is accepted
    Given a connector spec whose read maps the notification settings as an open-key map of flags
    When the caller validates the spec document
    Then the spec is accepted

  # Behaviour: SPC-030
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: An open-key map whose flag keyword is false is refused
    Given a connector spec whose open-key map sets its flag keyword to false
    When the caller validates the spec document
    Then the spec is refused

  # Behaviour: SPC-030
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: An open key set spelled any other way is refused
    Given a connector spec whose notification settings field is open through <spelling>
    When the caller validates the spec document
    Then the spec is refused

    Examples:
      | spelling                                        |
      | a star sub-field that matches any key           |
      | a schema spelling of additional properties      |
      | an undeclared value rule for the open keys      |

  # Behaviour: SPC-030
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: An open-key map that also sets a scalar keyword is refused
    Given a connector spec whose open-key map also sets a format on its field
    When the caller validates the spec document
    Then the spec is refused

  # Behaviour: SPC-034
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A format name from the closed list is accepted on a reply field
    Given a connector spec whose list read names the format <format> on its id field
    When the caller validates the spec document
    Then the spec is accepted

    Examples:
      | format   |
      | id       |
      | uuid     |
      | share-id |
      | strings  |

  # Behaviour: SPC-034
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A format name outside the closed list is refused at the format of the field
    Given a connector spec whose <field> field names the format <format>
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path <path>

    Examples:
      | field | format   | path                                      |
      | id    | pattern  | /operations/4/response/fields/id/format   |
      | id    | ID       | /operations/4/response/fields/id/format   |
      | slug  | share_id | /operations/4/response/fields/slug/format |
