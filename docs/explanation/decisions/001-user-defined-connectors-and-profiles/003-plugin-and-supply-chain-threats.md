---
title: Threats from plugins and the supply chain
description: Threats from the plugin, the only code-execution path, and from the way specs reach a user.
category: explanation
tags:
  - explanation
  - security
  - threat-model
  - planned
---

# Threats from plugins and the supply chain

A plugin is the only path on which code runs (boundary B4). Specs run no code. These threats belong to
[ADR 001](../001-user-defined-connectors-and-profiles.md) and follow the [scope](./001-threat-model.md).

## Plugin threats

**P1. Altered or unknown code.** The program at the plugin path is not the one the user approved.

- Control: the catalog gives an absolute path and a SHA-256 pin. The host copies the file into a private folder, hashes
  the copy, and runs the copy, so a swap after the check changes nothing.
- Proof: a plugin whose bytes differ from the pin never starts and fails with `retriever.connector.plugin_pin_mismatch`.
  An integration scenario replaces the file between the hash and the run, and the copy still runs.

**P2. Plugin runs without consent.** A spec or a default enables a plugin.

- Control: plugins are off by default. Only a catalog source can enable one, and only for that source. The spec schema
  has no plugin field.
- Proof: a catalog with no plugin block starts no process, and a spec that names a plugin fails validation.

**P3. Credential theft by environment.** The plugin reads credentials that are not its own.

- Control: the child gets an empty environment plus its own credential variable and, if declared, its identity
  variable. It starts in an empty private folder as its working directory.
- Proof: an integration scenario runs a plugin that prints its environment, and only those variables appear.

**P4. Runaway plugin.** The plugin hangs, floods output, or spawns work.

- Control: each call has a time limit, each frame is at most 1 MiB, and one process serves one source. On a timeout the
  host kills the process group and exits `124`.
- Proof: a plugin that never answers is killed at the limit, and a frame of 1 MiB plus one byte fails the call.

**P5. Protocol abuse.** The plugin answers out of turn, sends a notification, or asks the host for something.

- Control: the host accepts JSON-RPC 2.0 replies with the id it sent, and no other message. The host offers no callback,
  file, or network service.
- Proof: a reply with the wrong id, a notification, and a malformed frame each fail with
  `retriever.connector.plugin_protocol`.

**P6. Plugin output leaks a secret.** The plugin echoes a credential in a reply or on standard error.

- Control: the host redacts the plugin's credential values from replies and from captured standard error, and caps
  standard error at 64 KiB.
- Proof: a plugin that prints its own credential in both places produces output that shows `[REDACTED]`.

**Accepted risk.** The host does not sandbox a plugin. The plugin may use the network and the user's files. The pin and
the opt-in are the controls. Review a plugin as you review any program you run.

## Supply chain

**C1. A hostile third-party spec.** A user copies a spec from elsewhere.

- Control: a spec cannot run code (S1 and S2), cannot reach another host (S4 and S8), and cannot raise a limit (S3).
  `connector validate` prints each operation for review. Its remaining power is to send the user's token to the
  catalog host in requests the user did not expect, which the token's rights bound. retriever does not stop a write
  the token allows.
- Proof: the review output of a fixture spec lists every operation with label, method, and path.

**C2. A spec changed after review.** The file differs from the one the user read.

- Control: a catalog source may carry `spec_sha256`. The loader refuses a spec whose bytes differ.
- Proof: a changed spec fails with `retriever.connector.spec_pin_mismatch`, and no request goes out.

**C3. Private specs in the public repository.** A user spec or a real host lands in a commit.

- Control: user specs live outside the repository. A repository test fails if a bundled spec names any host other than
  `example.test`, `example.invalid`, or `example.com`. The public-safety gates still screen every commit.
- Proof: a bundled-spec fixture with another host fails that test.
