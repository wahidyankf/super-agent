# The public command-line contract: global --output, the one failure body, and the closed exit statuses 0, 1, 2, 124,
# and 125. The rules are in the command-line interface conventions and in the public contract record.
Feature: Command-line contract of retriever
  As a person or a harness that calls retriever
  I want every command to follow one published contract for flags, exit statuses, and failure bodies
  So that I can branch on the result without reading prose

  # No source is configured in these scenarios, so each one runs without any network or catalog.

  # Bare invocation is a usage mistake. The status is 2, and the diagnostic goes to standard error, as the help and
  # discovery convention requires. The convention permits a list of subcommands here, but does not require one, so the
  # diagnostic names the missing subcommand and the help command, and it lists nothing.
  @unit @integration @e2e
  Scenario: Running retriever with no command exits 2 and prints nothing to standard output
    When I run "retriever"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is not empty
    And the standard error contains "no valid subcommand has been specified"
    And the standard error contains "retriever --help"

  @unit @integration @e2e
  Scenario: A command group named alone exits 2 and prints nothing to standard output
    When I run "retriever outline"
    Then the exit status is 2
    And the standard output is empty
    And the standard error contains "no valid subcommand has been specified"

  @unit @integration @e2e
  Scenario: The version flag prints the tool name and the alpha banner and exits 0
    When I run "retriever --version"
    Then the exit status is 0
    And the standard output contains "retriever "
    And the standard output contains "ALPHA: expect breaking changes"
    And the standard error is empty

  @unit @integration @e2e
  Scenario: The root help lists the command groups and the output option
    When I run "retriever --help"
    Then the exit status is 0
    And the standard output contains "sources"
    And the standard output contains "outline"
    And the standard output contains "kiwi"
    And the standard output contains "jira"
    And the standard output contains "--output"

  @unit @integration @e2e
  Scenario: A command's help lists every exit status it can return
    When I run "retriever outline documents get --help"
    Then the exit status is 0
    And the standard output contains "EXIT STATUS:"
    And the standard output contains "124"
    And the standard output contains "125"

  @unit @integration @e2e
  Scenario: Help is available through the help word as well as the flag
    When I run "retriever help outline"
    Then the exit status is 0
    And the standard output contains "Outline wiki commands"

  @unit @integration @e2e
  Scenario: The help flag wins over the other arguments of the command
    When I run "retriever outline documents get --help"
    Then the exit status is 0
    And the standard output contains "USAGE: retriever outline documents get"

  @unit @integration @e2e
  Scenario Outline: The output option accepts only text and json
    When I run "retriever --output <value> serve --help"
    Then the exit status is <status>
    And the standard output is <stdout>

    Examples:
      | value | status | stdout   |
      | text  | 0      | not empty |
      | json  | 0      | not empty |
      | yaml  | 2      | empty    |
      | XML   | 2      | empty    |

  @unit @integration @e2e
  Scenario Outline: The colour option accepts only auto, always, and never
    When I run "retriever --color <value> --version"
    Then the exit status is <status>

    Examples:
      | value  | status |
      | auto   | 0      |
      | always | 0      |
      | never  | 0      |
      | sometimes | 2   |

  @unit @integration @e2e
  Scenario: An unknown argument is a usage error that keeps standard output empty
    When I run "retriever does-not-exist"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is not empty

  @unit @integration @e2e
  Scenario: A missing required argument in JSON mode is a failure body with its usage code
    When I run "retriever --output json outline documents get"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.outline.usage_missing_argument" and retryable false
    And the JSON field "schemaVersion" of the standard error is 1
    And the JSON field "command" of the standard error is "retriever outline documents get"
    And the JSON field "exitCode" of the standard error is 2

  @unit @integration @e2e
  Scenario: An unknown argument in JSON mode is a failure body, and no prose reaches standard output
    When I run "retriever --output json does-not-exist"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.usage.usage_unrecognized_argument" and retryable false

  @unit @integration @e2e
  Scenario Outline: A confirmation or preview flag that no command takes is an unrecognized argument
    When I run "<command>"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.outline.usage_unrecognized_argument" and retryable false

    Examples:
      | command                                                                                                    |
      | retriever --output json outline documents empty-trash --yes                                                |
      | retriever --output json outline documents empty-trash --confirm example-outline                            |
      | retriever --output json outline documents update d0c1d0c1-0000-4000-8000-000000000001 --title Renamed --dry-run |

  @unit @integration @e2e
  Scenario: The version flag ignores the output option and still prints plain text
    When I run "retriever --output json --version"
    Then the exit status is 0
    And the standard output contains "retriever "
    And the standard error is empty

  @unit @integration @e2e
  Scenario: A port outside the valid range is a usage error with its own code
    When I run "retriever --output json serve --port 70000"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.serve.usage_invalid_argument" and retryable false

  @unit @integration @e2e
  Scenario: The removed max-access flag is an unrecognized argument
    When I run "retriever --output json mcp --max-access read"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is a JSON error with the code "retriever.mcp.usage_unrecognized_argument" and retryable false

  @unit @integration @e2e
  Scenario: A text-mode failure keeps its message on standard error and prints nothing to standard output
    When I run "retriever outline documents get"
    Then the exit status is 2
    And the standard output is empty
    And the standard error is not empty

  @unit @integration @e2e
  Scenario: A standard output with no reader ends the command with status 141
    Given the standard output pipe has no reader
    When I run "retriever --help"
    Then the exit status is 141
