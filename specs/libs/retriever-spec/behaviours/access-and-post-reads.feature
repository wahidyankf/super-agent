Feature: Access labels, methods, and the POST read marker
  Each operation pairs an access label (read, write, or admin) with an HTTP method. The access matrix decides which pairs
  the spec may declare. A read sent over POST is a read only when the spec sets the read-over-POST marker, and its body
  takes values only from declared body parameters. The engine and the review both rely on these answers.

  # Behaviour: SPC-020
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: The access matrix accepts a label, method, and marker pair only when it allows the pair
    Given a connector spec with one operation labelled <access>, sent with method <method>, and marker <marker>
    When the caller checks the pair against the access matrix
    And the caller validates a spec that holds the operation
    Then the matrix answers <verdict> for the pair
    And the spec is <verdict>

    Examples:
      | access | method | marker   | verdict  |
      | read   | GET    | no       | accepted |
      | read   | HEAD   | no       | accepted |
      | read   | POST   | yes      | accepted |
      | read   | POST   | no       | refused  |
      | read   | PUT    | no       | refused  |
      | read   | GET    | yes      | refused  |
      | write  | POST   | no       | accepted |
      | write  | PUT    | no       | accepted |
      | write  | PATCH  | no       | accepted |
      | write  | DELETE | no       | accepted |
      | admin  | DELETE | no       | accepted |
      | admin  | GET    | no       | refused  |
      | owner  | GET    | no       | refused  |

  # Behaviour: SPC-022
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: An access label that the spec does not define is refused at the label
    Given a connector spec whose first operation is labelled owner
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/0/access

  # Behaviour: SPC-022
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A delete that the spec labels as a read is refused at the method
    Given a connector spec whose delete operation is labelled read
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/3/method

  # Behaviour: SPC-021
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A POST read without the read-over-POST marker is refused at the operation
    Given a connector spec whose POST read omits the read-over-POST marker
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/4/read_over_post

  # Behaviour: SPC-021
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The read-over-POST marker on a GET operation is refused at the marker
    Given a connector spec whose GET read sets the read-over-POST marker
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/0/read_over_post

  # Behaviour: SPC-021
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A POST read with no body is refused at the body
    Given a connector spec whose POST read declares no body
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/4/body

  # Behaviour: SPC-021
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A POST read whose body slot names a path parameter is refused at the slot
    Given a connector spec whose POST read body takes a path parameter in its collection field
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/4/body/collectionId

  # Behaviour: SPC-021
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A POST read body that holds a slot inside a longer text is refused
    Given a connector spec whose POST read body holds the text "id-" followed by a body slot
    When the caller validates the spec document
    Then the spec is refused

  # Behaviour: SPC-021
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A POST read body is accepted up to eight levels of nesting and refused past eight
    Given a connector spec whose POST read body nests <levels> levels
    When the caller validates the spec document
    Then the spec is <verdict>

    Examples:
      | levels | verdict  |
      | 8      | accepted |
      | 9      | refused  |

  # Behaviour: SPC-021
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A POST read may not set its own content type, in any letter case
    Given a connector spec whose POST read sets the header <header> to text/plain
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path <path>

    Examples:
      | header       | path                                 |
      | Content-Type | /operations/4/headers/Content-Type   |
      | content-type | /operations/4/headers/content-type   |

  # Behaviour: SPC-021
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A POST read that names a resolve on its parameter is refused at the parameter
    Given a connector spec whose POST read declares a parameter with a resolve
    When the caller validates the spec document
    Then the spec is refused
    And the finding names the path /operations/4/params/0/resolve

  # Behaviour: SPC-021
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The limits of a POST read body are 64 KiB and eight levels of nesting
    Given the module is loaded
    When the caller reads the POST body limits
    Then the size limit is 65536 bytes
    And the depth limit is 8 levels

  # Behaviour: SPC-021
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-spec:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A list in a body adds one level of depth, as an object does
    Given a body that is a list holding an object whose field is a list of one text value
    When the caller measures the depth of the body
    Then the depth is 3 levels
