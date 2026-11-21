# The Markdown export sink writes one file for each document into one folder. It refuses a name that is not one plain
# Markdown file name before it touches the disk, and a failed write leaves no temporary file behind.
Feature: Markdown export sink
  A caller writes an export file into the export folder and receives the path of the file.
  A refused name or a failed step leaves the folder as it was.

  # Behaviour: RUN-069
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A write stores the file under the folder and returns its path
    Given an export folder "/exports" and the file name "doc-1.md" with the bytes "# Notes"
    When the caller writes the export
    Then the file "/exports/doc-1.md" holds the bytes "# Notes" and the path "/exports/doc-1.md" is returned

  # Behaviour: RUN-069
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A write creates the folder when it is missing
    Given an export folder "/exports/new" that does not exist and the file name "doc-1.md"
    When the caller writes the export
    Then the folder is created and the file holds the bytes

  # Behaviour: RUN-070
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A write replaces an existing file and leaves no temporary file
    Given an export folder "/exports" with the file "doc-1.md" that holds "old"
    When the caller writes the export of "doc-1.md" with the bytes "new"
    Then the file "doc-1.md" holds "new" and no temporary file is left in the folder

  # Behaviour: RUN-071
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: A file name that is not one plain .md name is refused before any folder
    Given an export folder "/exports" and the file name "<name>"
    When the caller writes the export
    Then the call returns the error "the export file name must be one plain name ending in .md" and nothing is written

    Examples:
      | name        |
      | ../doc.md   |
      | sub/doc.md  |
      | doc.txt     |
      | .md         |
      | .           |
      | ..          |
      |             |

  # Behaviour: RUN-072
  # Exemption(integration): the refusal happens before the folder is touched, and the unit binding proves that no folder call is made; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A sink with no folder refuses to write
    Given a sink with no export folder
    When the caller writes the export of "doc-1.md"
    Then the call returns the error "no export folder was given" and nothing is written

  # Behaviour: RUN-073
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A folder that cannot be created returns the folder error
    Given an export folder that cannot be created
    When the caller writes the export of "doc-1.md"
    Then the call returns an error that starts with "creating the export folder" and nothing is written

  # Behaviour: RUN-074
  # Exemption(integration): a real folder cannot make one named step fail on demand, so the unit binding injects each failure; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A failed step removes the temporary file and places no export
    Given an export folder "/exports" where the step "<step>" fails
    When the caller writes the export of "doc-1.md"
    Then the call returns an error and the folder holds no file "doc-1.md" and no temporary file

    Examples:
      | step                  |
      | create the temporary file |
      | write the bytes       |
      | set the mode          |
      | close the file        |
      | rename into place     |
