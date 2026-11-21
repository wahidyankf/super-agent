# The in-memory file tree is a fake disk for the file loaders. A unit test runs a loader on it without touching the disk,
# and the names it reports match the names a real folder reports.
Feature: In-memory file tree
  A test builds a file tree in memory, writes and reads its files, and lists its folders.
  The tree reports the same errors and names as a real folder for the cases the loaders check.

  # Behaviour: TK-044
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: The root folder is always present
    Given a new in-memory file tree
    When the caller describes the root folder
    Then the root folder exists and it is a folder

  # Behaviour: TK-045
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A written file reads back with the same bytes
    Given a tree with the folder "/work"
    When the caller writes the file "/work/sources.yml" with the bytes "version: 1" and reads it back
    Then the bytes read back are "version: 1"

  # Behaviour: TK-045
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A write that replaces a file keeps the new bytes only
    Given a tree with the file "/work/sources.yml" that holds "old"
    When the caller writes the file "/work/sources.yml" with the bytes "new"
    Then the file holds "new"

  # Behaviour: TK-046
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario Outline: A relative path is refused
    Given a new in-memory file tree
    When the caller reads the path "<path>"
    Then the call returns an error and no bytes

    Examples:
      | path        |
      | work/x.yml  |
      | work/../x   |

  # Behaviour: TK-047
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A missing file reads as not existing
    Given a new in-memory file tree
    When the caller reads the file "/work/missing.yml"
    Then the call returns an error that matches not exist

  # Behaviour: TK-048
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A file cannot be written where a folder is
    Given a tree with the folder "/work"
    When the caller writes the file "/work" with the bytes "x"
    Then the call returns an error and the folder is unchanged

  # Behaviour: TK-048
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A file cannot take a child
    Given a tree with the file "/work/a.yml"
    When the caller writes the file "/work/a.yml/child" with the bytes "x"
    Then the call returns an error and the file is unchanged

  # Behaviour: TK-049
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: The stat of a folder is not a regular file, and the stat of a file is
    Given a tree with the folder "/work" and the file "/work/a.yml"
    When the caller reads the link information of "/work" and of "/work/a.yml"
    Then "/work" is not a regular file and "/work/a.yml" is a regular file

  # Behaviour: TK-050
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A walk of the root returns every file under it and no folder
    Given a tree with the files "/a/one.yml" and "/a/b/two.yml" and the folder "/empty"
    When the caller walks the root
    Then the walk returns "a/one.yml" and "a/b/two.yml" only

  # Behaviour: TK-050
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A walk of a missing root returns nothing
    Given a new in-memory file tree
    When the caller walks the root "/missing"
    Then the walk returns no entries

  # Behaviour: TK-051
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A folder view names its entries relative to the folder
    Given a tree with the files "/work/a.yml" and "/work/b.yml" and the folder "/work/sub"
    When the caller reads the folder "/work" as a file system
    Then "a.yml" and "b.yml" are regular files and "sub" is a folder

  # Behaviour: TK-051
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: A folder view of a missing folder reads nothing
    Given a new in-memory file tree
    When the caller reads the folder "/missing" as a file system
    Then no entry is read

  # Behaviour: TK-052
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: Listing a folder names its direct children only
    Given a tree with the files "/work/a.yml" and "/work/sub/b.yml"
    When the caller lists the files of "/work"
    Then the list holds "a.yml" and does not hold "b.yml"

  # Behaviour: TK-052
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: Listing a missing folder is empty and listing a file is an error
    Given a tree with the file "/work/a.yml"
    When the caller lists the files of "/missing" and of "/work/a.yml"
    Then the list of "/missing" is empty and the list of "/work/a.yml" is an error

  # Behaviour: TK-053
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  # Exemption(e2e): the stub library is linked into tests and has no process boundary of its own; alternative-proof: retriever-testkit:test:unit
  @e2e-exempt
  Scenario: Two files that are written apart are not the same file
    Given a tree with the files "/work/a.yml" and "/work/b.yml"
    When the caller compares the descriptions of the two files
    Then the descriptions are not the same file
