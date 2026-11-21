Feature: Reads and mapped fields
  A read sends one declared request and maps the reply to the fields the spec declares. A field holds one scalar value,
  one closed-format value, a list of strings, a nested record of scalars, or a flag map. A reply that does not fit its
  declared shape fails the read with no partial record, and a message names the path of the field and never its value.
  A call with an argument the spec does not declare, or with a bad path or parameter value, is refused before any request.

  # Behaviour: ENG-144
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A list read sends one declared request and maps each declared field
    Given the validated example spec and a replay of its list example
    When the caller runs the list read
    Then one request is sent with the declared method, path, and header
    And the result holds each declared field of the list items

  # Behaviour: ENG-145
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: An object read maps each declared field of its canned reply
    Given the validated example spec and a replay of its single-document example
    When the caller runs the object read
    Then the result holds each declared field of the object

  # Behaviour: ENG-146
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: An object read refuses a reply that is a list
    Given a validated object read whose reply is a list
    When the caller runs the object read
    Then the read fails with an unexpected response and returns no record

  # Behaviour: ENG-147
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A declared scalar field that holds an object fails the whole read
    Given a validated read whose declared field holds an object in the reply
    When the caller runs the read
    Then the read fails and returns no record

  # Behaviour: ENG-148
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A first-item read maps only item zero
    Given a validated first-item read and a reply with two items
    When the caller runs the read
    Then the result holds the fields of the first item only

  # Behaviour: ENG-149
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A first-item read of an empty list fails with not found and no partial record
    Given a validated first-item read and a reply with an empty list
    When the caller runs the read
    Then the read fails with the not-found code of its kind and returns no record

  # Behaviour: ENG-150
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario Outline: A first-item read whose first pointer holds a value that is not a list fails as unexpected
    Given a validated first-item read whose first pointer holds <value>
    When the caller runs the read
    Then the read fails with an unexpected response

    Examples:
      | value         |
      | an object     |
      | a text        |

  # Behaviour: ENG-151
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario Outline: A required reply field that is <state> fails the read with no partial record
    Given a validated read whose required field is <state> in the reply
    When the caller runs the read
    Then the read fails with an unexpected response and returns no record

    Examples:
      | state   |
      | missing |
      | null    |
      | empty   |

  # Behaviour: ENG-152
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: One list item with a missing required value fails the whole list read
    Given a validated list read whose second item lacks a required value
    When the caller runs the read
    Then the read fails with an unexpected response and returns no partial list

  # Behaviour: ENG-153
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A reply value that its closed format refuses fails the read as unexpected
    Given a validated read whose id field names the format id and the reply holds a value of another shape
    When the caller runs the read
    Then the read fails with an unexpected response

  # Behaviour: ENG-154
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario Outline: A reply value of the right shape for a closed format name maps to its field
    Given a validated read whose field names the format <format>
    When the caller runs the read with a reply value of the <format> form
    Then the field holds the value

    Examples:
      | format   |
      | id       |
      | uuid     |
      | share-id |

  # Behaviour: ENG-155
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A refusal of a reply value names the field path and never the value
    Given a validated read whose reply holds a value that a field refuses
    When the caller runs the read
    Then the error names the field path
    And the error does not show the value of the reply

  # Behaviour: ENG-156
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A strings field maps a reply list of strings in reply order
    Given a validated read whose field is a strings field and the reply holds a list of strings
    When the caller runs the read
    Then the field holds the strings in reply order

  # Behaviour: ENG-157
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario Outline: A strings field whose reply is not a list of strings fails the read
    Given a validated read whose strings field holds <reply>
    When the caller runs the read
    Then the read fails with an unexpected response and returns no record

    Examples:
      | reply                                |
      | a list with a number in it           |
      | a value that is not a list           |

  # Behaviour: ENG-158
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A missing optional strings field maps to null and the record keeps every declared field
    Given a validated read whose optional strings field is missing from the reply
    When the caller runs the read
    Then the strings field is null
    And the record holds every declared field

  # Behaviour: ENG-159
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A nested record maps its declared sub-fields and withholds an undeclared one
    Given a validated read with a nested record field and a reply that holds an extra e-mail address in the record
    When the caller runs the read
    Then the nested record holds the declared sub-fields only
    And the e-mail address reaches no output, no error, and no log

  # Behaviour: ENG-160
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A nested sub-field whose pointer reaches a whole object fails the read and names its dotted path
    Given a validated nested record whose sub-field points at an object in the reply
    When the caller runs the read
    Then the read fails with an unexpected response naming the dotted path of the sub-field
    And no record is returned

  # Behaviour: ENG-161
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A missing required nested sub-field fails the read and names its dotted path
    Given a validated nested record whose required sub-field is missing in the reply
    When the caller runs the read
    Then the read fails with an unexpected response naming the dotted path of the sub-field

  # Behaviour: ENG-162
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: An open flag map keeps every key in sorted order with its flag value
    Given a validated read whose field is an open flag map and a reply with three keys
    When the caller runs the read
    Then the field holds the three keys in sorted order with their flag values

  # Behaviour: ENG-163
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: An open flag map with an empty reply holds an empty record and no failure
    Given a validated read whose field is an open flag map and a reply with an empty object
    When the caller runs the read
    Then the field holds an empty record and the read succeeds

  # Behaviour: ENG-164
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario Outline: An open flag map that holds a value which is not a flag fails the read naming the dotted path
    Given a validated read whose flag map holds <value> under one key
    When the caller runs the read
    Then the read fails with an unexpected response naming the dotted path of the key
    And the value is not kept in the record

    Examples:
      | value        |
      | a number     |
      | a text       |

  # Behaviour: ENG-165
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: An absent optional flag map maps to an empty record
    Given a validated read with an optional flag map and a reply that lacks it
    When the caller runs the read
    Then the flag map is an empty record

  # Behaviour: ENG-166
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: An absent required flag map fails the read naming the field
    Given a validated read with a required flag map and a reply that lacks it
    When the caller runs the read
    Then the read fails with an unexpected response naming the field

  # Behaviour: ENG-167
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A flag map whose fallback pointer holds the second reply shape maps that shape
    Given a validated read whose flag map has a fallback pointer and a reply that holds only the fallback object
    When the caller runs the read
    Then the flag map holds the flags of the fallback object

  # Behaviour: ENG-168
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: An undeclared argument is refused before any request
    Given the validated example spec
    When the caller runs a read with an argument the spec does not declare
    Then the call is refused and no request is sent

  # Behaviour: ENG-169
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario Outline: A call that leaves out a required path argument or gives one of the wrong kind is refused before any request
    Given the validated example spec and a call with <problem>
    When the caller runs the read
    Then the call is refused and no request is sent

    Examples:
      | problem                                   |
      | a missing required path argument          |
      | a path argument of the wrong kind         |

  # Behaviour: ENG-170
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario Outline: An issue key that is not in the canonical form is refused before any request
    Given a read whose path takes an issue key and the caller passes <key>
    When the caller runs the read
    Then the call fails with invalid document ID and no request is sent
    And the error does not show the key

    Examples:
      | key                     |
      | a lower-case key        |
      | a key with a separator  |

  # Behaviour: ENG-171
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: A canonical issue key fills the path and the replay answers the read
    Given a read whose path takes an issue key and the caller passes a canonical key
    When the caller runs the read
    Then the replay answers the rendered request and the read succeeds

  # Behaviour: ENG-172
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario Outline: A parameter value that the catalog refuses is refused before any request
    Given a read with an <kind> argument of the value <value>
    When the caller runs the read
    Then the call is refused by the catalog and no request is sent

    Examples:
      | kind                | value                                      |
      | icon                | a blank icon                               |
      | URL list            | a redirect address that is not HTTPS       |
      | event list          | an event name that is neither a star nor dotted |

  # Behaviour: ENG-173
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  # Exemption(integration): the rule is decided in memory from the spec and a canned reply, with no file, process, or network boundary; alternative-proof: retriever-engine:test:unit
  @integration-exempt
  Scenario: Icon, URL list, and event list values the catalog accepts reach the replay stub
    Given a read with a valid icon, URL list, and event list argument
    When the caller runs the read
    Then the replay answers the rendered request and the read succeeds
