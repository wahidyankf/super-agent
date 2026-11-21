# The config path rules decide where the catalog, the env file, and the connectors folder are read from. The rules read no
# file: the caller passes in the environment, the working directory, and the existence checks.
Feature: Configuration paths
  A caller resolves the paths of the catalog and the env file in a fixed order: the flag, then the environment variable,
  then the file in the working directory, then the user configuration folder.
  A profile folder changes the rules, and an explicit path is then refused.

  # Behaviour: RUN-057
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: The first rule that applies sets the catalog path
    Given the catalog flag "<flag>", the variable "<variable>", a sources file in the working directory "<local>", and a home folder
    When the caller resolves the paths
    Then the catalog path is "<path>"

    Examples:
      | flag              | variable         | local | path                                              |
      | /flag/sources.yml | /env/sources.yml | yes   | /flag/sources.yml                                 |
      |                   | /env/sources.yml | yes   | /env/sources.yml                                  |
      |                   |                  | yes   | /work/sources.yml                                 |
      |                   |                  | no    | /srv/retriever/home/.config/retriever/sources.yml |

  # Behaviour: RUN-058
  # Exemption(integration): the rule chooses a path from the values it is given, and the real existence checks are bound at integration by the first-rule and profile scenarios; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: An empty environment variable counts as unset
    Given the catalog variable set to "" and no flag
    When the caller resolves the paths
    Then the catalog path follows the next rule

  # Behaviour: RUN-058
  # Exemption(integration): the rule chooses a path from the values it is given, and the real existence checks are bound at integration by the first-rule and profile scenarios; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A relative flag is made absolute against the working directory
    Given the catalog flag "config/sources.yml" and the working directory "/work"
    When the caller resolves the paths
    Then the catalog path is "/work/config/sources.yml"

  # Behaviour: RUN-059
  # Exemption(integration): the rule chooses a path from the values it is given, and the real existence checks are bound at integration by the first-rule and profile scenarios; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The user folder is XDG_CONFIG_HOME when set, else HOME/.config
    Given the variable XDG_CONFIG_HOME set to "<xdg>" and the home folder "/srv/retriever/home" and no sources file in the working directory
    When the caller resolves the paths
    Then the catalog path is "<path>"

    Examples:
      | xdg  | path                                              |
      | /xdg | /xdg/retriever/sources.yml                        |
      |      | /srv/retriever/home/.config/retriever/sources.yml |

  # Behaviour: RUN-060
  # Exemption(integration): the rule chooses a path from the values it is given, and the real existence checks are bound at integration by the first-rule and profile scenarios; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: An unreadable working directory refuses the rules that need it
    Given an unreadable working directory and no catalog flag or variable
    When the caller resolves the paths
    Then the call returns the error "the environment cannot name a configuration path"

  # Behaviour: RUN-060
  # Exemption(integration): the rule chooses a path from the values it is given, and the real existence checks are bound at integration by the first-rule and profile scenarios; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: No home folder refuses only when the user folder is needed
    Given no catalog flag, no catalog variable, no sources file in the working directory, and no home folder
    When the caller resolves the paths
    Then the call returns the error "the environment cannot name a configuration path"

  # Behaviour: RUN-061
  # Exemption(integration): the rule chooses a path from the values it is given, and the real existence checks are bound at integration by the first-rule and profile scenarios; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The explicit flags resolve the paths even with no home folder
    Given the catalog flag "/flag/sources.yml" and the env file flag "/flag/.env.local" and no home folder
    When the caller resolves the paths
    Then the catalog path is "/flag/sources.yml" and the env file path is "/flag/.env.local"

  # Behaviour: RUN-062
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A profile flag wins over the profile variable
    Given a profiles folder under the user configuration folder with the profiles "work" and "home", the profile flag "work", and the variable RETRIEVER_PROFILE "home"
    When the caller resolves the paths
    Then the profile is "work" and the catalog path is in the folder of "work"

  # Behaviour: RUN-062
  # Exemption(integration): the rule chooses a path from the values it is given, and the real existence checks are bound at integration by the first-rule and profile scenarios; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The profile variable selects the profile when no flag is given
    Given a profiles folder with the profile "home" and the variable RETRIEVER_PROFILE "home" and no profile flag
    When the caller resolves the paths
    Then the profile is "home"

  # Behaviour: RUN-063
  # Exemption(integration): the rule chooses a path from the values it is given, and the real existence checks are bound at integration by the first-rule and profile scenarios; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A profiles folder with no selected profile is refused as required
    Given a profiles folder with the profiles "work" and "home" and no profile flag or variable
    When the caller resolves the paths
    Then the call returns the error "no profile was selected: pass --profile <name> or set RETRIEVER_PROFILE" with the code "retriever.usage.profile_required"

  # Behaviour: RUN-064
  # Exemption(integration): the rule chooses a path from the values it is given, and the real existence checks are bound at integration by the first-rule and profile scenarios; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario Outline: An invalid profile name is refused with the profile invalid code
    Given a profiles folder with the profile flag "<name>"
    When the caller resolves the paths
    Then the call returns the error "<message>" with the code "retriever.usage.profile_invalid"

    Examples:
      | name       | message                                                                              |
      | ../work    | invalid profile name "../work": use lowercase letters, digits, and single hyphens    |
      | Work       | invalid profile name "Work": use lowercase letters, digits, and single hyphens       |
      | team--a    | invalid profile name "team--a": use lowercase letters, digits, and single hyphens    |

  # Behaviour: RUN-065
  # Exemption(integration): the rule chooses a path from the values it is given, and the real existence checks are bound at integration by the first-rule and profile scenarios; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: An explicit catalog path is refused when profiles exist
    Given a profiles folder with the profile "work" and the catalog flag "/flag/sources.yml"
    When the caller resolves the paths
    Then the call returns the error "explicit catalog paths are refused when profiles exist: pass --profile <name>" with the code "retriever.usage.profile_invalid"

  # Behaviour: RUN-065
  # Exemption(integration): the rule chooses a path from the values it is given, and the real existence checks are bound at integration by the first-rule and profile scenarios; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The profile flag is refused when no profiles folder exists
    Given no profiles folder and the profile flag "work"
    When the caller resolves the paths
    Then the call returns the error "no profiles are configured: --profile needs a profiles folder" with the code "retriever.usage.profile_invalid"

  # Behaviour: RUN-066
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: A profile folder that is a symbolic link is refused
    Given a profiles folder with the profile "work" that is a symbolic link and the profile flag "work"
    When the caller resolves the paths
    Then the call returns the error "the profile folder is a symbolic link, which is refused" with the code "retriever.usage.profile_invalid"

  # Behaviour: RUN-067
  # Exemption(integration): the rule chooses a path from the values it is given, and the real existence checks are bound at integration by the first-rule and profile scenarios; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: The connectors folder is the flag, else the user folder's connectors
    Given the connectors flag "<flag>" and the variable XDG_CONFIG_HOME set to "/xdg"
    When the caller resolves the paths
    Then the connectors folder is "<folder>"

    Examples:
      | flag            | folder                           |
      | /custom/specs   | /custom/specs                    |
      |                 | /xdg/retriever/connectors        |

  # Behaviour: RUN-068
  # Exemption(integration): the rule chooses a path from the values it is given, and the real existence checks are bound at integration by the first-rule and profile scenarios; alternative-proof: retriever-runtime:test:unit
  @integration-exempt
  # Exemption(e2e): the library is built into the retriever binary and has no process, network, or file boundary of its own at this level; alternative-proof: retriever-cli:test:e2e
  @e2e-exempt
  Scenario: No user configuration folder leaves no default connectors folder
    Given no XDG_CONFIG_HOME and no home folder, the catalog flag "/flag/sources.yml", and the env file flag "/flag/env-file"
    When the caller resolves the connectors folder
    Then the connectors folder is empty and no error is returned
