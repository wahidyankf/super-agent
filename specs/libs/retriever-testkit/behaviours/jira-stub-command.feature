# The Jira stub command serves one Jira stub as a process. It needs an account and a token, from flags or from a config
# file, and it can record each request as one JSON line without the token.
Feature: Jira stub command
  A test runs the Jira stub command with flags or a config file. The command refuses missing credentials before it listens.

  # Behaviour: TK-072
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  Scenario: The command refuses to start without a certificate file path
    Given the Jira stub command with the account and the token but no -ca-file flag
    When the command runs
    Then the command exits with status 1 and prints "the -ca-file flag is required" and listens on no port

  # Behaviour: TK-073
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  Scenario: The command refuses to start with no account, no token, and no config file
    Given the Jira stub command with the -ca-file flag and no account, token, or config file
    When the command runs
    Then the command exits with status 1 and prints "the -email and -token flags are required, or a -config file" and listens on no port

  # Behaviour: TK-074
  # Exemption(integration): the behaviour has no file, process, or network boundary; it is a pure in-memory rule; alternative-proof: retriever-testkit:test:unit
  @integration-exempt
  Scenario: The command reports a flag it does not know on its diagnostic output
    Given the Jira stub command with the flag "-no-such-flag"
    When the command runs
    Then the command exits with status 1 and prints the diagnostic on the error output

  # Behaviour: TK-075
  Scenario: A config file supplies the account and the token in place of the flags
    Given the Jira stub command with the -ca-file flag and a config file that holds the account and the token
    When the command starts and the client calls the path "/rest/api/3/myself" with the account and the token
    Then the stub answers the call with the status 200

  # Behaviour: TK-076
  Scenario: A config file that cannot be read or parsed is refused with its error
    Given the Jira stub command with a config file that "<problem>"
    When the command runs
    Then the command exits with status 1 and prints an error that starts with "<message>"

    Examples:
      | problem             | message                  |
      | cannot be read      | read the config file     |
      | is not valid JSON   | parse the config file    |

  # Behaviour: TK-077
  Scenario: The record file holds one JSON line for each request and never the token
    Given the Jira stub command with the -record flag and the account and the token
    When the client calls the path "/rest/api/3/myself" with the account and the token
    Then the record file holds one JSON line for the request and the line does not hold the token

  # Behaviour: TK-078
  Scenario: The command writes the certificate, then announces its address
    Given the Jira stub command with the -ca-file flag and the account and the token
    When the command starts
    Then the certificate file is written and the command prints "jira stub listening on" followed by its URL

  # Behaviour: TK-079
  Scenario: A clean run exits 0, and a failed run prints its error and exits 1
    Given the Jira stub command with a clean run and then with a certificate file that cannot be written
    When the command runs both times
    Then the clean run exits with status 0 and the failed run prints its error and exits with status 1
