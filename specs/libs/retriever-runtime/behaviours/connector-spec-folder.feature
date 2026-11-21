# The connector spec folder reads the YAML spec files of one folder in name order, with the SHA-256 that a catalog pin is
# checked against. Other entries are skipped, and a spec file that is unsafe or too large refuses the whole read.
Feature: Connector spec folder
  A caller reads the connector spec files of a folder. Each returned file carries its name, its bytes, and its digest.
  The folder reader parses nothing and names no path in a missing or unreadable folder error.

  # Behaviour: RUN-075
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: Every YAML spec file of the folder is read in name order with its digest
    Given a folder with the spec files "b.yml" and "a.yaml"
    When the caller reads the connector spec folder
    Then the files are returned in the order "a.yaml", "b.yml" and each holds its bytes and its SHA-256 digest

  # Behaviour: RUN-076
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: Entries that are not YAML spec files are skipped
    Given a folder with the spec file "a.yml" and the entries "notes.txt" and "readme"
    When the caller reads the connector spec folder
    Then only "a.yml" is returned

  # Behaviour: RUN-077
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: An empty folder has no spec files
    Given an empty connector spec folder
    When the caller reads the connector spec folder
    Then no files are returned and no error is returned

  # Behaviour: RUN-078
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A spec file that is a symbolic link or a folder refuses the whole read
    Given a folder with the spec entry "bad.yml" that is a "<kind>"
    When the caller reads the connector spec folder
    Then the call returns the error "connector spec file bad.yml: must be a regular file, not a symlink or a folder" and no files

    Examples:
      | kind          |
      | symbolic link |
      | folder        |

  # Behaviour: RUN-079
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A spec file over the size limit refuses the read and names the file
    Given a folder with the spec file "big.yml" of more than the size limit
    When the caller reads the connector spec folder
    Then the call returns the error "connector spec file big.yml: is larger than the size limit" and no files

  # Behaviour: RUN-079
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A spec file of exactly the size limit is read
    Given a folder with the spec file "edge.yml" of exactly the size limit
    When the caller reads the connector spec folder
    Then the file "edge.yml" is returned

  # Behaviour: RUN-080
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A folder that does not exist returns the no-folder error without a path
    Given a connector spec folder that does not exist
    When the caller reads the connector spec folder
    Then the call returns the error "the connectors folder does not exist" and the error holds no path

  # Behaviour: RUN-081
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A folder that cannot be read returns the unreadable error, no path
    Given a connector spec folder that exists and cannot be read
    When the caller reads the connector spec folder
    Then the call returns the error "could not read the connectors folder" and the error holds no path

  # Behaviour: RUN-082
  # Exemption(integration): the open and read faults of one file cannot be made on a real disk on demand, so the unit binding injects them; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A spec file that cannot be opened or read refuses the whole read
    Given a folder with the spec file "locked.yml" that "<fault>"
    When the caller reads the connector spec folder
    Then the call returns the error "connector spec file locked.yml: could not be read" and no files

    Examples:
      | fault                         |
      | cannot be opened              |
      | fails while it is being read  |
