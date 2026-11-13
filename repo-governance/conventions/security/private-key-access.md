---
description: >-
  Forbids agents from opening, printing, copying, or summarising private keys and other secret material, and names the
  only path for work that needs a key.
when_to_use: >-
  Use before any SSH connection, key-based signing, key generation, or task that touches a credential file.
---

# Private Key Access

Agents do not read private keys or other secret material. The rule covers every key type, including SSH, signing, and
TLS keys, and every tool that can show a file's contents.

## Rules

1. Do not open, print, copy, or summarise a private key file. Do not list its contents through any tool.
2. Do not place a key path in a command that echoes into a transcript or a log.
3. A diagnostic names a key by its role, such as "the signing key", never by its content or its fingerprint.
4. Work that needs a connection runs through the owner's own configuration on the owner's machine. The owner starts the
   connection, or the program uses an agent socket that holds the key without exposing it.
5. A key that may have been exposed is rotated at once. Follow [No Secrets in Tracked
   Files](no-secrets-in-tracked-files.md).

## Remote Work

Remote compute goes only through the [remote
dispatch](../../development/practice/resource-aware-development/remote-dispatch.md) rule. Its binding is found at
`REMOTE_DISPATCH_HOME`, and no dispatch uses a key that the binding does not name.
