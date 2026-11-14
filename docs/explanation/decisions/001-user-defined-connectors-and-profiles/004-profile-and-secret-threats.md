---
title: Threats to profiles and secrets
description: Threats to profile separation and to credentials in logs, errors, examples, and conformance output.
category: explanation
tags:
  - explanation
  - security
  - threat-model
  - planned
---

# Threats to profiles and secrets

These threats belong to [ADR 001](../001-user-defined-connectors-and-profiles.md) and follow the
[scope](./001-threat-model.md). The profile layout is in [profile layout](../../../reference/profile-layout.md).

## Profile threats

**F1. A personal run reads a work credential.** The reverse is the same threat.

- Control: a profile has its own catalog, env file, and spec folder, and the loader reads no other profile's folder. The
  process environment is read only for names the active catalog declares. A credential name in a profile catalog must
  start with the upper-case profile name, such as `WORK_`, so two profiles cannot name one variable.
- Proof: a personal catalog that names a variable with the work prefix is refused. With both profiles present, a
  personal run makes no read of the work folder, which a recording file system proves.

**F2. A run merges two profiles.** One run lists or reads sources of both.

- Control: the profile flag takes one value. A repeated flag, a list, or a profile together with `--catalog` is a usage
  error. No command takes a profile argument.
- Proof: `--profile work --profile personal` exits `2`. An MCP tool call with a profile argument is an unknown
  argument.

**F3. A silent default.** Profiles exist, none is chosen, and the tool picks one.

- Control: with a profiles folder present and no flag or variable, the tool refuses with
  `retriever.usage.profile_required`. The flag wins over the variable. Tags never select a profile.
- Proof: a run with no choice exits `2` and reads no catalog. A run with the flag and a different variable uses the
  flag.

**F4. A long-lived server switches profile.** `mcp` or `serve` changes profile while it runs.

- Control: the profile is fixed when the process starts. Changing it needs a new process.
- Proof: the `list_sources` result names the active profile, and no tool or query changes it.

**F5. Profile name escapes its folder.** The name holds a path.

- Control: a name is a lower-case slug, resolved only inside the profiles folder. A symbolic link is refused.
- Proof: `--profile ../other` exits `2`.

**F6. State crosses profiles.** A cache or an export carries data across.

- Control: the tool keeps no on-disk cache, and the gateway cache lives in one process, which has one profile.
- Proof: the architecture test shows that no production package writes to the user configuration folder.

## Secret threats

**R1. A credential in an error or a log.** A value reaches a message, a log line, or a JSON body.

- Control: the credential type redacts itself in every formatting path, as `ApiKey` does today. Error text names a
  variable and never a value. The engine passes every message through a redactor that holds the active values.
- Proof: a stub whose reply echoes the key produces an error and a log line that show `[REDACTED]`.

**R2. A credential in an example.** A spec's canned request or response holds a real token.

- Control: validation refuses an example that holds an authorization header or a value shaped like a token, and its
  message names the field path, not the value.
- Proof: a spec with a bearer-shaped string in an example fails, and the output omits that string.

**R3. A credential in conformance output.** The report echoes a value from a reply or a request.

- Control: the report is built from fixed messages and field paths, and it passes through the same redactor.
- Proof: with a credential value planted in a canned reply, the full report shows `[REDACTED]` where it would show the
  value.
