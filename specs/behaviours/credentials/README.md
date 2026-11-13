# Credentials behaviours

Where each source's key comes from (the process environment, then the env file), what a missing or refused key does, and
the rule that a key is never sent to another source and never printed.

## Files

- [credentials.feature](credentials.feature): the key sources and their precedence, missing and empty keys, the Jira
  e-mail address, refused keys, and the no-leak checks.
