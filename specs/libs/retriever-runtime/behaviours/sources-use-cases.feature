# The source use cases run one operation on one source. Each one checks the call before it asks a source, so a bad call
# makes no request. Reads, writes, exports, and the source list each follow their own rules.
Feature: Source use cases
  A caller runs a read, a write, an export, or a list against a source of a kind.
  A bad call is refused before any source is asked, and a reply of the wrong shape is never returned as a result.

  # Behaviour: RUN-001
  # Exemption(integration): the use case runs over the catalog and the gateway ports, and it crosses no file, process, or socket boundary of its own; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A read refuses a bad argument before any source is asked
    Given the read of "outline documents get" with the arguments "<arguments>"
    When the caller runs the read
    Then the call returns the error "<message>" and no source is asked

    Examples:
      | arguments                                | message                             |
      | bogus set to "x"                         | unknown argument                    |
      | no document                              | a required argument is missing      |
      | document "doc 1"                         | invalid document ID                 |

  # Behaviour: RUN-001
  # Exemption(integration): the use case runs over the catalog and the gateway ports, and it crosses no file, process, or socket boundary of its own; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A read with an invalid source ID refuses before any source is asked
    Given the read of "outline documents get" with the document "doc-1" and the source "Outline"
    When the caller runs the read
    Then the call returns the error "invalid source ID" and no source is asked

  # Behaviour: RUN-002
  # Exemption(integration): the use case runs over the catalog and the gateway ports, and it crosses no file, process, or socket boundary of its own; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A read with a named source asks that source only
    Given the outline sources "docs" and "wiki" and the read of "outline documents get" with the document "doc-1" and the source "wiki"
    When the caller runs the read
    Then only the source "wiki" is asked and the document is returned

  # Behaviour: RUN-003
  # Exemption(integration): the use case runs over the catalog and the gateway ports, and it crosses no file, process, or socket boundary of its own; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A read with no source uses the only source of the kind, or refuses
    Given the sources "<sources>" of the kind outline and the read of "outline collections list" with no source
    When the caller runs the read
    Then the result is "<result>"

    Examples:
      | sources      | result                                                |
      | docs         | the source "docs" is asked                            |
      | docs, wiki   | the error "more than one outline source exists: docs, wiki" |
      | none         | the error "no outline source is configured"           |

  # Behaviour: RUN-004
  # Exemption(integration): the use case runs over the catalog and the gateway ports, and it crosses no file, process, or socket boundary of its own; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A read reply of the wrong shape returns the unexpected response error
    Given the outline source "docs" whose reply to the read does not have the shape of its operation
    When the caller runs the read
    Then the call returns the error "unexpected response from Outline" and no result

  # Behaviour: RUN-005
  # Exemption(integration): the use case runs over the catalog and the gateway ports, and it crosses no file, process, or socket boundary of its own; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A write of a read operation is refused
    Given the read operation "outline documents get" and the source "docs"
    When the caller runs the write
    Then the call returns the error "the operation is not of this port's effect class" and no source is asked

  # Behaviour: RUN-006
  # Exemption(integration): the use case runs over the catalog and the gateway ports, and it crosses no file, process, or socket boundary of its own; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A write with no source is refused and names the source it needs
    Given the write of "outline documents create" with the title "Notes", the collection "550e8400-e29b-41d4-a716-446655440000", and the text "Body" and no source
    When the caller runs the write
    Then the call returns the error "this operation changes the wiki, so name its source with --source or the source argument" and no source is asked

  # Behaviour: RUN-007
  # Exemption(integration): the use case runs over the catalog and the gateway ports, and it crosses no file, process, or socket boundary of its own; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A write reply of the wrong shape reports an unknown outcome
    Given the outline source "docs" whose reply to the write does not have the shape of its operation
    When the caller runs the write with the source "docs"
    Then the call returns the error "the change may or may not have happened; check before you retry"

  # Behaviour: RUN-008
  # Exemption(integration): the use case runs over the catalog and the gateway ports, and it crosses no file, process, or socket boundary of its own; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A write with a named source returns the reply of the writer
    Given the outline source "docs" whose writer answers the write with the document "doc-1"
    When the caller runs the write with the source "docs"
    Then the result is the reply of the writer

  # Behaviour: RUN-009
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: An export writes the Markdown under a file name made from the ID
    Given the outline source "docs" with the document "<id>" titled "<title>" and the body "# Notes"
    When the caller exports the document "<id>"
    Then the file "<id>.md" holds the body "# Notes"

    Examples:
      | id    | title  |
      | doc-1 | Notes  |
      | doc-2 |        |

  # Behaviour: RUN-010
  # Exemption(integration): the use case runs over the catalog and the gateway ports, and it crosses no file, process, or socket boundary of its own; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: An export with an invalid document ID refuses before any fetch
    Given the outline source "docs"
    When the caller exports the document "doc 1"
    Then the call returns the error "invalid document ID" and no file is written

  # Behaviour: RUN-011
  # Exemption(integration): the use case runs over the catalog and the gateway ports, and it crosses no file, process, or socket boundary of its own; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: An export whose fetch fails writes nothing and keeps the fetch error
    Given the outline source "docs" whose fetch of the document "doc-1" fails with "not found"
    When the caller exports the document "doc-1"
    Then the call returns the error "not found" and no file is written

  # Behaviour: RUN-012
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: An export whose file write fails returns the write error
    Given the outline source "docs" with the document "doc-1" and a sink that cannot write
    When the caller exports the document "doc-1"
    Then the call returns the error "could not write the exported document"

  # Behaviour: RUN-013
  # Exemption(integration): the use case runs over the catalog and the gateway ports, and it crosses no file, process, or socket boundary of its own; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The source list shows ID, kind, and tags, never the endpoint
    Given the sources "docs" (outline, tags "wiki") and "tests" (kiwi, tags "tests")
    When the caller lists the sources with no filter
    Then the list holds "docs" with the kind "outline" and the tag "wiki", and "tests" with the kind "kiwi" and the tag "tests", and no endpoint or credential name

  # Behaviour: RUN-014
  # Exemption(integration): the use case runs over the catalog and the gateway ports, and it crosses no file, process, or socket boundary of its own; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: The source list filters by kind and by every tag given
    Given the sources "a" (outline, tags "wiki" and "tests"), "b" (outline, tag "wiki"), and "c" (kiwi, tags "wiki" and "tests")
    When the caller lists the sources with the kind "<kind>" and the tags "<tags>"
    Then the sources listed are "<listed>"

    Examples:
      | kind    | tags        | listed |
      | outline | wiki        | a, b   |
      | outline | wiki, tests | a      |
      | kiwi    | wiki        | c      |

  # Behaviour: RUN-015
  # Exemption(integration): the use case runs over the catalog and the gateway ports, and it crosses no file, process, or socket boundary of its own; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A named source of another kind is refused for the read
    Given the outline source "docs" and the kiwi source "tests" and the read of "outline documents get" with the source "tests"
    When the caller runs the read
    Then the call returns the error "the source is of another kind" and no source is asked

  # Behaviour: RUN-016
  # Exemption(integration): the use case runs over the catalog and the gateway ports, and it crosses no file, process, or socket boundary of its own; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A composed catalog replaces the built-in rows and runs through its reader
    Given a composed catalog with one row for the kind kiwi and the source "tests"
    When the caller runs the composed operation through the source "tests"
    Then the reader of the kind kiwi answers the operation

  # Behaviour: RUN-017
  # Exemption(integration): the use case runs over the catalog and the gateway ports, and it crosses no file, process, or socket boundary of its own; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: An operation outside the catalog is refused before any reader is asked
    Given the operation "documents.vanish" and the source "docs"
    When the caller runs the read
    Then the call returns the error "unsupported operation" and no reader is asked
